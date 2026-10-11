package fetch

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/wsoule/tmpl/internal/source"
)

// GitHub downloads repository tarballs. Anonymous requests go to codeload;
// with a token they go to the API's tarball endpoint so private repos work.
type GitHub struct {
	Client      *http.Client
	CodeloadURL string
	APIURL      string
	Token       string
}

// NewGitHub returns a fetcher pointed at github.com.
func NewGitHub(token string) *GitHub {
	return &GitHub{
		Client:      &http.Client{Timeout: 2 * time.Minute},
		CodeloadURL: "https://codeload.github.com",
		APIURL:      "https://api.github.com",
		Token:       token,
	}
}

// Fetch downloads owner/repo at ref ("" = default branch) and returns the
// files under subdir.
func (g *GitHub) Fetch(ctx context.Context, owner, repo, ref, subdir string) (Result, error) {
	res, status, err := g.fetch(ctx, owner, repo, ref, subdir, g.Token)
	if status == http.StatusUnauthorized && g.Token != "" {
		// A stale token shouldn't break public templates.
		res, _, err = g.fetch(ctx, owner, repo, ref, subdir, "")
	}
	return res, err
}

func (g *GitHub) fetch(ctx context.Context, owner, repo, ref, subdir, token string) (Result, int, error) {
	var url string
	if token != "" {
		url = fmt.Sprintf("%s/repos/%s/%s/tarball", g.APIURL, owner, repo)
		if ref != "" {
			url += "/" + ref
		}
	} else {
		r := ref
		if r == "" {
			r = "HEAD"
		}
		url = fmt.Sprintf("%s/%s/%s/tar.gz/%s", g.CodeloadURL, owner, repo, r)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return Result{}, 0, err
	}
	req.Header.Set("User-Agent", "tmpl")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := g.Client.Do(req)
	if err != nil {
		return Result{}, 0, fmt.Errorf("download %s/%s: %w", owner, repo, err)
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
	case http.StatusNotFound:
		return Result{}, resp.StatusCode, fmt.Errorf("%w: github.com/%s/%s", ErrNotFound, owner, repo)
	default:
		return Result{}, resp.StatusCode, fmt.Errorf("download %s/%s: unexpected status %s", owner, repo, resp.Status)
	}
	res, err := ExtractTarGz(resp.Body, subdir)
	if errors.Is(err, ErrNotFound) {
		return Result{}, resp.StatusCode, fmt.Errorf("%w: %s/%s/%s", ErrNotFound, owner, repo, subdir)
	}
	return res, resp.StatusCode, err
}

// TokenFromEnv returns a GitHub token from GITHUB_TOKEN or `gh auth token`,
// or "" when neither is available. It never fails.
func TokenFromEnv() string {
	if t := strings.TrimSpace(os.Getenv("GITHUB_TOKEN")); t != "" {
		return t
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	out, err := exec.CommandContext(ctx, "gh", "auth", "token").Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// Fetch loads the template a parsed source points at.
func Fetch(ctx context.Context, src source.Source, gh *GitHub) (Result, error) {
	switch src.Kind {
	case source.Local:
		return Local(src.Path)
	case source.GitHub:
		return gh.Fetch(ctx, src.Owner, src.Repo, src.Ref, src.Subdir)
	default:
		return Result{}, fmt.Errorf("unknown source kind %d", src.Kind)
	}
}
