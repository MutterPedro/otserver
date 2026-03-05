package otb_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/MutterPedro/otserver/pkg/otb"
)

// TestAcceptanceLoadItems loads the real items.otb file from the data directory
// and verifies that item types are parsed correctly.
func TestAcceptanceLoadItems(t *testing.T) {
	t.Parallel()

	path := filepath.Join("..", "..", "data", "items", "items.otb")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("skipping acceptance test: %v (place items.otb at data/items/items.otb)", err)
	}

	items, err := otb.Parse(data)
	if err != nil {
		t.Fatalf("Parse items.otb: %v", err)
	}

	// A typical TFS items.otb has well over 5000 items.
	if len(items) < 5000 {
		t.Errorf("item count = %d, want >= 5000", len(items))
	}

	// Spot-check: item 2160 is "gold coin" in most TFS distributions.
	gold, ok := items[2160]
	if !ok {
		t.Error("item ServerID=2160 (gold coin) not found")
	} else {
		if gold.ClientID == 0 {
			t.Error("gold coin ClientID = 0, expected non-zero client sprite ID")
		}
	}

	// Verify no item has a zero ServerID (they should all have valid IDs).
	for sid, item := range items {
		if item.ServerID != sid {
			t.Errorf("item map key %d does not match item.ServerID %d", sid, item.ServerID)
		}
	}
}
