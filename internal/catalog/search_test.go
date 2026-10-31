package catalog

import (
	"strings"
	"testing"
)

func TestFuzzy(t *testing.T) {
	if _, ok := Fuzzy("jbm", "JetBrains Mono"); !ok {
		t.Error("jbm should match JetBrains Mono")
	}
	if _, ok := Fuzzy("xyz", "Hack"); ok {
		t.Error("xyz should not match Hack")
	}
	if _, ok := Fuzzy("", "anything"); !ok {
		t.Error("empty pattern should match")
	}
	tight, _ := Fuzzy("hack", "Hack")
	loose, _ := Fuzzy("hack", "xhxaxcxk")
	if tight <= loose {
		t.Errorf("contiguous match (%d) should outscore scattered match (%d)", tight, loose)
	}
}

func TestSearch(t *testing.T) {
	c := load(t)

	res := c.Search("fira", "", "")
	if len(res) < 2 || !strings.Contains(res[0].Name, "Fira") || !strings.Contains(res[1].Name, "Fira") {
		t.Fatalf("Search(fira) top results = %v", names(res))
	}

	non := c.Search("", "nonmono", "")
	if len(non) != 5 {
		t.Errorf("nonmono count = %d, want 5", len(non))
	}
	for _, f := range non {
		if f.Monospaced {
			t.Errorf("%s is monospaced", f.Name)
		}
	}

	mit := c.Search("", "", "MIT")
	if len(mit) == 0 {
		t.Fatal("no MIT families")
	}
	for _, f := range mit {
		if f.License != "MIT" {
			t.Errorf("%s has license %q", f.Name, f.License)
		}
	}

	none := c.Search("zzzzqqq", "", "")
	if none == nil || len(none) != 0 {
		t.Errorf("no-match search = %#v, want empty non-nil slice", none)
	}

	if all := c.Search("", "", ""); len(all) != 72 {
		t.Errorf("unfiltered count = %d, want 72", len(all))
	}
}

func names(fs []Family) []string {
	var out []string
	for _, f := range fs {
		out = append(out, f.Name)
	}
	return out
}
