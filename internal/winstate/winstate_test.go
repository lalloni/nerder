package winstate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFallsBackToDefault(t *testing.T) {
	dir := t.TempDir()
	if got := Load(filepath.Join(dir, "missing.json")); got != Default {
		t.Errorf("missing file = %+v, want default", got)
	}
	bad := filepath.Join(dir, "bad.json")
	os.WriteFile(bad, []byte("{nope"), 0o644)
	if got := Load(bad); got != Default {
		t.Errorf("corrupt file = %+v, want default", got)
	}
	tiny := filepath.Join(dir, "tiny.json")
	os.WriteFile(tiny, []byte(`{"width":10,"height":10}`), 0o644)
	if got := Load(tiny); got != Default {
		t.Errorf("implausible size = %+v, want default", got)
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "window.json")
	want := State{Width: 1280, Height: 720, X: 40, Y: 60, HasPos: true, Maximised: true}
	if err := Save(path, want); err != nil {
		t.Fatal(err)
	}
	if got := Load(path); got != want {
		t.Errorf("Load = %+v, want %+v", got, want)
	}
}

func TestMergeKeepsNormalGeometryWhileMaximised(t *testing.T) {
	prev := State{Width: 900, Height: 700, X: 10, Y: 20, HasPos: true}
	got := Merge(prev, 1920, 1080, 0, 0, true)
	if got.Width != 900 || got.Height != 700 || got.X != 10 || !got.Maximised {
		t.Errorf("maximised merge = %+v", got)
	}
	got = Merge(got, 1000, 650, 5, 6, false)
	if got.Width != 1000 || got.Height != 650 || got.X != 5 || got.Maximised {
		t.Errorf("normal merge = %+v", got)
	}
}
