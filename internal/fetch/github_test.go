package fetch

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/wsoule/tmpl/internal/source"
)

func TestGitHubAnonymousUsesCodeload(t *testing.T) {
	var gotPath, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth = r.URL.Path, r.Header.Get("Authorization")
		_, _ = w.Write(tarball(t, []entry{{name: "tmpl-sha/templates/pierre/a.txt", body: "a"}}).Bytes())
	}))
	defer srv.Close()

	g := &GitHub{Client: srv.Client(), CodeloadURL: srv.URL, APIURL: "http://unused"}
	res, err := g.Fetch(context.Background(), "wsoule", "tmpl", "", "templates/pierre")
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != "/wsoule/tmpl/tar.gz/HEAD" || gotAuth != "" {
		t.Errorf("path=%q auth=%q", gotPath, gotAuth)
	}
	if len(res.Files) != 1 || res.Files[0].Path != "a.txt" {
		t.Errorf("files = %+v", res.Files)
	}
}

func TestGitHubTokenUsesAPI(t *testing.T) {
	var gotPath, gotAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotAuth = r.URL.Path, r.Header.Get("Authorization")
		_, _ = w.Write(tarball(t, []entry{{name: "r-sha/a.txt", body: "a"}}).Bytes())
	}))
	defer srv.Close()

	g := &GitHub{Client: srv.Client(), CodeloadURL: "http://unused", APIURL: srv.URL, Token: "secret"}
	if _, err := g.Fetch(context.Background(), "o", "r", "feature/x", ""); err != nil {
		t.Fatal(err)
	}
	if gotPath != "/repos/o/r/tarball/feature/x" || gotAuth != "Bearer secret" {
		t.Errorf("path=%q auth=%q", gotPath, gotAuth)
	}
}

func TestGitHubStaleTokenFallsBackToAnonymous(t *testing.T) {
	api := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer api.Close()
	codeload := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(tarball(t, []entry{{name: "r-sha/a.txt", body: "a"}}).Bytes())
	}))
	defer codeload.Close()

	g := &GitHub{Client: http.DefaultClient, CodeloadURL: codeload.URL, APIURL: api.URL, Token: "stale"}
	if _, err := g.Fetch(context.Background(), "o", "r", "", ""); err != nil {
		t.Fatalf("expected anonymous retry to succeed, got %v", err)
	}
}

func TestGitHubNotFound(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	g := &GitHub{Client: srv.Client(), CodeloadURL: srv.URL}
	if _, err := g.Fetch(context.Background(), "o", "nope", "", ""); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v", err)
	}
}

func TestGitHubServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()
	g := &GitHub{Client: srv.Client(), CodeloadURL: srv.URL}
	_, err := g.Fetch(context.Background(), "o", "r", "", "")
	if err == nil || errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want a non-NotFound error", err)
	}
}

func TestFetchDispatchesLocal(t *testing.T) {
	root := t.TempDir()
	write(t, root, "a.txt", "a", 0o644)
	res, err := Fetch(context.Background(), source.Source{Kind: source.Local, Path: root}, NewGitHub(""))
	if err != nil || len(res.Files) != 1 {
		t.Fatalf("res=%+v err=%v", res, err)
	}
}

func TestTokenFromEnv(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", " abc ")
	if got := TokenFromEnv(); got != "abc" {
		t.Errorf("got %q", got)
	}
}
