// Package fetch downloads Nerd Fonts family archives on demand and keeps the
// extracted font files in a disk cache.
package fetch

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

const (
	maxConcurrent = 3
	maxFontBytes  = 256 << 20
)

type call struct {
	done chan struct{}
	dir  string
	err  error
}

type Fetcher struct {
	CacheDir string       // fonts land in <CacheDir>/<tag>/<folder>/
	BaseURL  string       // archives live at <BaseURL>/<folder>.zip
	Client   *http.Client // defaults to a client with a generous timeout

	tag      string
	sem      chan struct{}
	mu       sync.Mutex
	inflight map[string]*call
}

func New(cacheDir, tag string) *Fetcher {
	return &Fetcher{
		CacheDir: cacheDir,
		BaseURL:  "https://github.com/ryanoasis/nerd-fonts/releases/download/" + tag,
		Client:   &http.Client{Timeout: 10 * time.Minute},
		tag:      tag,
		sem:      make(chan struct{}, maxConcurrent),
		inflight: map[string]*call{},
	}
}

func (f *Fetcher) dir(folder string) string { return filepath.Join(f.CacheDir, f.tag, folder) }

// Cached reports whether the family has already been downloaded.
func (f *Fetcher) Cached(folder string) (dir string, ok bool) {
	dir = f.dir(folder)
	files, err := FontFiles(dir)
	return dir, err == nil && len(files) > 0
}

// Ensure downloads and extracts the family unless it is cached. Concurrent
// calls for one family share a single download, and at most three downloads
// run at once.
func (f *Fetcher) Ensure(ctx context.Context, folder string) (string, error) {
	if dir, ok := f.Cached(folder); ok {
		return dir, nil
	}
	f.mu.Lock()
	c, running := f.inflight[folder]
	if !running {
		c = &call{done: make(chan struct{})}
		f.inflight[folder] = c
	}
	f.mu.Unlock()

	if running {
		select {
		case <-c.done:
			return c.dir, c.err
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
	c.dir, c.err = f.download(ctx, folder)
	f.mu.Lock()
	delete(f.inflight, folder)
	f.mu.Unlock()
	close(c.done)
	return c.dir, c.err
}

func (f *Fetcher) download(ctx context.Context, folder string) (string, error) {
	select {
	case f.sem <- struct{}{}:
		defer func() { <-f.sem }()
	case <-ctx.Done():
		return "", ctx.Err()
	}

	tagDir := filepath.Join(f.CacheDir, f.tag)
	if err := os.MkdirAll(tagDir, 0o755); err != nil {
		return "", err
	}
	tmp, err := os.CreateTemp(tagDir, folder+".*.zip")
	if err != nil {
		return "", err
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.BaseURL+"/"+folder+".zip", nil)
	if err != nil {
		return "", err
	}
	resp, err := f.Client.Do(req)
	if err != nil {
		return "", fmt.Errorf("downloading %s: %w", folder, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("downloading %s: %s", folder, resp.Status)
	}
	if _, err := io.Copy(tmp, resp.Body); err != nil {
		return "", fmt.Errorf("downloading %s: %w", folder, err)
	}

	partial := f.dir(folder) + ".partial"
	os.RemoveAll(partial)
	if err := os.MkdirAll(partial, 0o755); err != nil {
		return "", err
	}
	defer os.RemoveAll(partial)
	if err := extractFonts(tmp.Name(), partial); err != nil {
		return "", fmt.Errorf("extracting %s: %w", folder, err)
	}
	os.RemoveAll(f.dir(folder))
	if err := os.Rename(partial, f.dir(folder)); err != nil {
		return "", err
	}
	return f.dir(folder), nil
}

// extractFonts writes the .ttf/.otf entries of the archive into dir, flat,
// under their base names. Everything else is ignored.
func extractFonts(archive, dir string) error {
	zr, err := zip.OpenReader(archive)
	if err != nil && !errors.Is(err, zip.ErrInsecurePath) {
		return err
	}
	defer zr.Close()
	n := 0
	for _, e := range zr.File {
		name := filepath.Base(filepath.FromSlash(e.Name))
		if e.FileInfo().IsDir() || !isFont(name) || name == "." || name == ".." {
			continue
		}
		if err := extractOne(e, filepath.Join(dir, name)); err != nil {
			return err
		}
		n++
	}
	if n == 0 {
		return errors.New("archive contains no font files")
	}
	return nil
}

func extractOne(e *zip.File, dest string) error {
	rc, err := e.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, io.LimitReader(rc, maxFontBytes)); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func isFont(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	return ext == ".ttf" || ext == ".otf"
}

// FontFiles lists the sorted base names of the .ttf/.otf files in dir.
func FontFiles(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range entries {
		if !e.IsDir() && isFont(e.Name()) {
			out = append(out, e.Name())
		}
	}
	sort.Strings(out)
	return out, nil
}

// PreviewFiles picks the files used to render a family's card: the Regular
// file of the standard patched variant (not Mono or Propo), preferring the
// non-NL one, and its Bold sibling when one exists. files must be sorted.
func PreviewFiles(files []string) (regular, bold string) {
	has := strings.Contains
	pick := func(ok func(string) bool) string {
		for _, name := range files {
			if ok(name) {
				return name
			}
		}
		return ""
	}
	regular = pick(func(n string) bool { return has(n, "NerdFont-Regular") && !has(n, "NLNerdFont") })
	if regular == "" {
		regular = pick(func(n string) bool { return has(n, "NerdFont-Regular") })
	}
	if regular == "" {
		regular = pick(func(n string) bool {
			return has(n, "-Regular.") && !has(n, "Mono") && !has(n, "Propo")
		})
	}
	if regular == "" && len(files) > 0 {
		regular = files[0]
	}
	if want := strings.Replace(regular, "-Regular.", "-Bold.", 1); want != regular {
		bold = pick(func(n string) bool { return n == want })
	}
	return regular, bold
}
