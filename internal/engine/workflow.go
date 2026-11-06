package engine

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/alexandrmotologa/notiflux/internal/store"
	"github.com/google/uuid"
)

// TriggerOverrides specifies channel overrides for a trigger call.
type TriggerOverrides struct {
	Channels []string `json:"channels,omitempty"`
}

// TriggerRequest defines the incoming payload for triggering a workflow.
type TriggerRequest struct {
	Workflow   string            `json:"workflow"`
	Subscriber store.Subscriber  `json:"subscriber"`
	Payload    map[string]any    `json:"payload"`
	Overrides  *TriggerOverrides `json:"overrides,omitempty"`
}

// TriggerResponse provides the outcome summary of a trigger call.
type TriggerResponse struct {
	Success            bool              `json:"success"`
	NotificationID     string            `json:"notification_id,omitempty"`
	DispatchedChannels []string          `json:"dispatched_channels"`
	ScheduledChannels  []string          `json:"scheduled_channels,omitempty"`
	SkippedChannels    map[string]string `json:"skipped_channels,omitempty"`
	DigestBuffered     bool              `json:"digest_buffered,omitempty"`
	ErrorMessage       string            `json:"error,omitempty"`
}

// CompiledMessage is a fully interpolated message ready for channel dispatch.
type CompiledMessage struct {
	Channel      string
	Subject      string
	Body         string
	HTMLBody     string
	ActionURL    string
	TrackingID   string
	Subscriber   *store.Subscriber
	WorkflowID   string
	Payload      map[string]any
}

// WorkflowEngine coordinates workflow loading, recipient upsert, channel filtering, and template compilation.
type WorkflowEngine struct {
	store      *store.Store
	compiler   *Compiler
	prefEngine *PreferenceEvaluator
}

func NewWorkflowEngine(s *store.Store) *WorkflowEngine {
	return &WorkflowEngine{
		store:      s,
		compiler:   NewCompiler(),
		prefEngine: NewPreferenceEvaluator(s),
	}
}

// PrepareDispatch evaluates the trigger request, checks preferences, renders templates, and returns compiled messages.
func (we *WorkflowEngine) PrepareDispatch(req *TriggerRequest) ([]*CompiledMessage, string, map[string]string, error) {
	if req.Workflow == "" {
		return nil, "", nil, fmt.Errorf("workflow identifier is required")
	}
	if req.Subscriber.ID == "" {
		return nil, "", nil, fmt.Errorf("subscriber.id is required")
	}

	// 1. Fetch workflow definition
	wf, err := we.store.GetWorkflow(req.Workflow)
	if err != nil {
		return nil, "", nil, fmt.Errorf("error querying workflow: %w", err)
	}
	if wf == nil {
		return nil, "", nil, fmt.Errorf("workflow %q not found", req.Workflow)
	}

	// 2. Upsert subscriber information
	if err := we.store.UpsertSubscriber(&req.Subscriber); err != nil {
		return nil, "", nil, fmt.Errorf("error updating subscriber: %w", err)
	}
	sub, _ := we.store.GetSubscriber(req.Subscriber.ID)
	if sub == nil {
		sub = &req.Subscriber
	}

	// 3. Digest check: If digest is enabled and request is not marked immediate, buffer it
	isImmediate := false
	if immVal, ok := req.Payload["immediate"]; ok && isTruthy(immVal) {
		isImmediate = true
	}

	if wf.DigestEnabled && !isImmediate {
		payloadJSON, _ := json.Marshal(req.Payload)
		_ = we.store.AddDigestEntry(&store.DigestEntry{
			WorkflowID:   wf.ID,
			SubscriberID: sub.ID,
			Payload:      string(payloadJSON),
		})
		// Return empty messages and skipped indicator for digest buffering
		return nil, "", map[string]string{"digest": "buffered_in_window"}, nil
	}

	// 4. Determine active channels
	var overrideChannels []string
	if req.Overrides != nil && len(req.Overrides.Channels) > 0 {
		overrideChannels = req.Overrides.Channels
	}

	activeChannels, skipped, err := we.prefEngine.FilterActiveChannels(wf, sub, overrideChannels)
	if err != nil {
		return nil, "", nil, fmt.Errorf("error evaluating channel preferences: %w", err)
	}

	// 5. Merge data context for compiler
	contextData := make(map[string]any)
	for k, v := range req.Payload {
		contextData[k] = v
	}
	contextData["subscriber"] = map[string]any{
		"id":               sub.ID,
		"email":            sub.Email,
		"telegram_chat_id": sub.TelegramChatID,
	}

	notificationID := uuid.NewString()
	var compiledMessages []*CompiledMessage

	// Action URL lookup from payload
	rawActionURL := ""
	if urlVal, ok := req.Payload["action_url"]; ok && urlVal != nil {
		rawActionURL = fmt.Sprintf("%v", urlVal)
	}

	// Check Quiet Hours
	now := time.Now().UTC()
	inQuiet, quietEnd := IsInQuietHours(sub, now)
	isUrgent := false
	if urgVal, ok := req.Payload["urgent"]; ok && isTruthy(urgVal) {
		isUrgent = true
	}

	// 6. Compile each active channel
	for _, ch := range activeChannels {
		trackingID := "trk_" + strings.ReplaceAll(uuid.NewString(), "-", "")

		tpl, err := we.store.GetTemplateByWorkflowAndChannel(wf.ID, ch)
		if err != nil {
			return nil, "", nil, fmt.Errorf("error querying template for %s: %w", ch, err)
		}

		subjectTmpl := ""
		bodyTmpl := ""
		if tpl != nil {
			subjectTmpl = tpl.Subject
			bodyTmpl = tpl.Body
		} else {
			if tVal, ok := req.Payload["title"]; ok {
				subjectTmpl = fmt.Sprintf("%v", tVal)
			} else {
				subjectTmpl = wf.Name
			}
			if bVal, ok := req.Payload["body"]; ok {
				bodyTmpl = fmt.Sprintf("%v", bVal)
			} else {
				b, _ := json.Marshal(req.Payload)
				bodyTmpl = string(b)
			}
		}

		renderedSubject, _ := we.compiler.Render(subjectTmpl, contextData, false)
		renderedBody, _ := we.compiler.Render(bodyTmpl, contextData, ch == store.ChannelEmail)

		// Click tracking URL
		actionURL := rawActionURL

		htmlBody := ""
		if ch == store.ChannelEmail {
			rawRendered, _ := we.compiler.Render(bodyTmpl, contextData, false)
			convertedHTML := MarkdownToHTML(rawRendered)

			// Append 1x1 open tracking pixel
			openPixel := fmt.Sprintf(`<img src="/v1/o/%s.png" width="1" height="1" alt="" style="display:none;" />`, trackingID)
			convertedHTML += "\n" + openPixel

			htmlBody = WrapEmailHTML(convertedHTML, renderedSubject)
		}

		// Check if channel is delayed by a workflow step (e.g. Email after 15m if unread)
		var matchedStep *store.WorkflowStep
		for _, st := range wf.Steps {
			if st.Channel == ch && st.DelaySeconds > 0 {
				matchedStep = &st
				break
			}
		}

		// If delayed or in quiet hours, schedule for later delivery
		if matchedStep != nil || (inQuiet && !isUrgent && ch != store.ChannelInApp) {
			scheduledTime := now.Add(time.Duration(15) * time.Minute)
			condition := ""

			if matchedStep != nil {
				scheduledTime = now.Add(time.Duration(matchedStep.DelaySeconds) * time.Second)
				condition = matchedStep.Condition
			} else if inQuiet && !isUrgent {
				scheduledTime = quietEnd
			}

			payloadBytes, _ := json.Marshal(req.Payload)
			_ = we.store.CreateScheduledDelivery(&store.ScheduledDelivery{
				NotificationID: notificationID,
				WorkflowID:     wf.ID,
				SubscriberID:   sub.ID,
				Channel:        ch,
				Subject:        renderedSubject,
				Body:           strings.TrimSpace(renderedBody),
				HTMLBody:       htmlBody,
				ActionURL:      actionURL,
				Payload:        string(payloadBytes),
				ScheduledFor:   scheduledTime,
				Condition:      condition,
				Status:         "pending",
			})

			skipped[ch] = fmt.Sprintf("scheduled_for_%s", scheduledTime.Format(time.RFC3339))
			continue
		}

		compiled := &CompiledMessage{
			Channel:    ch,
			Subject:    renderedSubject,
			Body:       strings.TrimSpace(renderedBody),
			HTMLBody:   htmlBody,
			ActionURL:  actionURL,
			TrackingID: trackingID,
			Subscriber: sub,
			WorkflowID: wf.ID,
			Payload:    req.Payload,
		}

		compiledMessages = append(compiledMessages, compiled)
	}

	return compiledMessages, notificationID, skipped, nil
}
