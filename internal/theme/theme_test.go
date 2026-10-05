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
	th, ok := theme.Get("onedark")
	if !ok {
		t.Fatal("expected onedark to exist")
	}
	if th.Light {
		t.Error("expected onedark to be dark")
	}
	if _, ok := theme.Get("nonexistent-theme-xyz"); ok {
		t.Error("expected nonexistent theme to return false")
	}
}

func TestUserCustomThemeLoading(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	themesDir := filepath.Join(tmpDir, "tsuzuri", "themes")
	if err := os.MkdirAll(themesDir, 0o755); err != nil {
		t.Fatalf("failed to create themes dir: %v", err)
	}

	customTheme := map[string]any{
		"light":      false,
		"fg":         "#cccfd1",
		"darker_bg":  "#040708",
		"bg":         "#0a1014",
		"bg2":        "#162128",
		"one_bg":     "#222e36",
		"one_bg2":    "#283943",
		"one_bg3":    "#354955",
		"grey":       "#596d7b",
		"grey_fg":    "#758a96",
		"grey_fg2":   "#8e9ca4",
		"light_grey": "#a9b0b5",
		"red":        "#bf878c",
		"pink":       "#daafce",
		"line":       "#283943",
		"green":      "#7b9f7e",
		"nord_blue":  "#7f97be",
		"blue":       "#99c1dc",
		"yellow":     "#a1966d",
		"purple":     "#b389a7",
		"dark_purple": "#739bb5",
		"teal":       "#5ba1a3",
		"orange":     "#b88d75",
		"cyan":       "#82c9ca",
		"status_bg":  "#162128",
		"light_bg":   "#222e36",
		"folder":     "#99c1dc",
	}

	data, err := json.Marshal(customTheme)
	if err != nil {
		t.Fatalf("failed to marshal custom theme: %v", err)
	}
	if err := os.WriteFile(filepath.Join(themesDir, "darkknight.json"), data, 0o644); err != nil {
		t.Fatalf("failed to write custom theme: %v", err)
	}

	for _, name := range []string{"darkknight", "DarkKnight", "DARKKNIGHT"} {
		th, ok := theme.Get(name)
		if !ok {
			t.Fatalf("expected custom theme %q to be loaded", name)
		}
		if th.Light {
			t.Errorf("expected %q to be dark theme", name)
		}
		if string(th.Bg) != "#0a1014" {
			t.Errorf("expected Bg #0a1014, got %s", th.Bg)
		}
		if string(th.DarkerBg) != "#040708" {
			t.Errorf("expected DarkerBg #040708, got %s", th.DarkerBg)
		}
		if string(th.Blue) != "#99c1dc" {
			t.Errorf("expected Blue #99c1dc, got %s", th.Blue)
		}
	}

	names := theme.Names()
	found := false
	for _, n := range names {
		if n == "darkknight" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected 'darkknight' in theme.Names()")
	}
}
