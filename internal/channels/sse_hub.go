package channels

import (
	"encoding/json"
	"fmt"
	"sync"
)

// SSEHub manages active Server-Sent Events subscribers and dispatches events.
type SSEHub struct {
	mu          sync.RWMutex
	subscribers map[string]map[chan []byte]struct{}
}

func NewSSEHub() *SSEHub {
	return &SSEHub{
		subscribers: make(map[string]map[chan []byte]struct{}),
	}
}

// Subscribe adds an SSE listener channel for a subscriberID, returning a cleanup function.
func (h *SSEHub) Subscribe(subscriberID string) (chan []byte, func()) {
	h.mu.Lock()
	defer h.mu.Unlock()

	ch := make(chan []byte, 32)
	if _, exists := h.subscribers[subscriberID]; !exists {
		h.subscribers[subscriberID] = make(map[chan []byte]struct{})
	}
	h.subscribers[subscriberID][ch] = struct{}{}

	cleanup := func() {
		h.mu.Lock()
		defer h.mu.Unlock()

		if clientMap, exists := h.subscribers[subscriberID]; exists {
			delete(clientMap, ch)
			close(ch)
			if len(clientMap) == 0 {
				delete(h.subscribers, subscriberID)
			}
		}
	}

	return ch, cleanup
}

// Broadcast sends an SSE event to all connected channels for subscriberID.
func (h *SSEHub) Broadcast(subscriberID, eventName string, payload any) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	clientMap, exists := h.subscribers[subscriberID]
	if !exists || len(clientMap) == 0 {
		return
	}

	dataBytes, err := json.Marshal(payload)
	if err != nil {
		dataBytes = []byte(`{}`)
	}

	rawEvent := fmt.Sprintf("event: %s\ndata: %s\n\n", eventName, string(dataBytes))
	msgBytes := []byte(rawEvent)

	for ch := range clientMap {
		select {
		case ch <- msgBytes:
		default:
			// Client buffer is full, drop to prevent blocking
		}
	}
}

// ActiveCount returns total connected SSE client sessions.
func (h *SSEHub) ActiveCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	count := 0
	for _, clients := range h.subscribers {
		count += len(clients)
	}
	return count
}
