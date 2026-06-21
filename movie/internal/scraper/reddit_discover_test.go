package scraper

import "testing"

func TestNormalizeSub(t *testing.T) {
	if got := normalizeSub("r/Movies"); got != "movies" {
		t.Fatalf("normalizeSub(r/Movies) = %q, want movies", got)
	}
	if got := normalizeSub("AskReddit"); got != "" {
		t.Fatalf("normalizeSub(AskReddit) = %q, want empty", got)
	}
}

func TestExtractSubredditsFromText(t *testing.T) {
	text := "Try r/TrueFilm and also check /r/horror for more"
	subs := extractSubredditsFromText(text)
	if len(subs) != 2 {
		t.Fatalf("len(subs) = %d, want 2", len(subs))
	}
}

func TestRecommendationTitle(t *testing.T) {
	if !recommendationTitle("best horror movies you missed") {
		t.Fatal("expected recommendation title")
	}
	if recommendationTitle("random discussion") {
		t.Fatal("did not expect recommendation title")
	}
}

func TestLooksLikeMovieSub(t *testing.T) {
	if !looksLikeMovieSub("TrueFilm", "TrueFilm", "in depth film discussion") {
		t.Fatal("expected movie sub")
	}
	if looksLikeMovieSub("soccer", "Soccer", "football news") {
		t.Fatal("did not expect movie sub")
	}
}
