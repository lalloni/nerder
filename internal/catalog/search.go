package catalog

import (
	"sort"
	"strings"
	"unicode"
)

// Fuzzy reports whether pattern is a case-insensitive subsequence of
// candidate, with a score that rewards consecutive runs and matches at the
// start of a word (or at a lower-to-upper camel-case boundary).
func Fuzzy(pattern, candidate string) (score int, ok bool) {
	pat := []rune(strings.ToLower(pattern))
	cand := []rune(candidate)
	if len(pat) == 0 {
		return 0, true
	}
	pi, prev := 0, -2
	for i, r := range cand {
		if pi >= len(pat) {
			break
		}
		if unicode.ToLower(r) != pat[pi] {
			continue
		}
		score++
		if i == prev+1 {
			score += 5
		}
		if i == 0 || !unicode.IsLetter(cand[i-1]) && !unicode.IsDigit(cand[i-1]) ||
			unicode.IsLower(cand[i-1]) && unicode.IsUpper(r) {
			score += 8
		}
		prev = i
		pi++
	}
	if pi < len(pat) {
		return 0, false
	}
	return score, true
}

// Search filters and ranks families. mono is "", "mono" or "nonmono"; license
// is "" or an exact licenseId. With empty text the alphabetical order is kept;
// otherwise results are ordered by descending score, ties alphabetical.
func (c *Catalog) Search(text, mono, license string) []Family {
	type scored struct {
		f     Family
		score int
	}
	out := []scored{}
	for _, f := range c.families {
		if mono == "mono" && !f.Monospaced || mono == "nonmono" && f.Monospaced {
			continue
		}
		if license != "" && f.License != license {
			continue
		}
		best, ok := Fuzzy(text, f.Name)
		if s, ok2 := Fuzzy(text, f.PatchedName); ok2 && (!ok || s > best) {
			best, ok = s, true
		}
		if !ok {
			continue
		}
		out = append(out, scored{f, best})
	}
	if text != "" {
		sort.SliceStable(out, func(i, j int) bool { return out[i].score > out[j].score })
	}
	res := make([]Family, len(out))
	for i, s := range out {
		res[i] = s.f
	}
	return res
}
