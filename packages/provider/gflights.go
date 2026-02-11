package provider

import (
	"context"

	"github.com/tobyrushton/flyvia/packages/leg"
	"github.com/tobyrushton/gflights"
)

type GFlights struct {
	s *gflights.Session
}

func NewGFlights() (*GFlights, error) {
	s, err := gflights.New()
	if err != nil {
		return nil, err
	}

	return &GFlights{
		s: s,
	}, nil
}

func (g *GFlights) Explore(
	ctx context.Context,
	origin string,
) ([]leg.Leg, error) {
	return nil, nil
}

func (g *GFlights) Search(
	ctx context.Context,
	origin, destination string,
) ([]leg.Leg, error) {
	return nil, nil
}

var _ Provider = (*GFlights)(nil)
