package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Silo-Server/silo-plugin-audiobook-metadata/metadata"
)

func TestAudnexusFetch(t *testing.T) {
	fixture, err := os.ReadFile("testdata/audnexus_book.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(fixture)
	}))
	defer srv.Close()

	client := NewAudnexusClient()
	client.baseURL = srv.URL

	m, err := client.Fetch(context.Background(), "B002V0QHBU")
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}
	if m == nil {
		t.Fatal("expected non-nil match")
	}

	if m.Provider != "audnexus" {
		t.Errorf("Provider = %q, want %q", m.Provider, "audnexus")
	}
	if m.ASIN != "B002V0QHBU" {
		t.Errorf("ASIN = %q, want %q", m.ASIN, "B002V0QHBU")
	}
	if m.Title != "The Hitchhiker's Guide to the Galaxy" {
		t.Errorf("Title = %q", m.Title)
	}
	if len(m.Authors) == 0 || m.Authors[0] != "Douglas Adams" {
		t.Errorf("Authors = %v", m.Authors)
	}
	if len(m.Narrators) == 0 || m.Narrators[0] != "Stephen Fry" {
		t.Errorf("Narrators = %v", m.Narrators)
	}
	if m.DurationMin != 193 {
		t.Errorf("DurationMin = %d, want 193", m.DurationMin)
	}
	if m.PublishYear != 2005 {
		t.Errorf("PublishYear = %d, want 2005", m.PublishYear)
	}
	if m.SeriesName != "Hitchhiker's Guide to the Galaxy" {
		t.Errorf("SeriesName = %q", m.SeriesName)
	}
	if m.SeriesPosition != "1" {
		t.Errorf("SeriesPosition = %q, want %q", m.SeriesPosition, "1")
	}
	// Only "genre"-type entries should appear.
	if len(m.Genres) != 1 || m.Genres[0] != "Science Fiction" {
		t.Errorf("Genres = %v, want [Science Fiction]", m.Genres)
	}
	// HTML tags should be stripped.
	if m.Description == "" {
		t.Error("Description should not be empty after HTML strip")
	}
}

func TestAudnexusSearchByASIN(t *testing.T) {
	fixture, err := os.ReadFile("testdata/audnexus_book.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(fixture)
	}))
	defer srv.Close()

	client := NewAudnexusClient()
	client.baseURL = srv.URL

	q := metadata.SearchQuery{
		ProviderIDs: map[string]string{"asin": "B002V0QHBU"},
	}
	results, err := client.Search(context.Background(), q)
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].ASIN != "B002V0QHBU" {
		t.Errorf("ASIN = %q", results[0].ASIN)
	}
}

func TestAudnexusTitleOnlyQueryIsDeclined(t *testing.T) {
	// Audnexus has NO title-search endpoint: GET /books?q= and /books?title=
	// both return 404 "Route not found" (verified against the live API
	// 2026-08-21); only /books/{asin} exists.
	//
	// The previous version of this test pointed a mock server at any path and
	// had it return a book array, which "proved" a title search that the real
	// service has never offered. It passed for years while production logged
	// "decode search response: unexpected end of JSON input" on every call.
	// So this test now asserts the contract that matters: with no ASIN, the
	// client must decline WITHOUT making a request.
	var hits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"Route not found","statusCode":404}`))
	}))
	defer srv.Close()

	client := NewAudnexusClient()
	client.baseURL = srv.URL

	results, err := client.Search(context.Background(), metadata.SearchQuery{Title: "Hitchhiker"})
	if err != nil {
		t.Fatalf("declining should not be an error, got: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected no results, got %d", len(results))
	}
	if hits != 0 {
		t.Errorf("expected no HTTP request for a title-only query, got %d", hits)
	}
}

func TestStripHTML(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"<p>Hello <b>World</b></p>", "Hello World"},
		{"No tags here", "No tags here"},
		{"&amp; &lt;tag&gt;", "& <tag>"},
	}
	for _, tt := range tests {
		got := stripHTML(tt.in)
		if got != tt.want {
			t.Errorf("stripHTML(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
