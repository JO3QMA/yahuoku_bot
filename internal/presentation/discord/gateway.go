package discord

import (
	"context"

	"github.com/diamondburned/arikawa/v3/state"
)

// GatewaySession は Bot の Gateway 接続を抽象化する（テストで差し替え可能）。
type GatewaySession interface {
	AddHandler(fn interface{}) func()
	Connect(ctx context.Context) error
}

// stateGateway は本番用に *state.State を GatewaySession としてラップする。
type stateGateway struct {
	*state.State
}

func newStateGateway(s *state.State) GatewaySession {
	return &stateGateway{State: s}
}
