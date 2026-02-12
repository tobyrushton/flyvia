package search

import (
	"context"
	"sort"
	"sync"

	"github.com/tobyrushton/flyvia/packages/search/itinery"
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

func (s *Search) Search(req provider.Request) ([]Result, error) {
	return nil, nil
}

func (s *Search) doSearch(req provider.Request) ([]Result, error) {
	return nil, nil
}

func (s *Search) explore(req provider.Request) ([]itinery.ExploreItinery, []itinery.ExploreItinery, error) {
	origins := []string{req.Destination, req.Origin}

	wg := sync.WaitGroup{}
	res := make([][]itinery.ExploreItinery, len(origins))
	var err error

	for i, origin := range origins {
		wg.Add(1)
		go func(i int, origin string) {
			defer wg.Done()
			it, err := s.p.Explore(s.ctx, req, origin)
			if err != nil {
				return
			}
			sort.Slice(it, func(i, j int) bool { return it[i].Price < it[j].Price })
			res[i] = it
		}(i, origin)
	}
	wg.Wait()

	if err != nil {
		return nil, nil, err
	}
	return res[0], res[1], nil
}
