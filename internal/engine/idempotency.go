package engine

import (
	"encoding/json"

	"github.com/alexandrmotologa/notiflux/internal/store"
)

// IdempotencyEngine handles deduplication of requests with Idempotency-Key headers.
type IdempotencyEngine struct {
	store *store.Store
}

func NewIdempotencyEngine(s *store.Store) *IdempotencyEngine {
	return &IdempotencyEngine{store: s}
}

// Check returns a cached response if key exists.
func (ie *IdempotencyEngine) Check(key string) (*store.IdempotencyRecord, error) {
	if key == "" {
		return nil, nil
	}
	return ie.store.GetIdempotencyKey(key)
}

// Save stores the response associated with key.
func (ie *IdempotencyEngine) Save(key string, statusCode int, resp any) error {
	if key == "" {
		return nil
	}
	b, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	return ie.store.SaveIdempotencyKey(&store.IdempotencyRecord{
		Key:          key,
		StatusCode:   statusCode,
		ResponseBody: string(b),
	})
}
