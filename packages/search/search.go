package search

import (
	"context"

	"github.com/tobyrushton/flyvia/packages/search/provider"
)

type Search struct {
	ctx context.Context
	p   provider.Provider
}

func New(
	ctx context.Context,
	provider provider.Provider,
) *Search {
	return &Search{
		ctx: ctx,
		p:   provider,
	}
}
