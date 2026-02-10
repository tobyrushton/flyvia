package search

import (
	"context"

	"github.com/tobyrushton/gflights"
)

type Search struct {
	ctx context.Context

	s *gflights.Session
}

func New(ctx context.Context) (*Search, error) {
	s, err := gflights.New()
	if err != nil {
		return nil, err
	}

	return &Search{
		ctx: ctx,
		s:   s,
	}, nil
}
