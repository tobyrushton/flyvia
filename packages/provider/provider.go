package provider

import (
	"context"

	"github.com/tobyrushton/flyvia/packages/leg"
)

type Provider interface {
	Explore(
		ctx context.Context,
		origin string,
	) ([]leg.Leg, error)
	Search(
		ctx context.Context,
		origin, destination string,
	) ([]leg.Leg, error)
}
