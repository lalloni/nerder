package installer

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nerder/internal/catalog"
)

type env struct {
	in      *Installer
	user    string
	sys     string
	refresh int
	failing bool
}

func newEnv(t *testing.T) *env {
	t.Helper()
	cat, err := catalog.Load()
	if err != nil {
		t.Fatal(err)
	}
	e := &env{user: filepath.Join(t.TempDir(), "fonts"), sys: t.TempDir()}
	e.in = New(cat)
	e.in.UserDir = e.user
	e.in.SystemDirs = []string{e.sys}
	e.in.RefreshCache = func(string) error {
		e.refresh++
		if e.failing {
			return errors.New("fc-cache exploded")
		}
		return nil
	}
	return e
}

func touch(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("font"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func exists(path string) bool { _, err := os.Stat(path); return err == nil }

func srcDir(t *testing.T, names ...string) string {
	dir := t.TempDir()
	for _, n := range names {
		touch(t, filepath.Join(dir, n))
	}
	return dir
}

func status(t *testing.T, e *env, folder string) Status {
	t.Helper()
	m, err := e.in.Statuses()
	if err != nil {
		t.Fatal(err)
	}
	if s, ok := m[folder]; ok {
		return s
	}
	return NotInstalled
}

func TestInstallCopiesAndRefreshes(t *testing.T) {
	e := newEnv(t)
	src := srcDir(t, "HackNerdFont-Regular.ttf", "HackNerdFont-Bold.ttf")
	if err := e.in.Install("Hack", src); err != nil {
		t.Fatal(err)
	}
	for _, n := range []string{"HackNerdFont-Regular.ttf", "HackNerdFont-Bold.ttf"} {
		if !exists(filepath.Join(e.user, "NerdFonts", "Hack", n)) {
			t.Errorf("%s not installed", n)
		}
	}
	if status(t, e, "Hack") != UserInstalled || e.refresh != 1 {
		t.Errorf("status=%v refresh=%d", status(t, e, "Hack"), e.refresh)
	}
}

func TestInstallIsIdempotent(t *testing.T) {
	e := newEnv(t)
	src := srcDir(t, "HackNerdFont-Regular.ttf")
	for i := 0; i < 2; i++ {
		if err := e.in.Install("Hack", src); err != nil {
			t.Fatal(err)
		}
	}
	entries, _ := os.ReadDir(filepath.Join(e.user, "NerdFonts", "Hack"))
	if len(entries) != 1 {
		t.Errorf("%d files after double install, want 1", len(entries))
	}
}

func TestInstallLeavesNoPartialFiles(t *testing.T) {
	e := newEnv(t)
	src := srcDir(t, "HackNerdFont-Regular.ttf", "HackNerdFont-Bold.ttf")
	os.Chmod(filepath.Join(src, "HackNerdFont-Regular.ttf"), 0) // unreadable: copy fails
	if os.Geteuid() == 0 {
		t.Skip("root can read anything")
	}
	if err := e.in.Install("Hack", src); err == nil {
		t.Fatal("expected an error")
	}
	filepath.Walk(e.user, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			t.Errorf("leftover file after failed install: %s", p)
		}
		return nil
	})
}

func TestStatusDetectsHandInstalled(t *testing.T) {
	e := newEnv(t)
	touch(t, filepath.Join(e.user, "JetBrainsMonoNerdFont-Regular.ttf"))
	touch(t, filepath.Join(e.sys, "deep", "FiraCodeNerdFont-Regular.ttf"))
	touch(t, filepath.Join(e.user, "FiraCodeNerdFont-Bold.ttf"))
	touch(t, filepath.Join(e.sys, "HackNerdFont-Regular.ttf"))
	if s := status(t, e, "JetBrainsMono"); s != UserInstalled {
		t.Errorf("JetBrainsMono = %v", s)
	}
	if s := status(t, e, "FiraCode"); s != UserInstalled {
		t.Errorf("FiraCode (user and system) = %v, want user", s)
	}
	if s := status(t, e, "Hack"); s != SystemInstalled {
		t.Errorf("Hack = %v", s)
	}
}

func TestStatusSimilarNames(t *testing.T) {
	e := newEnv(t)
	touch(t, filepath.Join(e.user, "IosevkaTermNerdFont-Regular.ttf"))
	if s := status(t, e, "Iosevka"); s != NotInstalled {
		t.Errorf("Iosevka = %v, want none", s)
	}
	if s := status(t, e, "IosevkaTerm"); s != UserInstalled {
		t.Errorf("IosevkaTerm = %v", s)
	}
}

func TestUninstallRemovesOnlyThatFamily(t *testing.T) {
	e := newEnv(t)
	nested := filepath.Join(e.user, "NerdFonts", "Iosevka", "IosevkaNerdFont-Regular.ttf")
	flat := filepath.Join(e.user, "IosevkaNerdFontMono-Bold.ttf")
	term := filepath.Join(e.user, "NerdFonts", "IosevkaTerm", "IosevkaTermNerdFont-Regular.ttf")
	hack := filepath.Join(e.user, "HackNerdFont-Regular.ttf")
	for _, p := range []string{nested, flat, term, hack} {
		touch(t, p)
	}
	if err := e.in.Uninstall("Iosevka"); err != nil {
		t.Fatal(err)
	}
	if exists(nested) || exists(flat) {
		t.Error("Iosevka files remain")
	}
	if !exists(term) || !exists(hack) {
		t.Error("another family's files were removed")
	}
	if exists(filepath.Dir(nested)) {
		t.Error("emptied family dir remains")
	}
	if e.refresh != 1 {
		t.Errorf("refresh = %d", e.refresh)
	}
}

func TestUninstallNotInstalledIsNoop(t *testing.T) {
	e := newEnv(t)
	if err := e.in.Uninstall("Hack"); err != nil {
		t.Fatal(err)
	}
}

func TestRefreshFailureIsReported(t *testing.T) {
	e := newEnv(t)
	e.failing = true
	if err := e.in.Install("Hack", srcDir(t, "HackNerdFont-Regular.ttf")); !errors.Is(err, ErrCacheRefresh) {
		t.Fatalf("Install err = %v, want ErrCacheRefresh", err)
	}
	if status(t, e, "Hack") != UserInstalled {
		t.Error("files should stay installed when the cache refresh fails")
	}
	if err := e.in.Uninstall("Hack"); !errors.Is(err, ErrCacheRefresh) {
		t.Fatalf("Uninstall err = %v, want ErrCacheRefresh", err)
	}
	if status(t, e, "Hack") != NotInstalled {
		t.Error("files should be gone even when the cache refresh fails")
	}
}

func TestDefaultRefreshMissingBinary(t *testing.T) {
	t.Setenv("PATH", "")
	_, _, refresh := defaultsFor("linux", t.TempDir(), "")
	err := refresh(t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "fc-cache") {
		t.Errorf("err = %v, want mention of fc-cache", err)
	}
}

func TestDefaultsFor(t *testing.T) {
	user, system, refresh := defaultsFor("linux", "/home/me", "")
	if user != "/home/me/.local/share/fonts" || len(system) != 2 || refresh == nil {
		t.Errorf("linux = %q %q refresh=%v", user, system, refresh != nil)
	}
	if user, _, _ := defaultsFor("linux", "/home/me", "/data"); user != "/data/fonts" {
		t.Errorf("XDG_DATA_HOME ignored: %q", user)
	}
	user, system, refresh = defaultsFor("darwin", "/Users/me", "/ignored")
	if user != "/Users/me/Library/Fonts" || system[0] != "/Library/Fonts" || refresh != nil {
		t.Errorf("darwin = %q %q refresh=%v", user, system, refresh != nil)
	}
}

func TestInstallWithoutCacheRefresh(t *testing.T) {
	e := newEnv(t)
	e.in.RefreshCache = nil // macOS
	if err := e.in.Install("Hack", srcDir(t, "HackNerdFont-Regular.ttf")); err != nil {
		t.Fatalf("Install without a refresh hook: %v", err)
	}
	if err := e.in.Uninstall("Hack"); err != nil {
		t.Fatalf("Uninstall without a refresh hook: %v", err)
	}
}
