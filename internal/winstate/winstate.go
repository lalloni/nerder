// Package winstate persists the main window's geometry between launches.
package winstate

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// State is the window geometry worth restoring. Width and Height describe the
// normal (un-maximised) window, so un-maximising later gives back a sensible size.
type State struct {
	Width     int  `json:"width"`
	Height    int  `json:"height"`
	X         int  `json:"x"`
	Y         int  `json:"y"`
	HasPos    bool `json:"hasPos"`
	Maximised bool `json:"maximised"`
}

const (
	minWidth, minHeight = 400, 300
	maxSize             = 10000
)

// Default is used on first launch or when the saved file is unusable.
var Default = State{Width: 1100, Height: 800}

// Path returns the state file inside the user's config directory.
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "nerder", "window.json"), nil
}

// Load reads the state at path. A missing, corrupt or implausible file yields Default.
func Load(path string) State {
	b, err := os.ReadFile(path)
	if err != nil {
		return Default
	}
	var s State
	if json.Unmarshal(b, &s) != nil || s.Width < minWidth || s.Height < minHeight || s.Width > maxSize || s.Height > maxSize {
		return Default
	}
	return s
}

// Save writes s to path, creating the directory if needed.
func Save(path string, s State) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// Merge builds the state to save from what the window reports now. While the
// window is maximised its reported size is the screen's, so the previously
// saved normal size and position are kept.
func Merge(prev State, w, h, x, y int, maximised bool) State {
	if maximised {
		prev.Maximised = true
		return prev
	}
	return State{Width: w, Height: h, X: x, Y: y, HasPos: true}
}
