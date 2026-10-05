package theme_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/jaisuriya-11/tsuzuri/internal/theme"
)

func TestBundledThemes(t *testing.T) {
	def := theme.DefaultTheme()
	if def.Name != theme.DefaultName {
		t.Errorf("expected default theme %q, got %q", theme.DefaultName, def.Name)
	}
	for _, name := range []string{"onedark", "darkknight"} {
		th, ok := theme.Get(name)
		if !ok {
			t.Fatalf("expected bundled theme %q to exist", name)
		}
		if th.Light {
			t.Errorf("expected %q to be dark theme", name)
		}
	}
	if _, ok := theme.Get("nonexistent-theme-xyz"); ok {
		t.Error("expected nonexistent theme to return false")
	}
}

func TestUserCustomThemeLoading(t *testing.T) {
	tmpDir := t.TempDir()
	restore := theme.SetUserThemesDirForTest(func() string {
		return tmpDir
	})
	defer restore()

	customTheme := map[string]any{
		"light":       false,
		"fg":          "#ffffff",
		"darker_bg":   "#050505",
		"bg":          "#111111",
		"bg2":         "#1a1a1a",
		"one_bg":      "#222222",
		"one_bg2":     "#2a2a2a",
		"one_bg3":     "#333333",
		"grey":        "#444444",
		"grey_fg":     "#555555",
		"grey_fg2":    "#666666",
		"light_grey":  "#777777",
		"red":         "#ff0000",
		"pink":        "#ff00ff",
		"line":        "#222222",
		"green":       "#00ff00",
		"nord_blue":   "#0000ff",
		"blue":        "#0088ff",
		"yellow":      "#ffff00",
		"purple":      "#8800ff",
		"dark_purple": "#440088",
		"teal":        "#00ffff",
		"orange":      "#ff8800",
		"cyan":        "#00aaff",
		"status_bg":   "#1a1a1a",
		"light_bg":    "#222222",
		"folder":      "#0088ff",
	}

	data, err := json.Marshal(customTheme)
	if err != nil {
		t.Fatalf("failed to marshal custom theme: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "custom-test.json"), data, 0o644); err != nil {
		t.Fatalf("failed to write custom theme: %v", err)
	}

	for _, name := range []string{"custom-test", "Custom-Test", "CUSTOM-TEST"} {
		th, ok := theme.Get(name)
		if !ok {
			t.Fatalf("expected custom theme %q to be loaded from disk", name)
		}
		if th.Light {
			t.Errorf("expected %q to be dark theme", name)
		}
		if string(th.Bg) != "#111111" {
			t.Errorf("expected Bg #111111, got %s", th.Bg)
		}
		if string(th.DarkerBg) != "#050505" {
			t.Errorf("expected DarkerBg #050505, got %s", th.DarkerBg)
		}
		if string(th.Blue) != "#0088ff" {
			t.Errorf("expected Blue #0088ff, got %s", th.Blue)
		}
	}

	names := theme.Names()
	found := false
	for _, n := range names {
		if n == "custom-test" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'custom-test' in theme.Names()")
	}
}
