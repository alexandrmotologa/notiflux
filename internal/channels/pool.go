package channels

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/alexandrmotologa/notiflux/internal/config"
	"github.com/alexandrmotologa/notiflux/internal/engine"
	"github.com/alexandrmotologa/notiflux/internal/store"
)

// DispatcherPool coordinates execution across all registered notification channels.
type DispatcherPool struct {
	store       *store.Store
	dispatchers map[string]Dispatcher
}

func NewDispatcherPool(cfg *config.Config, s *store.Store, sseHub *SSEHub) *DispatcherPool {
	pool := &DispatcherPool{
		store:       s,
		dispatchers: make(map[string]Dispatcher),
	}

	// Register channel dispatchers
	pool.Register(NewInAppDispatcher(s, sseHub))
	pool.Register(NewEmailDispatcher(cfg))
	pool.Register(NewWebPushDispatcher(cfg))
	pool.Register(NewTelegramDispatcher(cfg))
	pool.Register(NewSlackDispatcher(cfg))
	pool.Register(NewDiscordDispatcher())
	pool.Register(NewWebhookDispatcher(cfg))

	return pool
}

func (p *DispatcherPool) Register(d Dispatcher) {
	p.dispatchers[d.Channel()] = d
}

// DispatchAll dispatches compiled messages concurrently across all respective channels.
func (p *DispatcherPool) DispatchAll(
	ctx context.Context,
	messages []*engine.CompiledMessage,
	notificationID string,
) (dispatched []string, dispatchErrors map[string]string) {
	var mu sync.Mutex
	dispatchErrors = make(map[string]string)

	var wg sync.WaitGroup
	for _, msg := range messages {
		dispatcher, exists := p.dispatchers[msg.Channel]
		if !exists {
			mu.Lock()
			dispatchErrors[msg.Channel] = fmt.Sprintf("no dispatcher registered for channel %s", msg.Channel)
			mu.Unlock()
			continue
		}

		wg.Add(1)
		go func(d Dispatcher, m *engine.CompiledMessage) {
			defer wg.Done()

			err := d.Dispatch(ctx, m)

			// Record delivery log
			status := store.StatusSent
			errMsg := ""
			if err != nil {
				status = store.StatusFailed
				errMsg = err.Error()
			}

			payloadBytes, _ := json.Marshal(m.Payload)

			logEntry := &store.DeliveryLog{
				NotificationID: notificationID,
				WorkflowID:     m.WorkflowID,
				SubscriberID:   m.Subscriber.ID,
				Channel:        m.Channel,
				Status:         status,
				ErrorMessage:   errMsg,
				Payload:        string(payloadBytes),
				TrackingID:     m.TrackingID,
				TargetURL:      m.ActionURL,
			}
			_ = p.store.RecordDeliveryLog(logEntry)

			mu.Lock()
			if err != nil {
				dispatchErrors[m.Channel] = err.Error()
			} else {
				dispatched = append(dispatched, m.Channel)
			}
			mu.Unlock()
		}(dispatcher, msg)
	}

	wg.Wait()
	return dispatched, dispatchErrors
}
