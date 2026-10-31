package app

import (
	"archive/zip"
	"bytes"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"nerder/internal/catalog"
	"nerder/internal/fetch"
	"nerder/internal/installer"
)

func zipOf(t *testing.T, names ...string) []byte {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, n := range names {
		w, _ := zw.Create(n)
		w.Write([]byte("font-bytes"))
	}
	zw.Close()
	return buf.Bytes()
}

type fixture struct {
	app     *App
	refresh int
	failing bool
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		folder := strings.TrimSuffix(strings.TrimPrefix(r.URL.Path, "/"), ".zip")
		w.Write(zipOf(t, folder+"NerdFont-Regular.ttf", folder+"NerdFont-Bold.ttf", folder+"NerdFontMono-Regular.ttf"))
	}))
	t.Cleanup(srv.Close)
	cat, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	f := fetch.New(t.TempDir(), "test")
	f.BaseURL = srv.URL
	in := installer.New(cat)
	in.UserDir = filepath.Join(t.TempDir(), "fonts")
	in.SystemDirs = nil
	fx := &fixture{}
	in.RefreshCache = func(string) error {
		fx.refresh++
		if fx.failing {
			return errString("fc-cache exploded")
		}
		return nil
	}
	fx.app = New(cat, f, in)
	return fx
}

type errString string

func (e errString) Error() string { return string(e) }

func TestListFiltersByInstalledOnly(t *testing.T) {
	fx := newFixture(t)
	if _, err := fx.app.Install("Hack"); err != nil {
		t.Fatal(err)
	}
	got, err := fx.app.List(ListQuery{InstalledOnly: true})
	if err != nil || len(got) != 1 || got[0].Folder != "Hack" || got[0].Status != "user" {
		t.Fatalf("List(installedOnly) = %+v, %v", got, err)
	}
}

func TestListPassesSearchAndFilters(t *testing.T) {
	fx := newFixture(t)
	got, _ := fx.app.List(ListQuery{Text: "jetbr"})
	found := false
	for _, f := range got {
		found = found || f.Folder == "JetBrainsMono"
	}
	if !found {
		t.Error("jetbr should find JetBrainsMono")
	}
	got, _ = fx.app.List(ListQuery{Mono: "nonmono"})
	if len(got) != 5 {
		t.Errorf("nonmono = %d, want 5", len(got))
	}
	for _, f := range got {
		if f.Status != "none" {
			t.Errorf("%s status %q", f.Folder, f.Status)
		}
	}
}

func TestPreviewReturnsURLs(t *testing.T) {
	fx := newFixture(t)
	info, err := fx.app.Preview("Hack")
	if err != nil {
		t.Fatal(err)
	}
	if info.FontFamily != "nf-Hack" || info.RegularURL != "/fonts/Hack/HackNerdFont-Regular.ttf" || info.BoldURL != "/fonts/Hack/HackNerdFont-Bold.ttf" {
		t.Errorf("Preview = %+v", info)
	}
	if _, err := fx.app.Preview("NoSuchFont"); err == nil {
		t.Error("unknown folder should error")
	}
}

func TestFontHandlerServesCachedFileOnly(t *testing.T) {
	fx := newFixture(t)
	if _, err := fx.app.Preview("Hack"); err != nil {
		t.Fatal(err)
	}
	h := fx.app.FontHandler()
	get := func(path string) *httptest.ResponseRecorder {
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, httptest.NewRequest("GET", path, nil))
		return rec
	}
	ok := get("/fonts/Hack/HackNerdFont-Regular.ttf")
	if ok.Code != 200 || !strings.HasPrefix(ok.Header().Get("Content-Type"), "font/ttf") || ok.Body.String() != "font-bytes" {
		t.Errorf("good request: %d %q %q", ok.Code, ok.Header().Get("Content-Type"), ok.Body.String())
	}
	for _, bad := range []string{
		"/fonts/Hack/../../etc/passwd", "/fonts/NoSuchFont/x.ttf", "/fonts/Hack/Other.ttf",
		"/fonts/Hack", "/fonts/Fira/FiraNerdFont-Regular.ttf", "/other",
	} {
		if rec := get(bad); rec.Code != 404 {
			t.Errorf("GET %s = %d, want 404", bad, rec.Code)
		}
	}
}

func TestInstallThenUninstallRoundTrip(t *testing.T) {
	fx := newFixture(t)
	status := func() string {
		got, _ := fx.app.List(ListQuery{Text: "hack"})
		for _, f := range got {
			if f.Folder == "Hack" {
				return f.Status
			}
		}
		return "?"
	}
	if s := status(); s != "none" {
		t.Fatalf("before = %s", s)
	}
	v, err := fx.app.Install("Hack")
	if err != nil || v.Status != "user" {
		t.Fatalf("Install = %+v, %v", v, err)
	}
	if s := status(); s != "user" {
		t.Fatalf("after install = %s", s)
	}
	v, err = fx.app.Uninstall("Hack")
	if err != nil || v.Status != "none" {
		t.Fatalf("Uninstall = %+v, %v", v, err)
	}
}

func TestInstallRefreshFailureMessage(t *testing.T) {
	fx := newFixture(t)
	fx.failing = true
	_, err := fx.app.Install("Hack")
	if err == nil || !strings.Contains(err.Error(), "cache refresh") {
		t.Fatalf("err = %v, want mention of cache refresh", err)
	}
	got, _ := fx.app.List(ListQuery{InstalledOnly: true})
	if len(got) != 1 || got[0].Folder != "Hack" {
		t.Errorf("Hack should show as installed after a refresh failure, got %+v", got)
	}
}
