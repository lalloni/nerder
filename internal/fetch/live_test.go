//go:build live

package fetch

import (
	"net/http"
	"testing"

	"nerder/internal/catalog"
)

// TestLiveArchivesExist checks that every family in the embedded catalog has
// a downloadable archive in the pinned release. Run with: go test -tags live
func TestLiveArchivesExist(t *testing.T) {
	cat, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	f := New(t.TempDir(), catalog.ReleaseTag)
	for _, fam := range cat.Families() {
		resp, err := http.Head(f.BaseURL + "/" + fam.Folder + ".zip")
		if err != nil {
			t.Errorf("%s: %v", fam.Folder, err)
			continue
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("%s: %s", fam.Folder, resp.Status)
		}
	}
}
