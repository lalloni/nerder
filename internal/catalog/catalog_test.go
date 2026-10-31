package catalog

import (
	"sort"
	"strings"
	"testing"
)

func load(t *testing.T) *Catalog {
	t.Helper()
	c, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	return c
}

func TestLoad(t *testing.T) {
	c := load(t)
	fams := c.Families()
	if len(fams) != 72 {
		t.Fatalf("got %d families, want 72", len(fams))
	}
	jb, ok := c.Find("JetBrainsMono")
	if !ok || !jb.Monospaced || jb.License == "" {
		t.Fatalf("JetBrainsMono = %+v, ok=%v", jb, ok)
	}
	if !sort.SliceIsSorted(fams, func(i, j int) bool {
		return strings.ToLower(fams[i].Name) < strings.ToLower(fams[j].Name)
	}) {
		t.Error("Families() is not sorted by name")
	}
	seen := map[string]bool{}
	for _, l := range c.Licenses() {
		if seen[l] {
			t.Errorf("duplicate license %q", l)
		}
		seen[l] = true
	}
	if len(seen) == 0 {
		t.Error("no licenses")
	}
}

func TestOwnerOfFile(t *testing.T) {
	c := load(t)
	tests := []struct{ file, want string }{
		{"JetBrainsMonoNerdFont-Regular.ttf", "JetBrainsMono"},
		{"JetBrainsMonoNLNerdFont-Regular.ttf", "JetBrainsMono"},
		{"IosevkaNerdFontMono-Bold.ttf", "Iosevka"},
		{"IosevkaTermNerdFont-Regular.ttf", "IosevkaTerm"},
		{"IosevkaTermSlabNerdFontPropo-Italic.ttf", "IosevkaTermSlab"},
		{"MesloLGSNerdFont-Regular.ttf", "Meslo"},
		{"CaskaydiaCoveNerdFont-Regular.ttf", "CascadiaCode"},
		{"HackNerdFontPropo-Italic.otf", "Hack"},
		{"hacknerdfont-regular.TTF", "Hack"},
		{"Arial.ttf", ""},
		{"README.md", ""},
		{"HackNerdFont", ""},
	}
	for _, tt := range tests {
		got, ok := c.OwnerOfFile(tt.file)
		if got != tt.want || ok != (tt.want != "") {
			t.Errorf("OwnerOfFile(%q) = %q, %v; want %q", tt.file, got, ok, tt.want)
		}
	}
}
