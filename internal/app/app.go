// Package app is the thin layer bound to the front end: it ties the catalog,
// fetcher and installer together and serves cached font files to the web view.
package app

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"nerder/internal/catalog"
	"nerder/internal/fetch"
	"nerder/internal/installer"
	"nerder/internal/snippet"
)

type FamilyView struct {
	catalog.Family
	Status string `json:"status"` // "none" | "user" | "system"
}

type ListQuery struct {
	Text          string `json:"text"`
	InstalledOnly bool   `json:"installedOnly"`
	Mono          string `json:"mono"` // "" | "mono" | "nonmono"
	License       string `json:"license"`
}

type PreviewInfo struct {
	FontFamily string `json:"fontFamily"` // "nf-<folder>"
	RegularURL string `json:"regularUrl"` // "/fonts/<folder>/<file>"
	BoldURL    string `json:"boldUrl"`    // "" if the family has no Bold
}

type App struct {
	cat *catalog.Catalog
	f   *fetch.Fetcher
	in  *installer.Installer
	ctx context.Context
}

func New(cat *catalog.Catalog, f *fetch.Fetcher, in *installer.Installer) *App {
	return &App{cat: cat, f: f, in: in, ctx: context.Background()}
}

// Startup stores the application context (Wails does not pass one to bound
// methods); downloads are cancelled when the app closes.
func (a *App) Startup(ctx context.Context) { a.ctx = ctx }

func (a *App) List(q ListQuery) ([]FamilyView, error) {
	statuses, err := a.in.Statuses()
	if err != nil {
		return nil, err
	}
	out := []FamilyView{}
	for _, f := range a.cat.Search(q.Text, q.Mono, q.License) {
		s := statuses[f.Folder]
		if s == "" {
			s = installer.NotInstalled
		}
		if q.InstalledOnly && s == installer.NotInstalled {
			continue
		}
		out = append(out, FamilyView{Family: f, Status: string(s)})
	}
	return out, nil
}

func (a *App) Licenses() []string { return a.cat.Licenses() }

func (a *App) Snippet() []snippet.Span { return snippet.Highlight(snippet.Source) }

func (a *App) Preview(folder string) (PreviewInfo, error) {
	if _, ok := a.cat.Find(folder); !ok {
		return PreviewInfo{}, fmt.Errorf("unknown font family %q", folder)
	}
	dir, err := a.f.Ensure(a.ctx, folder)
	if err != nil {
		return PreviewInfo{}, err
	}
	files, err := fetch.FontFiles(dir)
	if err != nil {
		return PreviewInfo{}, err
	}
	regular, bold := fetch.PreviewFiles(files)
	info := PreviewInfo{FontFamily: "nf-" + folder, RegularURL: fontURL(folder, regular)}
	if bold != "" {
		info.BoldURL = fontURL(folder, bold)
	}
	return info, nil
}

func fontURL(folder, file string) string {
	return "/fonts/" + url.PathEscape(folder) + "/" + url.PathEscape(file)
}

// Install downloads the family if needed and installs it. If only the font
// cache refresh failed, the fonts are installed and the error says so.
func (a *App) Install(folder string) (FamilyView, error) {
	if _, ok := a.cat.Find(folder); !ok {
		return FamilyView{}, fmt.Errorf("unknown font family %q", folder)
	}
	dir, err := a.f.Ensure(a.ctx, folder)
	if err != nil {
		return FamilyView{}, err
	}
	err = a.in.Install(folder, dir)
	v, verr := a.view(folder)
	if err != nil {
		return v, err
	}
	return v, verr
}

func (a *App) Uninstall(folder string) (FamilyView, error) {
	if _, ok := a.cat.Find(folder); !ok {
		return FamilyView{}, fmt.Errorf("unknown font family %q", folder)
	}
	err := a.in.Uninstall(folder)
	v, verr := a.view(folder)
	if err != nil {
		return v, err
	}
	return v, verr
}

func (a *App) view(folder string) (FamilyView, error) {
	fam, _ := a.cat.Find(folder)
	statuses, err := a.in.Statuses()
	s := statuses[folder]
	if s == "" {
		s = installer.NotInstalled
	}
	return FamilyView{Family: fam, Status: string(s)}, err
}

// FontHandler serves GET /fonts/<folder>/<file> from the download cache. Only
// files that actually sit in a family's cache directory are served.
func (a *App) FontHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/fonts/"), "/")
		if !strings.HasPrefix(r.URL.Path, "/fonts/") || len(parts) != 2 {
			http.NotFound(w, r)
			return
		}
		folder, file := parts[0], parts[1]
		dir, ok := a.f.Cached(folder)
		if _, known := a.cat.Find(folder); !ok || !known {
			http.NotFound(w, r)
			return
		}
		files, _ := fetch.FontFiles(dir)
		for _, name := range files {
			if name == file {
				ctype := "font/ttf"
				if strings.HasSuffix(strings.ToLower(name), ".otf") {
					ctype = "font/otf"
				}
				w.Header().Set("Content-Type", ctype)
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
				http.ServeFile(w, r, dir+"/"+name)
				return
			}
		}
		http.NotFound(w, r)
	})
}
