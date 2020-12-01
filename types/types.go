package types

import(
	"context"
	"time"
)

type ChaosInterface interface {
	Invade(ctx context.Context , timeout time.Duration) error
}

