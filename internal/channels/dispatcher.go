package channels

import (
	"context"

	"github.com/alexandrmotologa/notiflux/internal/engine"
)

// Dispatcher defines the interface implemented by each notification delivery channel.
type Dispatcher interface {
	Channel() string
	Dispatch(ctx context.Context, msg *engine.CompiledMessage) error
}
