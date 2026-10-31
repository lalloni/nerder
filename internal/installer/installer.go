// Package installer puts downloaded Nerd Fonts into the user's fonts
// directory, removes them again, and works out which families are installed.
package installer

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"nerder/internal/catalog"
	"nerder/internal/fetch"
)

type Status string

const (
	NotInstalled    Status = "none"
	UserInstalled   Status = "user"
	SystemInstalled Status = "system"
)

// ErrCacheRefresh is wrapped when the font files were changed but refreshing
// the fontconfig cache failed.
var ErrCacheRefresh = errors.New("font cache refresh failed")

type Installer struct {
	UserDir      string                 // $XDG_DATA_HOME/fonts, else ~/.local/share/fonts
	SystemDirs   []string               // /usr/share/fonts, /usr/local/share/fonts
	RefreshCache func(dir string) error // runs fc-cache -f <dir>

	cat *catalog.Catalog
}

func New(cat *catalog.Catalog) *Installer {
	data := os.Getenv("XDG_DATA_HOME")
	if data == "" {
		home, _ := os.UserHomeDir()
		data = filepath.Join(home, ".local", "share")
	}
	return &Installer{
		UserDir:      filepath.Join(data, "fonts"),
		SystemDirs:   []string{"/usr/share/fonts", "/usr/local/share/fonts"},
		RefreshCache: fcCache,
		cat:          cat,
	}
}

func fcCache(dir string) error {
	out, err := exec.Command("fc-cache", "-f", dir).CombinedOutput()
	if err != nil {
		return fmt.Errorf("fc-cache: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}

// owned walks root and returns the font files each family owns, by folder.
// Missing or unreadable directories are skipped.
func (i *Installer) owned(root string) map[string][]string {
	out := map[string][]string{}
	filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if folder, ok := i.cat.OwnerOfFile(d.Name()); ok {
			out[folder] = append(out[folder], path)
		}
		return nil
	})
	return out
}

// Statuses reports, by family folder, where each installed family lives.
// Families that are not installed are absent from the map.
func (i *Installer) Statuses() (map[string]Status, error) {
	res := map[string]Status{}
	for _, dir := range i.SystemDirs {
		for folder := range i.owned(dir) {
			res[folder] = SystemInstalled
		}
	}
	for folder := range i.owned(i.UserDir) {
		res[folder] = UserInstalled
	}
	return res, nil
}

// Install copies the font files in srcDir to <UserDir>/NerdFonts/<folder>/
// and refreshes the font cache. Files are staged under temporary names first,
// so a failed copy leaves nothing behind.
func (i *Installer) Install(folder, srcDir string) error {
	if _, ok := i.cat.Find(folder); !ok {
		return fmt.Errorf("unknown font family %q", folder)
	}
	files, err := fetch.FontFiles(srcDir)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		return fmt.Errorf("no font files in %s", srcDir)
	}
	dest := filepath.Join(i.UserDir, "NerdFonts", folder)
	if err := os.MkdirAll(dest, 0o755); err != nil {
		return err
	}
	var staged []string
	cleanup := func() {
		for _, t := range staged {
			os.Remove(t)
		}
		os.Remove(dest) // only succeeds if empty
	}
	for _, name := range files {
		tmp := filepath.Join(dest, "."+name+".tmp")
		staged = append(staged, tmp)
		if err := copyFile(filepath.Join(srcDir, name), tmp); err != nil {
			cleanup()
			return err
		}
	}
	for n, name := range files {
		if err := os.Rename(staged[n], filepath.Join(dest, name)); err != nil {
			cleanup()
			return err
		}
	}
	return i.refresh()
}

// Uninstall removes every file the family owns in the user fonts directory,
// wherever it sits, and refreshes the font cache. A family that is not
// installed is left alone.
func (i *Installer) Uninstall(folder string) error {
	paths := i.owned(i.UserDir)[folder]
	if len(paths) == 0 {
		return nil
	}
	for _, p := range paths {
		if err := os.Remove(p); err != nil {
			return err
		}
		for dir := filepath.Dir(p); dir != i.UserDir && strings.HasPrefix(dir, i.UserDir); dir = filepath.Dir(dir) {
			if os.Remove(dir) != nil { // not empty
				break
			}
		}
	}
	return i.refresh()
}

func (i *Installer) refresh() error {
	if err := i.RefreshCache(i.UserDir); err != nil {
		return fmt.Errorf("%w: %v", ErrCacheRefresh, err)
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
