package provider

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/Silo-Server/silo-plugin-audiobook-metadata/metadata"
)

func TestITunesSearch(t *testing.T) {
	fixture, err := os.ReadFile("testdata/itunes_search.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/search" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("media") != "audiobook" {
			t.Errorf("missing media=audiobook param")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(fixture)
	}))
	defer srv.Close()

	client := NewITunesClient()
	client.baseURL = srv.URL

	q := metadata.SearchQuery{Title: "Hitchhiker"}
	results, err := client.Search(context.Background(), q)
	if err != nil {
		t.Fatalf("Search error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}

	m := results[0]
	if m.Provider != "itunes" {
		t.Errorf("Provider = %q, want itunes", m.Provider)
	}
	if m.Title != "The Hitchhiker's Guide to the Galaxy" {
		t.Errorf("Title = %q", m.Title)
	}
	if len(m.Authors) == 0 || m.Authors[0] != "Douglas Adams" {
		t.Errorf("Authors = %v", m.Authors)
	}
	if m.PublishYear != 2005 {
		t.Errorf("PublishYear = %d", m.PublishYear)
	}
	// artworkUrl600 should be used directly.
	if m.CoverURL != "https://is1-ssl.mzstatic.com/image/thumb/Music/abc/600x600bb.jpg" {
		t.Errorf("CoverURL = %q", m.CoverURL)
	}
	// 11580000 ms / 60000 = 193 min
	if m.DurationMin != 193 {
		t.Errorf("DurationMin = %d, want 193", m.DurationMin)
	}
	if len(m.Genres) == 0 || m.Genres[0] != "Science Fiction" {
		t.Errorf("Genres = %v", m.Genres)
	}
}

func TestITunesSearchEmpty(t *testing.T) {
	client := NewITunesClient()
	results, err := client.Search(context.Background(), metadata.SearchQuery{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for empty query, got %d", len(results))
	}
}

func TestITunesCoverArtworkFallback(t *testing.T) {
	client := NewITunesClient()

	// When artworkUrl600 is absent, the 100px URL should be substituted.
	r := iTunesResult{
		ArtworkURL100: "https://example.com/image/100x100bb.jpg",
	}
	got := client.coverArtwork(r)
	want := "https://example.com/image/600x600bb.jpg"
	if got != want {
		t.Errorf("coverArtwork = %q, want %q", got, want)
	}
}

// The host's enrichment sweep resolves an item by calling GetMetadata with the
// ID that Search accumulated, so a Fetch that returns nothing throws away every
// iTunes match and the item ends as "no metadata found" (issue #462).
func TestITunesFetchReturnsMatchForCollectionID(t *testing.T) {
	fixture, err := os.ReadFile("testdata/itunes_search.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/lookup" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("id"); got != "1440742" {
			t.Errorf("id param = %q, want %q", got, "1440742")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(fixture)
	}))
	defer srv.Close()

	client := NewITunesClient()
	client.baseURL = srv.URL

	match, err := client.Fetch(context.Background(), "1440742")
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}
	if match == nil {
		t.Fatal("Fetch returned no match")
	}
	if match.CoverURL == "" {
		t.Error("match has no cover URL")
	}
	if match.Provider != "itunes" {
		t.Errorf("Provider = %q, want %q", match.Provider, "itunes")
	}
}

func TestITunesFetchUnknownIDReturnsNoMatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"resultCount":0,"results":[]}`))
	}))
	defer srv.Close()

	client := NewITunesClient()
	client.baseURL = srv.URL

	match, err := client.Fetch(context.Background(), "0")
	if err != nil {
		t.Fatalf("Fetch error: %v", err)
	}
	if match != nil {
		t.Fatalf("match = %+v, want nil", match)
	}
}
