// Package catalog holds the embedded Nerd Fonts family list and knows which
// installed font files belong to which family.
package catalog

import (
	_ "embed"
	"encoding/json"
	"path/filepath"
	"sort"
	"strings"
)

// ReleaseTag is the pinned Nerd Fonts release that fonts.json and the
// downloaded archives come from.
const ReleaseTag = "v3.5.1"

// maxGap is how many characters may sit between a patched name and "NerdFont"
// (the NL variant marker, or Meslo's S/M/L/SDZ size letters).
const maxGap = 3

//go:embed fonts.json
var fontsJSON []byte

type Family struct {
	Name        string `json:"name"`        // unpatchedName, shown to the user
	PatchedName string `json:"patchedName"` // prefix of the installed file names
	Folder      string `json:"folder"`      // folderName; archive is <Folder>.zip
	License     string `json:"license"`     // licenseId
	Version     string `json:"version"`
	Monospaced  bool   `json:"monospaced"`
	Description string `json:"description"`
}

type prefix struct {
	norm   string
	folder string
}

type Catalog struct {
	families []Family
	byFolder map[string]Family
	prefixes []prefix // longest first
}

// Load parses the embedded snapshot.
func Load() (*Catalog, error) {
	var raw struct {
		Fonts []struct {
			UnpatchedName string `json:"unpatchedName"`
			PatchedName   string `json:"patchedName"`
			FolderName    string `json:"folderName"`
			LicenseID     string `json:"licenseId"`
			Version       string `json:"version"`
			IsMonospaced  bool   `json:"isMonospaced"`
			Description   string `json:"description"`
		} `json:"fonts"`
	}
	if err := json.Unmarshal(fontsJSON, &raw); err != nil {
		return nil, err
	}
	c := &Catalog{byFolder: map[string]Family{}}
	for _, f := range raw.Fonts {
		fam := Family{
			Name: f.UnpatchedName, PatchedName: f.PatchedName, Folder: f.FolderName,
			License: f.LicenseID, Version: f.Version, Monospaced: f.IsMonospaced,
			Description: f.Description,
		}
		c.families = append(c.families, fam)
		c.byFolder[fam.Folder] = fam
		c.prefixes = append(c.prefixes, prefix{normalise(f.PatchedName), f.FolderName})
	}
	sort.SliceStable(c.families, func(i, j int) bool {
		return strings.ToLower(c.families[i].Name) < strings.ToLower(c.families[j].Name)
	})
	sort.SliceStable(c.prefixes, func(i, j int) bool {
		return len(c.prefixes[i].norm) > len(c.prefixes[j].norm)
	})
	return c, nil
}

// Families returns all families sorted by name, case-insensitively.
func (c *Catalog) Families() []Family {
	return append([]Family(nil), c.families...)
}

func (c *Catalog) Find(folder string) (Family, bool) {
	f, ok := c.byFolder[folder]
	return f, ok
}

// Licenses returns the distinct license ids, sorted.
func (c *Catalog) Licenses() []string {
	set := map[string]bool{}
	for _, f := range c.families {
		set[f.License] = true
	}
	out := make([]string, 0, len(set))
	for l := range set {
		out = append(out, l)
	}
	sort.Strings(out)
	return out
}

// OwnerOfFile reports which family an installed font file belongs to. Files
// are named <PatchedName>[NL|S|M|L|...]NerdFont[Mono|Propo]-<Style>.<ttf|otf>;
// the longest matching patched name wins, and "NerdFont" must follow it within
// maxGap characters, so IosevkaTerm files never count as Iosevka.
func (c *Catalog) OwnerOfFile(filename string) (folder string, ok bool) {
	ext := strings.ToLower(filepath.Ext(filename))
	if ext != ".ttf" && ext != ".otf" {
		return "", false
	}
	name := normalise(strings.TrimSuffix(filepath.Base(filename), filepath.Ext(filename)))
	for _, p := range c.prefixes {
		rest, found := strings.CutPrefix(name, p.norm)
		if i := strings.Index(rest, "nerdfont"); found && i >= 0 && i <= maxGap {
			return p.folder, true
		}
	}
	return "", false
}

// normalise lowercases and drops spaces; "+" becomes "plus" (M+ -> mplus).
func normalise(s string) string {
	s = strings.ReplaceAll(strings.ToLower(s), "+", "plus")
	return strings.ReplaceAll(s, " ", "")
}
