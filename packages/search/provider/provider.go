package provider

import (
	"context"

	"github.com/tobyrushton/flyvia/packages/search/itinery"
)

type Provider interface {
	Explore(
		ctx context.Context,
		origin string,
	) ([]itinery.Itinery, error)
	Search(
		ctx context.Context,
		req Request,
	) ([]itinery.Itinery, error)
}
