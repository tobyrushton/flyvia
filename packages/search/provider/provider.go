package provider

//go:generate counterfeiter -o providerfakes/fake_provider.go . Provider

import (
	"context"

	"github.com/tobyrushton/flyvia/packages/search/itinery"
)

type Provider interface {
	Explore(
		ctx context.Context,
		req Request,
		origin string,
	) ([]itinery.ExploreItinery, error)
	Search(
		ctx context.Context,
		req Request,
	) ([]itinery.Itinery, error)
	SortByPrice(itineries *[]itinery.Itinery)
}
