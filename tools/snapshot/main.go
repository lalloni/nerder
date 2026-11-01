// Command snapshot regenerates internal/catalog/stats.json: for every family,
// the glyph count of its preview font, the download count of its release
// archive, the GitHub stars of its upstream project (from
// internal/catalog/upstream.json) and its Google Fonts popularity rank.
//
// Glyph counts come from the release archives over HTTP range requests, so
// only the first part of one font per archive is transferred.
package main

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"nerder/internal/catalog"
	"nerder/internal/fetch"
	"nerder/internal/sfnt"
)

const block = 256 << 10

// rangeReader is an io.ReaderAt over an HTTP resource, fetched in cached blocks.
type rangeReader struct {
	url    string
	size   int64
	mu     sync.Mutex
	blocks map[int64][]byte
}

func newRangeReader(url string) (*rangeReader, error) {
	resp, err := http.Head(url)
	if err != nil {
		return nil, err
	}
	resp.Body.Close()
	if resp.StatusCode != 200 || resp.ContentLength <= 0 {
		return nil, fmt.Errorf("HEAD %s: %s", url, resp.Status)
	}
	return &rangeReader{url: url, size: resp.ContentLength, blocks: map[int64][]byte{}}, nil
}

func (r *rangeReader) block(i int64) ([]byte, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if b, ok := r.blocks[i]; ok {
		return b, nil
	}
	start, end := i*block, min((i+1)*block, r.size)-1
	req, _ := http.NewRequest("GET", r.url, nil)
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", start, end))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusPartialContent {
		return nil, fmt.Errorf("range request got %s", resp.Status)
	}
	b, err := io.ReadAll(resp.Body)
	r.blocks[i] = b
	return b, err
}

func (r *rangeReader) ReadAt(p []byte, off int64) (int, error) {
	n := 0
	for n < len(p) {
		if off+int64(n) >= r.size {
			return n, io.EOF
		}
		b, err := r.block((off + int64(n)) / block)
		if err != nil {
			return n, err
		}
		n += copy(p[n:], b[(off+int64(n))%block:])
	}
	return n, nil
}

func glyphs(folder string) (int, error) {
	r, err := newRangeReader(fmt.Sprintf("https://github.com/ryanoasis/nerd-fonts/releases/download/%s/%s.zip", catalog.ReleaseTag, folder))
	if err != nil {
		return 0, err
	}
	zr, err := zip.NewReader(r, r.size)
	if err != nil {
		return 0, err
	}
	var names []string
	for _, f := range zr.File {
		names = append(names, f.Name)
	}
	sort.Strings(names)
	regular, _ := fetch.PreviewFiles(names)
	if regular == "" {
		return 0, fmt.Errorf("no preview file among %d entries", len(names))
	}
	f, err := zr.Open(regular)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	return sfnt.GlyphCountReader(f)
}

func stars(repo string) (int, error) {
	var v struct {
		Stars int `json:"stargazers_count"`
	}
	err := githubGet("https://api.github.com/repos/"+repo, &v)
	return v.Stars, err
}

func githubGet(url string, v any) error {
	req, _ := http.NewRequest("GET", url, nil)
	if t := os.Getenv("GITHUB_TOKEN"); t != "" {
		req.Header.Set("Authorization", "Bearer "+t)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("%s: %s", url, resp.Status)
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

// downloads returns the download count of each .zip asset of the pinned release, by folder name.
func downloads() (map[string]int, error) {
	var rel struct {
		Assets []struct {
			Name  string `json:"name"`
			Count int    `json:"download_count"`
		} `json:"assets"`
	}
	if err := githubGet("https://api.github.com/repos/ryanoasis/nerd-fonts/releases/tags/"+catalog.ReleaseTag, &rel); err != nil {
		return nil, err
	}
	out := map[string]int{}
	for _, a := range rel.Assets {
		if name, ok := strings.CutSuffix(a.Name, ".zip"); ok {
			out[name] = a.Count
		}
	}
	return out, nil
}

var nonAlnum = regexp.MustCompile(`[^a-z0-9]`)

func key(s string) string { return nonAlnum.ReplaceAllString(strings.ToLower(s), "") }

// googleAliases maps families whose Google Fonts name differs from every name the catalog knows.
var googleAliases = map[string]string{"Noto": "Noto Sans Mono"}

// googleRanks returns the Google Fonts popularity rank (1 = most popular) of
// each family that Google Fonts hosts, by normalised family name.
func googleRanks() (map[string]int, error) {
	resp, err := http.Get("https://fonts.google.com/metadata/fonts")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if i := strings.IndexByte(string(body), '{'); i > 0 { // skip the anti-XSSI prefix
		body = body[i:]
	}
	var meta struct {
		Families []struct {
			Family     string `json:"family"`
			Popularity int    `json:"popularity"`
		} `json:"familyMetadataList"`
	}
	if err := json.Unmarshal(body, &meta); err != nil {
		return nil, err
	}
	out := map[string]int{}
	for _, f := range meta.Families {
		out[key(f.Family)] = f.Popularity
	}
	return out, nil
}

func main() {
	cat, err := catalog.Load()
	if err != nil {
		log.Fatal(err)
	}
	raw, err := os.ReadFile("internal/catalog/upstream.json")
	if err != nil {
		log.Fatal(err)
	}
	var upstream map[string]string
	if err := json.Unmarshal(raw, &upstream); err != nil {
		log.Fatal(err)
	}

	type stat struct {
		Glyphs    int `json:"glyphs"`
		Downloads int `json:"downloads,omitempty"`
		Stars     int `json:"stars,omitempty"`
		GFRank    int `json:"gfRank,omitempty"`
	}
	dl, err := downloads()
	if err != nil {
		log.Fatal(err)
	}
	ranks, err := googleRanks()
	if err != nil {
		log.Fatal(err)
	}
	out := struct {
		AsOf  string          `json:"asOf"`
		Stats map[string]stat `json:"stats"`
	}{AsOf: time.Now().Format("2006-01-02"), Stats: map[string]stat{}}

	repoStars := map[string]int{}
	failed := false
	for _, f := range cat.Families() {
		s := stat{}
		if s.Glyphs, err = glyphs(f.Folder); err != nil {
			log.Printf("glyphs %s: %v", f.Folder, err)
			failed = true
		}
		if repo := upstream[f.Folder]; repo != "" {
			n, seen := repoStars[repo]
			if !seen {
				if n, err = stars(repo); err != nil {
					log.Printf("stars %s: %v", f.Folder, err)
					failed = true
				}
				repoStars[repo] = n
			}
			s.Stars = n
		}
		s.Downloads = dl[f.Folder]
		names := []string{f.Name, f.PatchedName, f.Folder}
		if alias, ok := googleAliases[f.Folder]; ok {
			names = []string{alias}
		}
		for _, n := range names {
			if r, ok := ranks[key(n)]; ok {
				s.GFRank = r
				break
			}
		}
		out.Stats[f.Folder] = s
		fmt.Fprintf(os.Stderr, "%-26s glyphs=%-6d downloads=%-7d stars=%-6d gf=%d\n", f.Folder, s.Glyphs, s.Downloads, s.Stars, s.GFRank)
	}
	if failed {
		log.Fatal("some lookups failed; not writing stats.json")
	}
	b, _ := json.MarshalIndent(out, "", "  ")
	if err := os.WriteFile("internal/catalog/stats.json", append(b, '\n'), 0o644); err != nil {
		log.Fatal(err)
	}
}
