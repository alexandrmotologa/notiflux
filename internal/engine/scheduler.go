package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/alexandrmotologa/notiflux/internal/store"
)

// DispatchInvoker defines the minimal dispatch capability needed by the scheduler.
type DispatchInvoker interface {
	DispatchAll(ctx context.Context, messages []*CompiledMessage, notificationID string) ([]string, map[string]string)
}

// Scheduler manages delayed step deliveries, retry queues, quiet hour releases, and digest windows.
type Scheduler struct {
	store    *store.Store
	compiler *Compiler
	invoker  DispatchInvoker
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

func NewScheduler(s *store.Store, invoker DispatchInvoker) *Scheduler {
	return &Scheduler{
		store:    s,
		compiler: NewCompiler(),
		invoker:  invoker,
		stopCh:   make(chan struct{}),
	}
}

// Start initiates the background scheduler loop.
func (sc *Scheduler) Start() {
	sc.wg.Add(1)
	go func() {
		defer sc.wg.Done()
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-sc.stopCh:
				return
			case <-ticker.C:
				sc.processPendingDeliveries()
				sc.processDigests()
			}
		}
	}()
}

// Stop gracefully shuts down the scheduler.
func (sc *Scheduler) Stop() {
	close(sc.stopCh)
	sc.wg.Wait()
}

func (sc *Scheduler) processPendingDeliveries() {
	now := time.Now().UTC()
	pending, err := sc.store.ListPendingScheduledDeliveries(now, 50)
	if err != nil || len(pending) == 0 {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()

	for _, item := range pending {
		// Check conditional rule (e.g. if_unread)
		if item.Condition == "if_unread" && item.NotificationID != "" {
			notif, err := sc.store.GetNotification(item.NotificationID)
			if err == nil && notif != nil && notif.IsRead {
				// User already read in-app notification, skip fallback!
				_ = sc.store.UpdateScheduledDeliveryStatus(item.ID, "skipped", "notification was read before fallback", item.Attempts)
				continue
			}
		}

		// Rebuild compiled message
		var payload map[string]any
		if item.Payload != "" {
			_ = json.Unmarshal([]byte(item.Payload), &payload)
		}

		sub, err := sc.store.GetSubscriber(item.SubscriberID)
		if err != nil || sub == nil {
			sub = &store.Subscriber{ID: item.SubscriberID}
		}

		msg := &CompiledMessage{
			Channel:    item.Channel,
			Subject:    item.Subject,
			Body:       item.Body,
			HTMLBody:   item.HTMLBody,
			ActionURL:  item.ActionURL,
			Subscriber: sub,
			WorkflowID: item.WorkflowID,
			Payload:    payload,
		}

		dispatched, errs := sc.invoker.DispatchAll(ctx, []*CompiledMessage{msg}, item.NotificationID)
		if len(errs) > 0 {
			attempts := item.Attempts + 1
			if attempts >= item.MaxAttempts {
				_ = sc.store.UpdateScheduledDeliveryStatus(item.ID, "failed", errs[item.Channel], attempts)
			} else {
				// Exponential backoff: 1m, 2m, 4m...
				backoffDuration := time.Duration(1<<attempts) * time.Minute
				nextRetry := time.Now().UTC().Add(backoffDuration)
				_ = sc.rescheduleDelivery(item.ID, nextRetry, attempts)
			}
		} else if len(dispatched) > 0 {
			_ = sc.store.UpdateScheduledDeliveryStatus(item.ID, "completed", "", item.Attempts)
		}
	}
}

func (sc *Scheduler) rescheduleDelivery(id string, nextRun time.Time, attempts int) error {
	query := `UPDATE scheduled_deliveries SET status = 'pending', scheduled_for = ?, attempts = ? WHERE id = ?`
	_, err := sc.store.DB().Exec(query, nextRun, attempts, id)
	return err
}

func (sc *Scheduler) processDigests() {
	pairs, err := sc.store.ListPendingDigestSubscribers()
	if err != nil || len(pairs) == 0 {
		return
	}

	now := time.Now().UTC()

	for _, pair := range pairs {
		wf, err := sc.store.GetWorkflow(pair.WorkflowID)
		if err != nil || wf == nil || !wf.DigestEnabled {
			continue
		}

		entries, err := sc.store.GetDigestEntries(pair.WorkflowID, pair.SubscriberID)
		if err != nil || len(entries) == 0 {
			continue
		}

		// Check if digest window expired or max items reached
		windowSec := wf.DigestWindowSeconds
		if windowSec <= 0 {
			windowSec = 300 // default 5 minutes
		}

		oldest := entries[0].CreatedAt
		timeElapsed := now.Sub(oldest).Seconds()
		maxItems := wf.DigestMaxItems
		if maxItems <= 0 {
			maxItems = 10
		}

		shouldFlush := timeElapsed >= float64(windowSec) || len(entries) >= maxItems
		if !shouldFlush {
			continue
		}

		// Flush digest!
		sc.flushDigest(wf, pair.SubscriberID, entries)
	}
}

func (sc *Scheduler) flushDigest(wf *store.Workflow, subscriberID string, entries []*store.DigestEntry) {
	defer func() {
		_ = sc.store.ClearDigestEntries(wf.ID, subscriberID)
	}()

	sub, err := sc.store.GetSubscriber(subscriberID)
	if err != nil || sub == nil {
		sub = &store.Subscriber{ID: subscriberID}
	}

	var items []any
	for _, e := range entries {
		var itemPayload map[string]any
		if err := json.Unmarshal([]byte(e.Payload), &itemPayload); err == nil {
			items = append(items, itemPayload)
		}
	}

	mergedData := map[string]any{
		"items":      items,
		"item_count": len(items),
		"workflow":   wf.Name,
		"subscriber": map[string]any{
			"id":    sub.ID,
			"email": sub.Email,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Compile digest message for enabled channels
	for _, ch := range wf.Channels {
		tpl, _ := sc.store.GetTemplateByWorkflowAndChannel(wf.ID, ch)
		subjectTmpl := fmt.Sprintf("%s Digest (%d updates)", wf.Name, len(items))
		bodyTmpl := fmt.Sprintf("You have %d new updates:\n{{#each items}}\n- {{title}}{{/each}}", len(items))

		if tpl != nil {
			subjectTmpl = tpl.Subject
			bodyTmpl = tpl.Body
		}

		renderedSubject, _ := sc.compiler.Render(subjectTmpl, mergedData, false)
		renderedBody, _ := sc.compiler.Render(bodyTmpl, mergedData, ch == store.ChannelEmail)
		htmlBody := ""
		if ch == store.ChannelEmail {
			rawBody, _ := sc.compiler.Render(bodyTmpl, mergedData, false)
			htmlBody = WrapEmailHTML(MarkdownToHTML(rawBody), renderedSubject)
		}

		msg := &CompiledMessage{
			Channel:    ch,
			Subject:    renderedSubject,
			Body:       renderedBody,
			HTMLBody:   htmlBody,
			Subscriber: sub,
			WorkflowID: wf.ID,
			Payload:    mergedData,
		}

		sc.invoker.DispatchAll(ctx, []*CompiledMessage{msg}, "digest_"+wf.ID)
	}

	log.Printf("[notiflux] flushed digest for subscriber=%s workflow=%s count=%d", subscriberID, wf.ID, len(items))
}
