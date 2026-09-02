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

// Audnexus exposes ASIN routes only. A title-only search must not invent a
// /books?q= request: the API answers that with 404, and the empty body then
// decoded as "unexpected end of JSON input" on every search (issue #462).
func TestAudnexusTitleOnlySearchIssuesNoRequest(t *testing.T) {
	var requests int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests++
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"message":"Route GET:/books not found","statusCode":404}`))
	}))
	defer srv.Close()

	client := NewAudnexusClient()
	client.baseURL = srv.URL

	results, err := client.Search(context.Background(), metadata.SearchQuery{Title: "Hitchhiker"})
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("results = %d, want 0", len(results))
	}
	if requests != 0 {
		t.Fatalf("requests = %d, want 0 (audnexus has no title search)", requests)
	}
}

// A 404 from the ASIN route means "no such book", which get() reports as a nil
// body. Decoding that nil body produced a bogus JSON error instead.
func TestAudnexusFetchUnknownASINReturnsNoMatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := NewAudnexusClient()
	client.baseURL = srv.URL

	match, err := client.Fetch(context.Background(), "B0000000000")
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}
	if match != nil {
		t.Fatalf("match = %+v, want nil", match)
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
