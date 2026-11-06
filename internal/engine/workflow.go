package engine

import (
	"encoding/json"
	"fmt"
	"strings"

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
	SkippedChannels    map[string]string `json:"skipped_channels,omitempty"`
	ErrorMessage       string            `json:"error,omitempty"`
}

// CompiledMessage is a fully interpolated message ready for channel dispatch.
type CompiledMessage struct {
	Channel      string
	Subject      string
	Body         string
	HTMLBody     string
	ActionURL    string
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

	// 3. Determine active channels
	var overrideChannels []string
	if req.Overrides != nil && len(req.Overrides.Channels) > 0 {
		overrideChannels = req.Overrides.Channels
	}

	activeChannels, skipped, err := we.prefEngine.FilterActiveChannels(wf, &req.Subscriber, overrideChannels)
	if err != nil {
		return nil, "", nil, fmt.Errorf("error evaluating channel preferences: %w", err)
	}

	// 4. Merge data context for compiler
	contextData := make(map[string]any)
	for k, v := range req.Payload {
		contextData[k] = v
	}
	contextData["subscriber"] = map[string]any{
		"id":               req.Subscriber.ID,
		"email":            req.Subscriber.Email,
		"telegram_chat_id": req.Subscriber.TelegramChatID,
	}

	notificationID := uuid.NewString()
	var compiledMessages []*CompiledMessage

	// Action URL lookup from payload
	actionURL := ""
	if urlVal, ok := req.Payload["action_url"]; ok {
		actionURL = fmt.Sprintf("%v", urlVal)
	}

	// 5. Compile each active channel
	for _, ch := range activeChannels {
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
			// Fallback: derive from payload title/body or workflow name
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

		// Interpolate subject
		renderedSubject, err := we.compiler.Render(subjectTmpl, contextData, false)
		if err != nil {
			renderedSubject = subjectTmpl
		}

		// Interpolate body
		escapeHTML := (ch == store.ChannelEmail)
		renderedBody, err := we.compiler.Render(bodyTmpl, contextData, escapeHTML)
		if err != nil {
			renderedBody = bodyTmpl
		}

		htmlBody := ""
		if ch == store.ChannelEmail {
			// Convert markdown in body to styled HTML email
			rawRendered, _ := we.compiler.Render(bodyTmpl, contextData, false)
			convertedHTML := MarkdownToHTML(rawRendered)
			htmlBody = WrapEmailHTML(convertedHTML, renderedSubject)
		}

		compiled := &CompiledMessage{
			Channel:    ch,
			Subject:    renderedSubject,
			Body:       strings.TrimSpace(renderedBody),
			HTMLBody:   htmlBody,
			ActionURL:  actionURL,
			Subscriber: &req.Subscriber,
			WorkflowID: wf.ID,
			Payload:    req.Payload,
		}

		compiledMessages = append(compiledMessages, compiled)
	}

	return compiledMessages, notificationID, skipped, nil
}
