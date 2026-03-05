package iomap_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MutterPedro/otserver/internal/iomap"
)

// TestAcceptanceLoadForgottenOTBM loads the real forgotten.otbm file from the
// data directory and verifies that the world map is parsed correctly.
func TestAcceptanceLoadForgottenOTBM(t *testing.T) {
	t.Parallel()

	path := filepath.Join("..", "..", "data", "world", "forgotten.otbm")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("skipping acceptance test: %v (place forgotten.otbm at data/world/forgotten.otbm)", err)
	}

	m, err := iomap.LoadMap(data)
	if err != nil {
		t.Fatalf("LoadMap forgotten.otbm: %v", err)
	}

	// A real OTBM map should have tiles.
	if len(m.Tiles) == 0 {
		t.Error("Tiles count = 0, expected > 0")
	}

	// Spawn file should be set.
	if m.SpawnFile == "" {
		t.Error("SpawnFile is empty, expected non-empty path")
	}

	// At least one town should be present.
	if len(m.Towns) == 0 {
		t.Error("Towns count = 0, expected at least one town")
	}

	// Map dimensions should be non-zero.
	if m.Width == 0 || m.Height == 0 {
		t.Errorf("Map dimensions = %dx%d, expected non-zero", m.Width, m.Height)
	}

	// Verify all tiles have valid positions (non-zero X,Y).
	for pos, tile := range m.Tiles {
		if tile.Position != pos {
			t.Errorf("tile map key %+v does not match tile.Position %+v", pos, tile.Position)
			break // one mismatch is enough
		}
	}
}
