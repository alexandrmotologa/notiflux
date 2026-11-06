package channels

import (
	"context"
	"fmt"
	"time"

	"github.com/alexandrmotologa/notiflux/internal/engine"
	"github.com/alexandrmotologa/notiflux/internal/store"
	"github.com/google/uuid"
)

// InAppDispatcher handles persistence and real-time SSE broadcasts for in-app bell notifications.
type InAppDispatcher struct {
	store  *store.Store
	sseHub *SSEHub
}

func NewInAppDispatcher(s *store.Store, hub *SSEHub) *InAppDispatcher {
	return &InAppDispatcher{
		store:  s,
		sseHub: hub,
	}
}

func (d *InAppDispatcher) Channel() string {
	return store.ChannelInApp
}

func (d *InAppDispatcher) Dispatch(ctx context.Context, msg *engine.CompiledMessage) error {
	if msg.Subscriber == nil || msg.Subscriber.ID == "" {
		return fmt.Errorf("subscriber id is required for in-app dispatch")
	}

	notif := &store.Notification{
		ID:           uuid.NewString(),
		WorkflowID:   msg.WorkflowID,
		SubscriberID: msg.Subscriber.ID,
		Title:        msg.Subject,
		Body:         msg.Body,
		Data:         msg.Payload,
		ActionURL:    msg.ActionURL,
		IsRead:       false,
		CreatedAt:    time.Now().UTC(),
	}

	if err := d.store.CreateNotification(notif); err != nil {
		return fmt.Errorf("failed persisting in-app notification: %w", err)
	}

	// Calculate current unread count
	unreadCount, _ := d.store.GetUnreadCount(msg.Subscriber.ID)

	// Broadcast to connected web/client sessions
	if d.sseHub != nil {
		eventPayload := map[string]any{
			"notification": notif,
			"unread_count": unreadCount,
		}
		d.sseHub.Broadcast(msg.Subscriber.ID, "notification.new", eventPayload)
	}

	return nil
}
