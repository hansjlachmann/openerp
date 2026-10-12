package themes

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

func TestParseRepoURL(t *testing.T) {
	good := map[string]Repo{
		"https://github.com/r-bart/omarchy-starsend-theme":        {"r-bart", "omarchy-starsend-theme", ""},
		"https://github.com/r-bart/omarchy-starsend-theme/":       {"r-bart", "omarchy-starsend-theme", ""},
		"https://github.com/r-bart/omarchy-starsend-theme.git":    {"r-bart", "omarchy-starsend-theme", ""},
		"https://github.com/TyRichards/x.y_z/tree/main":           {"TyRichards", "x.y_z", "main"},
		"https://github.com/a/b/tree/release/v1.2":                {"a", "b", "release/v1.2"},
		"  https://github.com/robzolkos/omarchy-robzee84-theme  ": {"robzolkos", "omarchy-robzee84-theme", ""},
	}
	for in, want := range good {
		got, err := ParseRepoURL(in)
		if err != nil || got != want {
			t.Errorf("ParseRepoURL(%q) = %+v, %v; want %+v", in, got, err, want)
		}
	}
	bad := []string{
		"", "github.com/a/b", "http://github.com/a/b", "https://github.com/a", "https://github.com.evil.com/a/b",
		"https://evil.com/a/b", "https://user@github.com/a/b", "https://github.com:8080/a/b",
		"https://github.com/a/b?x=1", "https://github.com/a/b#x", "https://github.com/a/..",
		"https://github.com/a/b/tree/../../x", "https://github.com/a/b/blob/main/colors.toml",
		"https://github.com/-a/b", "https://github.com/a_b/c", "javascript:alert(1)",
		"https://github.com/a/b/tree/main;rm", "https://raw.githubusercontent.com/a/b/main/colors.toml",
	}
	for _, in := range bad {
		if r, err := ParseRepoURL(in); err == nil {
			t.Errorf("ParseRepoURL(%q) = %+v, want error", in, r)
		}
	}
}

// fakeGitHub serves the API and the raw files of one repository (both on one test server).
type fakeGitHub struct {
	srv       *httptest.Server
	mu        sync.Mutex
	requested []string
	auth      map[string]string // path → Authorization header
	commit    string
	tree      []map[string]any
	truncated bool
	files     map[string][]byte
	handle    func(w http.ResponseWriter, r *http.Request) bool // override; true = handled
}

const testCommit = "0123456789abcdef0123456789abcdef01234567"

func newFakeGitHub(t *testing.T, files map[string][]byte) *fakeGitHub {
	t.Helper()
	g := &fakeGitHub{commit: testCommit, files: files, auth: map[string]string{}}
	for p, b := range files {
		g.tree = append(g.tree, map[string]any{"path": p, "type": "blob", "size": len(b)})
	}
	g.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		g.mu.Lock()
		g.requested = append(g.requested, r.URL.Path)
		g.auth[r.URL.Path] = r.Header.Get("Authorization")
		g.mu.Unlock()
		if g.handle != nil && g.handle(w, r) {
			return
		}
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/repos/o/r/commits/"):
			_, _ = fmt.Fprint(w, g.commit)
		case r.URL.Path == "/api/repos/o/r/git/trees/"+g.commit:
			_ = json.NewEncoder(w).Encode(map[string]any{"sha": g.commit, "tree": g.tree, "truncated": g.truncated})
		case strings.HasPrefix(r.URL.Path, "/raw/o/r/"+g.commit+"/"):
			p := strings.TrimPrefix(r.URL.Path, "/raw/o/r/"+g.commit+"/")
			if b, ok := g.files[p]; ok {
				_, _ = w.Write(b)
				return
			}
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(g.srv.Close)
	return g
}

func (g *fakeGitHub) fetcher(token string) *Fetcher {
	return newFetcher(g.srv.URL+"/api", g.srv.URL+"/raw", token)
}

var testRepo = Repo{Owner: "o", Name: "r"}

func pngBytes(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for x := range w {
		for y := range h {
			img.Set(x, y, color.RGBA{uint8(x), uint8(y), 80, 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestFetchReadsOnlyThemeFiles(t *testing.T) {
	colors := []byte("background = \"#101010\"\nforeground = \"#eeeeee\"\naccent = \"#ff8800\"\n")
	img := pngBytes(t, 40, 30)
	g := newFakeGitHub(t, map[string][]byte{
		"colors.toml":              colors,
		"preview.png":              img,
		"backgrounds/1-dune.png":   img,
		"backgrounds/2-arch.png":   img,
		"install.sh":               []byte("rm -rf /"),
		"waybar.css":               []byte("* { color: red }"),
		"neovim.lua":               []byte("os.execute('x')"),
		"backgrounds/logo.svg":     []byte("<svg onload=alert(1)>"),
		"backgrounds/sub/deep.png": img,
		"backgrounds/anim.gif":     []byte("GIF89a"),
	})
	d, err := g.fetcher("secret-token").Fetch(context.Background(), testRepo)
	if err != nil {
		t.Fatal(err)
	}
	if d.CommitSHA != testCommit || d.ColorsName != "colors.toml" || !bytes.Equal(d.Colors, colors) || len(d.Backgrounds) != 2 || d.Preview == nil {
		t.Fatalf("download = commit %s, %s, %d backgrounds, preview %v", d.CommitSHA, d.ColorsName, len(d.Backgrounds), d.Preview != nil)
	}
	for _, p := range g.requested {
		for _, banned := range []string{"install.sh", "waybar.css", "neovim.lua", ".svg", ".gif", "sub/deep"} {
			if strings.Contains(p, banned) {
				t.Errorf("requested %s", p)
			}
		}
	}
	// The token goes to the API only, never to the file host
	for p, auth := range g.auth {
		isAPI := strings.HasPrefix(p, "/api/")
		if isAPI && auth != "Bearer secret-token" || !isAPI && auth != "" {
			t.Errorf("%s: Authorization %q", p, auth)
		}
	}
}

func TestFetchLimits(t *testing.T) {
	colors := []byte("background = \"#101010\"\nforeground = \"#eeeeee\"\n")

	t.Run("no theme file", func(t *testing.T) {
		g := newFakeGitHub(t, map[string][]byte{"README.md": []byte("x")})
		if _, err := g.fetcher("").Fetch(context.Background(), testRepo); !errors.Is(err, ErrNoThemeColors) {
			t.Errorf("err = %v", err)
		}
	})
	t.Run("truncated tree", func(t *testing.T) {
		g := newFakeGitHub(t, map[string][]byte{"colors.toml": colors})
		g.truncated = true
		if _, err := g.fetcher("").Fetch(context.Background(), testRepo); !errors.Is(err, ErrTreeTooLarge) {
			t.Errorf("err = %v", err)
		}
	})
	t.Run("file larger than the tree says", func(t *testing.T) {
		g := newFakeGitHub(t, map[string][]byte{"colors.toml": colors})
		g.files["colors.toml"] = bytes.Repeat([]byte("#"), MaxThemeFileSize+10) // tree still says small
		if _, err := g.fetcher("").Fetch(context.Background(), testRepo); !errors.Is(err, ErrTooLarge) {
			t.Errorf("err = %v", err)
		}
	})
	t.Run("gzip bomb", func(t *testing.T) {
		g := newFakeGitHub(t, map[string][]byte{"colors.toml": colors})
		var bomb bytes.Buffer
		zw := gzip.NewWriter(&bomb)
		_, _ = zw.Write(bytes.Repeat([]byte{' '}, 50<<20)) // 50 MB of spaces, ~50 KB compressed
		_ = zw.Close()
		g.handle = func(w http.ResponseWriter, r *http.Request) bool {
			if !strings.HasSuffix(r.URL.Path, "/colors.toml") {
				return false
			}
			w.Header().Set("Content-Encoding", "gzip")
			_, _ = w.Write(bomb.Bytes())
			return true
		}
		if _, err := g.fetcher("").Fetch(context.Background(), testRepo); !errors.Is(err, ErrTooLarge) {
			t.Errorf("err = %v", err)
		}
	})
	t.Run("redirect to another host", func(t *testing.T) {
		evil := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t.Error("request reached the other host")
		}))
		defer evil.Close()
		g := newFakeGitHub(t, map[string][]byte{"colors.toml": colors})
		g.handle = func(w http.ResponseWriter, r *http.Request) bool {
			if strings.HasSuffix(r.URL.Path, "/colors.toml") {
				http.Redirect(w, r, evil.URL+"/colors.toml", http.StatusFound)
				return true
			}
			return false
		}
		if _, err := g.fetcher("").Fetch(context.Background(), testRepo); err == nil {
			t.Error("redirect followed")
		}
	})
	t.Run("rate limit", func(t *testing.T) {
		g := newFakeGitHub(t, map[string][]byte{"colors.toml": colors})
		g.handle = func(w http.ResponseWriter, r *http.Request) bool {
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.WriteHeader(http.StatusForbidden)
			return true
		}
		if _, err := g.fetcher("").Fetch(context.Background(), testRepo); !errors.Is(err, ErrRateLimited) {
			t.Errorf("err = %v", err)
		}
	})
	t.Run("bad commit id", func(t *testing.T) {
		g := newFakeGitHub(t, map[string][]byte{"colors.toml": colors})
		g.handle = func(w http.ResponseWriter, r *http.Request) bool {
			if strings.Contains(r.URL.Path, "/commits/") {
				_, _ = fmt.Fprint(w, "../../etc/passwd")
				return true
			}
			return false
		}
		if _, err := g.fetcher("").Fetch(context.Background(), testRepo); err == nil {
			t.Error("accepted a bad commit id")
		}
	})
	t.Run("too many backgrounds are capped", func(t *testing.T) {
		files := map[string][]byte{"colors.toml": colors}
		for i := range 8 {
			files[fmt.Sprintf("backgrounds/%d.png", i)] = pngBytes(t, 4, 4)
		}
		g := newFakeGitHub(t, files)
		d, err := g.fetcher("").Fetch(context.Background(), testRepo)
		if err != nil || len(d.Backgrounds) != MaxBackgrounds {
			t.Errorf("backgrounds = %d, %v", len(d.Backgrounds), err)
		}
	})
}
