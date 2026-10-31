package fetch

import (
	"archive/zip"
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func makeZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		w.Write([]byte(body))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func newFetcher(t *testing.T, h http.Handler) (*Fetcher, string) {
	t.Helper()
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	cache := t.TempDir()
	f := New(cache, "v0.0.0")
	f.BaseURL = srv.URL
	return f, cache
}

func serveZip(t *testing.T, body []byte, hits *atomic.Int32) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits != nil {
			hits.Add(1)
		}
		w.Write(body)
	})
}

func TestEnsureDownloadsAndExtracts(t *testing.T) {
	var hits atomic.Int32
	f, _ := newFetcher(t, serveZip(t, makeZip(t, map[string]string{
		"FooNerdFont-Regular.ttf":         "r",
		"FooNerdFont-Bold.ttf":            "b",
		"LICENSE":                         "l",
		"README.md":                       "m",
		"sub/FooNerdFontMono-Regular.otf": "o",
	}), &hits))
	dir, err := f.Ensure(context.Background(), "Foo")
	if err != nil {
		t.Fatal(err)
	}
	files, _ := FontFiles(dir)
	want := []string{"FooNerdFont-Bold.ttf", "FooNerdFont-Regular.ttf", "FooNerdFontMono-Regular.otf"}
	if !reflect.DeepEqual(files, want) {
		t.Fatalf("files = %v, want %v", files, want)
	}
	if _, err := f.Ensure(context.Background(), "Foo"); err != nil {
		t.Fatal(err)
	}
	if hits.Load() != 1 {
		t.Errorf("server hit %d times, want 1", hits.Load())
	}
	if d, ok := f.Cached("Foo"); !ok || d != dir {
		t.Errorf("Cached = %q, %v", d, ok)
	}
}

func TestEnsureRejectsTraversal(t *testing.T) {
	f, cache := newFetcher(t, serveZip(t, makeZip(t, map[string]string{
		"../../evil.ttf": "x", "ok-Regular.ttf": "y",
	}), nil))
	dir, err := f.Ensure(context.Background(), "Foo")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(cache, "..", "evil.ttf")); err == nil {
		t.Error("file escaped the cache dir")
	}
	if _, err := os.Stat(filepath.Join(cache, "evil.ttf")); err == nil {
		t.Error("file escaped the family dir")
	}
	files, _ := FontFiles(dir)
	for _, name := range files {
		if filepath.Base(name) != name {
			t.Errorf("non-flat name %q", name)
		}
	}
}

func TestEnsureFailureLeavesNoCache(t *testing.T) {
	for name, h := range map[string]http.Handler{
		"500":      http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { http.Error(w, "boom", 500) }),
		"corrupt":  http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("not a zip")) }),
		"no fonts": serveZip(t, makeZip(t, map[string]string{"README.md": "x"}), nil),
	} {
		t.Run(name, func(t *testing.T) {
			f, cache := newFetcher(t, h)
			if _, err := f.Ensure(context.Background(), "Foo"); err == nil {
				t.Fatal("expected an error")
			}
			if _, ok := f.Cached("Foo"); ok {
				t.Error("failed download is reported as cached")
			}
			var leftovers []string
			filepath.Walk(cache, func(p string, info os.FileInfo, err error) error {
				if err == nil && !info.IsDir() {
					leftovers = append(leftovers, p)
				}
				return nil
			})
			if len(leftovers) != 0 {
				t.Errorf("leftover files: %v", leftovers)
			}
		})
	}
}

func TestEnsureDeduplicatesConcurrentCalls(t *testing.T) {
	var hits atomic.Int32
	body := makeZip(t, map[string]string{"FooNerdFont-Regular.ttf": "r"})
	f, _ := newFetcher(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		time.Sleep(50 * time.Millisecond)
		w.Write(body)
	}))
	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := f.Ensure(context.Background(), "Foo"); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if hits.Load() != 1 {
		t.Errorf("server hit %d times, want 1", hits.Load())
	}
}

func TestEnsureCapsConcurrency(t *testing.T) {
	var cur, max atomic.Int32
	body := makeZip(t, map[string]string{"FooNerdFont-Regular.ttf": "r"})
	f, _ := newFetcher(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := cur.Add(1)
		for {
			m := max.Load()
			if n <= m || max.CompareAndSwap(m, n) {
				break
			}
		}
		time.Sleep(40 * time.Millisecond)
		cur.Add(-1)
		w.Write(body)
	}))
	var wg sync.WaitGroup
	for _, folder := range []string{"A", "B", "C", "D", "E", "F", "G", "H"} {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := f.Ensure(context.Background(), folder); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	if max.Load() > 3 {
		t.Errorf("saw %d simultaneous downloads, want at most 3", max.Load())
	}
}

func TestEnsureHonoursContext(t *testing.T) {
	f, _ := newFetcher(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
		case <-time.After(5 * time.Second):
		}
	}))
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	start := time.Now()
	if _, err := f.Ensure(ctx, "Foo"); err == nil {
		t.Fatal("expected an error")
	}
	if time.Since(start) > 2*time.Second {
		t.Error("Ensure did not return promptly after cancellation")
	}
}

func TestPreviewFiles(t *testing.T) {
	tests := []struct {
		name          string
		files         []string
		regular, bold string
	}{
		{"prefers plain over Mono and Propo",
			[]string{"HackNerdFont-Bold.ttf", "HackNerdFont-BoldItalic.ttf", "HackNerdFont-Regular.ttf", "HackNerdFontMono-Regular.ttf", "HackNerdFontPropo-Regular.ttf"},
			"HackNerdFont-Regular.ttf", "HackNerdFont-Bold.ttf"},
		{"prefers non-NL",
			[]string{"JBNLNerdFont-Regular.ttf", "JBNerdFont-Regular.ttf"},
			"JBNerdFont-Regular.ttf", ""},
		{"falls back to any Regular", []string{"Foo-Regular.otf", "FooMono-Regular.otf"}, "Foo-Regular.otf", ""},
		{"falls back to first file", []string{"A-Light.ttf", "B-Thin.ttf"}, "A-Light.ttf", ""},
		{"empty", nil, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, b := PreviewFiles(tt.files)
			if r != tt.regular || b != tt.bold {
				t.Errorf("got (%q, %q), want (%q, %q)", r, b, tt.regular, tt.bold)
			}
		})
	}
}
