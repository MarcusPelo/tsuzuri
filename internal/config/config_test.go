package config_test

import (
	"path/filepath"
	"testing"

	"github.com/jaisuriya-11/tsuzuri/internal/config"
)

func TestLoadSaveRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "config.json")
	c, err := config.Load(path)
	if err != nil || c.Theme != "" {
		t.Fatalf("missing file should load empty config, got %+v %v", c, err)
	}
	if err := config.Save(path, config.Config{Theme: "gruvbox"}); err != nil {
		t.Fatal(err)
	}
	c, err = config.Load(path)
	if err != nil || c.Theme != "gruvbox" {
		t.Fatalf("round trip got %+v %v", c, err)
	}
}
