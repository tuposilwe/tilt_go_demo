package main

import (
	"math"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestValidateRestaurant(t *testing.T) {
	valid := Restaurant{
		Name:       "Example Bistro",
		Cuisine:    "French",
		Address:    "10 Main Street",
		City:       "Portland",
		Country:    "United States",
		PriceLevel: 2,
		Rating:     4.5,
	}

	tests := []struct {
		name    string
		change  func(*Restaurant)
		wantErr bool
	}{
		{name: "valid restaurant"},
		{name: "missing name", change: func(r *Restaurant) { r.Name = "" }, wantErr: true},
		{name: "invalid price level", change: func(r *Restaurant) { r.PriceLevel = 5 }, wantErr: true},
		{name: "invalid rating", change: func(r *Restaurant) { r.Rating = 5.1 }, wantErr: true},
		{name: "non-finite rating", change: func(r *Restaurant) { r.Rating = math.NaN() }, wantErr: true},
		{name: "invalid email", change: func(r *Restaurant) { r.Email = "not-an-email" }, wantErr: true},
		{name: "invalid website", change: func(r *Restaurant) { r.Website = "ftp://example.com" }, wantErr: true},
		{name: "valid contact details", change: func(r *Restaurant) {
			r.Email = "hello@example.com"
			r.Website = "https://example.com"
		}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			restaurant := valid
			if test.change != nil {
				test.change(&restaurant)
			}
			err := validateRestaurant(restaurant)
			if (err != nil) != test.wantErr {
				t.Fatalf("validateRestaurant() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestDecodeInputRejectsUnknownAndTrailingJSON(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{name: "unknown field", body: `{"name":"Bistro","unknown":"value"}`},
		{name: "multiple values", body: `{} {}`},
		{name: "invalid JSON", body: `{"name":`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest("POST", "/api/restaurants", strings.NewReader(test.body))
			request.Header.Set("Content-Type", "application/json")
			recorder := httptest.NewRecorder()
			if decodeInput(recorder, request, &restaurantInput{}) {
				t.Fatal("decodeInput() accepted invalid request body")
			}
			if recorder.Code != 400 {
				t.Fatalf("status = %d, want 400", recorder.Code)
			}
		})
	}
}

func TestParseQueryInt(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		key      string
		fallback int
		min      int
		max      int
		want     int
		wantErr  bool
	}{
		{name: "default", fallback: 20, min: 1, max: 100, want: 20},
		{name: "valid", query: "?limit=100", key: "limit", fallback: 20, min: 1, max: 100, want: 100},
		{name: "below minimum", query: "?limit=0", key: "limit", min: 1, max: 100, wantErr: true},
		{name: "not an integer", query: "?limit=large", key: "limit", min: 1, max: 100, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest("GET", "/api/restaurants"+test.query, nil)
			got, err := parseQueryInt(request, test.key, test.fallback, test.min, test.max)
			if (err != nil) != test.wantErr {
				t.Fatalf("parseQueryInt() error = %v, wantErr %v", err, test.wantErr)
			}
			if err == nil && got != test.want {
				t.Fatalf("parseQueryInt() = %d, want %d", got, test.want)
			}
		})
	}
}
