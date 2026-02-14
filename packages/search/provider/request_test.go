package provider

import (
	"testing"
	"time"

	"golang.org/x/text/currency"
)

func TestClassConstants(t *testing.T) {
	if Economy != 1 {
		t.Errorf("expected Economy=1, got %d", Economy)
	}
	if PremiumEconomy != 2 {
		t.Errorf("expected PremiumEconomy=2, got %d", PremiumEconomy)
	}
	if Business != 3 {
		t.Errorf("expected Business=3, got %d", Business)
	}
	if First != 4 {
		t.Errorf("expected First=4, got %d", First)
	}
}

func TestRequest_StructFields(t *testing.T) {
	depDate := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	retDate := time.Date(2025, 6, 8, 0, 0, 0, 0, time.UTC)

	req := Request{
		Origin:        "LHR",
		Destination:   "LAX",
		DepartureDate: depDate,
		ReturnDate:    retDate,
		Adults:        2,
		Children:      1,
		Class:         Business,
		Currency:      currency.GBP,
	}

	if req.Origin != "LHR" {
		t.Errorf("expected origin LHR, got %s", req.Origin)
	}
	if req.Destination != "LAX" {
		t.Errorf("expected destination LAX, got %s", req.Destination)
	}
	if !req.DepartureDate.Equal(depDate) {
		t.Errorf("expected departure date %v, got %v", depDate, req.DepartureDate)
	}
	if !req.ReturnDate.Equal(retDate) {
		t.Errorf("expected return date %v, got %v", retDate, req.ReturnDate)
	}
	if req.Adults != 2 {
		t.Errorf("expected 2 adults, got %d", req.Adults)
	}
	if req.Children != 1 {
		t.Errorf("expected 1 child, got %d", req.Children)
	}
	if req.Class != Business {
		t.Errorf("expected Business class, got %d", req.Class)
	}
	if req.Currency != currency.GBP {
		t.Errorf("expected GBP currency, got %v", req.Currency)
	}
}

func TestRequest_ZeroValue(t *testing.T) {
	var req Request

	if req.Origin != "" {
		t.Errorf("expected empty origin, got %s", req.Origin)
	}
	if req.Destination != "" {
		t.Errorf("expected empty destination, got %s", req.Destination)
	}
	if req.Adults != 0 {
		t.Errorf("expected 0 adults, got %d", req.Adults)
	}
	if req.Children != 0 {
		t.Errorf("expected 0 children, got %d", req.Children)
	}
	if req.Class != 0 {
		t.Errorf("expected 0 class, got %d", req.Class)
	}
}

func TestRequest_DifferentCurrencies(t *testing.T) {
	currencies := []currency.Unit{
		currency.GBP,
		currency.USD,
		currency.EUR,
	}

	for _, cur := range currencies {
		req := Request{Currency: cur}
		if req.Currency != cur {
			t.Errorf("expected currency %v, got %v", cur, req.Currency)
		}
	}
}

func TestRequest_AllClasses(t *testing.T) {
	classes := []Class{Economy, PremiumEconomy, Business, First}

	for i, c := range classes {
		expected := Class(i + 1)
		if c != expected {
			t.Errorf("class at index %d: expected %d, got %d", i, expected, c)
		}
	}
}
