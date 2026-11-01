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

func TestNameNotes(t *testing.T) {
	c := load(t)
	cases := map[string][3]string{
		"0xProto":      {"0xProto Nerd Font", "", ""},
		"3270":         {"3270 Nerd Font", "Original", "IBM 3270"},
		"AdwaitaMono":  {"AdwaitaMono Nerd Font", "", ""}, // differs from "Adwaita Mono" only by a space
		"AnonymousPro": {"AnonymicePro Nerd Font", "Reserved", "Anonymous Pro"},
	}
	for folder, want := range cases {
		f, ok := c.Find(folder)
		if !ok {
			t.Fatalf("%s not found", folder)
		}
		if got := [3]string{f.Heading, f.NameKind, f.NameValue}; got != want {
			t.Errorf("%s = %q, want %q", folder, got, want)
		}
	}
}

func TestStatsSnapshot(t *testing.T) {
	c := load(t)
	for _, f := range c.Families() {
		if f.Glyphs < 5000 {
			t.Errorf("%s glyph count = %d, want a patched font's thousands", f.Name, f.Glyphs)
		}
	}
	for _, f := range c.Families() {
		if f.Downloads == 0 {
			t.Errorf("%s has no download count", f.Name)
		}
	}
	hack, _ := c.Find("Hack")
	if hack.Stars < 1000 || hack.GFRank != 0 {
		t.Errorf("Hack stars = %d, Google Fonts rank = %d (not hosted there)", hack.Stars, hack.GFRank)
	}
	if ubuntu, _ := c.Find("Ubuntu"); ubuntu.Stars != 0 || ubuntu.GFRank == 0 {
		t.Errorf("Ubuntu stars = %d (no upstream mapping), Google Fonts rank = %d", ubuntu.Stars, ubuntu.GFRank)
	}
}
