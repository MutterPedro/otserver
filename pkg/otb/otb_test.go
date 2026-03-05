package otb_test

import (
	"encoding/binary"
	"testing"

	"github.com/MutterPedro/otserver/pkg/otb"
)

// buildOTBFile constructs a synthetic OTB file from a root header and item entries.
// Each item entry specifies the item group (node type) and a list of attribute chunks.
func buildOTBFile(items []testItem) []byte {
	var buf []byte

	// OTB files start with 4 zero bytes (file identifier)
	buf = append(buf, 0x00, 0x00, 0x00, 0x00)

	// Root node: NODE_START, type=0x00
	buf = append(buf, 0xFE, 0x00)

	// Root header: flags(4) + attr(1) + majorVersion(4) + minorVersion(4) + buildNumber(4) + CSDVersion(128)
	rootFlags := make([]byte, 4) // flags = 0
	buf = append(buf, escapeBytes(rootFlags)...)
	buf = append(buf, escapeBytes([]byte{0x01})...) // attr = OTBI root version attr
	// major version = 3
	ver := make([]byte, 4)
	binary.LittleEndian.PutUint32(ver, 3)
	buf = append(buf, escapeBytes(ver)...)
	// minor version = 0
	binary.LittleEndian.PutUint32(ver, 0)
	buf = append(buf, escapeBytes(ver)...)
	// build number = 0
	binary.LittleEndian.PutUint32(ver, 0)
	buf = append(buf, escapeBytes(ver)...)
	// CSD version string (128 bytes, zero-padded)
	csd := make([]byte, 128)
	copy(csd, "OTBTest")
	buf = append(buf, escapeBytes(csd)...)

	// Child item nodes
	for _, item := range items {
		buf = append(buf, 0xFE)       // NODE_START
		buf = append(buf, item.group) // node type = item group

		// Item flags (4 bytes)
		flags := make([]byte, 4)
		binary.LittleEndian.PutUint32(flags, uint32(item.flags))
		buf = append(buf, escapeBytes(flags)...)

		// Attributes
		for _, attr := range item.attrs {
			buf = append(buf, escapeBytes([]byte{attr.typ})...) // attr type
			attrLen := make([]byte, 2)
			binary.LittleEndian.PutUint16(attrLen, uint16(len(attr.data)))
			buf = append(buf, escapeBytes(attrLen)...) // attr length
			buf = append(buf, escapeBytes(attr.data)...)
		}

		buf = append(buf, 0xFF) // NODE_END
	}

	buf = append(buf, 0xFF) // Root NODE_END
	return buf
}

// escapeBytes applies OTB escape encoding: 0xFD, 0xFE, and 0xFF in data
// are prefixed with the escape byte 0xFD.
func escapeBytes(data []byte) []byte {
	var out []byte
	for _, b := range data {
		if b == 0xFD || b == 0xFE || b == 0xFF {
			out = append(out, 0xFD, b)
		} else {
			out = append(out, b)
		}
	}
	return out
}

type testAttr struct {
	typ  byte
	data []byte
}

type testItem struct {
	group byte
	flags uint32
	attrs []testAttr
}

func makeServerIDAttr(id uint16) testAttr {
	data := make([]byte, 2)
	binary.LittleEndian.PutUint16(data, id)
	return testAttr{typ: 0x10, data: data} // ATTR_SERVERID
}

func makeNameAttr(name string) testAttr {
	return testAttr{typ: 0x12, data: []byte(name)} // ATTR_NAME
}

func makeSpeedAttr(speed uint16) testAttr {
	data := make([]byte, 2)
	binary.LittleEndian.PutUint16(data, speed)
	return testAttr{typ: 0x14, data: data} // ATTR_SPEED
}

func makeWeightAttr(weight uint32) testAttr {
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, weight)
	return testAttr{typ: 0x17, data: data} // ATTR_WEIGHT
}

func makeClientIDAttr(id uint16) testAttr {
	data := make([]byte, 2)
	binary.LittleEndian.PutUint16(data, id)
	return testAttr{typ: 0x11, data: data} // ATTR_CLIENTID
}

func makeArmorAttr(armor uint16) testAttr {
	data := make([]byte, 2)
	binary.LittleEndian.PutUint16(data, armor)
	return testAttr{typ: 0x1A, data: data} // ATTR_ARMOR
}

func makeLightAttr(level, color uint16) testAttr {
	data := make([]byte, 4)
	binary.LittleEndian.PutUint16(data[0:], level)
	binary.LittleEndian.PutUint16(data[2:], color)
	return testAttr{typ: 0x2A, data: data} // ATTR_LIGHT2
}

// TestOTBParseMinimalFile verifies that a synthetic 2-node OTB (root + one item)
// parses correctly and returns the expected item.
func TestOTBParseMinimalFile(t *testing.T) {
	t.Parallel()

	data := buildOTBFile([]testItem{
		{
			group: 0x01, // OTBI_GROUND
			attrs: []testAttr{
				makeServerIDAttr(200),
				makeClientIDAttr(100),
			},
		},
	})

	items, err := otb.Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if len(items) == 0 {
		t.Fatal("Parse returned no items")
	}

	item, ok := items[200]
	if !ok {
		t.Fatalf("item ServerID=200 not found in result map")
	}

	if item.ServerID != 200 {
		t.Errorf("ServerID = %d, want 200", item.ServerID)
	}
	if item.ClientID != 100 {
		t.Errorf("ClientID = %d, want 100", item.ClientID)
	}
	if item.Group != otb.ItemGroupGround {
		t.Errorf("Group = %d, want %d (ItemGroupGround)", item.Group, otb.ItemGroupGround)
	}
}

// TestOTBEscapeByteDecoding verifies that the escape byte 0xFD correctly
// passes through literal control bytes (0xFE, 0xFF) in attribute data.
func TestOTBEscapeByteDecoding(t *testing.T) {
	t.Parallel()

	// Item with ServerID=0xFE (a value that would normally be NODE_START)
	data := buildOTBFile([]testItem{
		{
			group: 0x01,
			attrs: []testAttr{
				makeServerIDAttr(0x00FE), // 0xFE in the low byte
			},
		},
	})

	items, err := otb.Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	item, ok := items[0x00FE]
	if !ok {
		t.Fatal("item ServerID=0x00FE not found — escape byte decoding may be broken")
	}
	if item.ServerID != 0x00FE {
		t.Errorf("ServerID = 0x%04X, want 0x00FE", item.ServerID)
	}
}

// TestOTBItemAttributeServerID verifies that ATTR_SERVERID is parsed correctly.
func TestOTBItemAttributeServerID(t *testing.T) {
	t.Parallel()

	data := buildOTBFile([]testItem{
		{
			group: 0x01,
			attrs: []testAttr{
				makeServerIDAttr(100),
			},
		},
	})

	items, err := otb.Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	item, ok := items[100]
	if !ok {
		t.Fatal("item ServerID=100 not found")
	}
	if item.ServerID != 100 {
		t.Errorf("ServerID = %d, want 100", item.ServerID)
	}
}

// TestOTBItemAttributeName verifies that ATTR_NAME is parsed correctly.
func TestOTBItemAttributeName(t *testing.T) {
	t.Parallel()

	data := buildOTBFile([]testItem{
		{
			group: 0x03, // OTBI_WEAPON
			attrs: []testAttr{
				makeServerIDAttr(500),
				makeNameAttr("Sword"),
			},
		},
	})

	items, err := otb.Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	item, ok := items[500]
	if !ok {
		t.Fatal("item ServerID=500 not found")
	}
	if item.Name != "Sword" {
		t.Errorf("Name = %q, want %q", item.Name, "Sword")
	}
}

// TestOTBItemGroup verifies that the node type correctly maps to ItemGroup.
func TestOTBItemGroup(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		group   byte
		want    otb.ItemGroup
	}{
		{"ground", 0x01, otb.ItemGroupGround},
		{"container", 0x02, otb.ItemGroupContainer},
		{"weapon", 0x03, otb.ItemGroupWeapon},
		{"ammunition", 0x04, otb.ItemGroupAmmunition},
		{"armor", 0x05, otb.ItemGroupArmor},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			data := buildOTBFile([]testItem{
				{
					group: tc.group,
					attrs: []testAttr{
						makeServerIDAttr(uint16(tc.group) * 100),
					},
				},
			})

			items, err := otb.Parse(data)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}

			sid := uint16(tc.group) * 100
			item, ok := items[sid]
			if !ok {
				t.Fatalf("item ServerID=%d not found", sid)
			}
			if item.Group != tc.want {
				t.Errorf("Group = %d, want %d", item.Group, tc.want)
			}
		})
	}
}

// TestOTBTruncatedFile verifies that a truncated file returns an error, not a panic.
func TestOTBTruncatedFile(t *testing.T) {
	t.Parallel()

	// Build a valid file, then truncate it
	data := buildOTBFile([]testItem{
		{
			group: 0x01,
			attrs: []testAttr{makeServerIDAttr(100)},
		},
	})

	// Truncate at various points
	truncations := []int{0, 4, 5, 10, len(data) / 2, len(data) - 1}
	for _, cut := range truncations {
		if cut > len(data) {
			continue
		}
		truncated := data[:cut]
		_, err := otb.Parse(truncated)
		if err == nil {
			t.Errorf("expected error for data truncated at %d bytes, got nil", cut)
		}
	}
}

// TestOTBEmptyFile verifies that zero bytes returns an error.
func TestOTBEmptyFile(t *testing.T) {
	t.Parallel()

	_, err := otb.Parse([]byte{})
	if err == nil {
		t.Error("expected error for empty file, got nil")
	}
}

// TestOTBEmptyNilInput verifies that nil input returns an error.
func TestOTBEmptyNilInput(t *testing.T) {
	t.Parallel()

	_, err := otb.Parse(nil)
	if err == nil {
		t.Error("expected error for nil input, got nil")
	}
}

// TestOTBMultipleItems verifies that multiple items are correctly parsed and
// indexed by their ServerID.
func TestOTBMultipleItems(t *testing.T) {
	t.Parallel()

	data := buildOTBFile([]testItem{
		{group: 0x01, attrs: []testAttr{makeServerIDAttr(100)}},
		{group: 0x03, attrs: []testAttr{makeServerIDAttr(200), makeNameAttr("Axe")}},
		{group: 0x05, attrs: []testAttr{makeServerIDAttr(300), makeNameAttr("Plate Armor")}},
	})

	items, err := otb.Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	if len(items) != 3 {
		t.Errorf("len(items) = %d, want 3", len(items))
	}

	if items[200].Name != "Axe" {
		t.Errorf("item 200 Name = %q, want %q", items[200].Name, "Axe")
	}
	if items[300].Name != "Plate Armor" {
		t.Errorf("item 300 Name = %q, want %q", items[300].Name, "Plate Armor")
	}
}

// TestOTBItemAttributes verifies that various item attributes (speed, weight)
// are parsed correctly.
func TestOTBItemAttributes(t *testing.T) {
	t.Parallel()

	data := buildOTBFile([]testItem{
		{
			group: 0x01,
			attrs: []testAttr{
				makeServerIDAttr(400),
				makeSpeedAttr(220),
				makeWeightAttr(3500),
			},
		},
	})

	items, err := otb.Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	item, ok := items[400]
	if !ok {
		t.Fatal("item ServerID=400 not found")
	}
	if item.Speed != 220 {
		t.Errorf("Speed = %d, want 220", item.Speed)
	}
	if item.Weight != 3500 {
		t.Errorf("Weight = %d, want 3500", item.Weight)
	}
}

// TestOTBInvalidEscapeByte verifies that an escape byte (0xFD) at the very
// end of the file (with no following byte) returns an error.
func TestOTBInvalidEscapeByte(t *testing.T) {
	t.Parallel()

	// Start with a valid file and append a dangling escape byte
	data := buildOTBFile([]testItem{
		{group: 0x01, attrs: []testAttr{makeServerIDAttr(100)}},
	})
	// Remove the final NODE_END and add a dangling escape
	data = data[:len(data)-1]
	data = append(data, 0xFD) // dangling escape with no byte after it

	_, err := otb.Parse(data)
	if err == nil {
		t.Error("expected error for dangling escape byte, got nil")
	}
}

// TestOTBAttributeLengthOutOfBounds verifies that an attribute whose length
// field exceeds the node data returns an error.
func TestOTBAttributeLengthOutOfBounds(t *testing.T) {
	t.Parallel()

	// Manually craft a file with a bad attribute length
	var buf []byte
	buf = append(buf, 0x00, 0x00, 0x00, 0x00) // identifier
	buf = append(buf, 0xFE, 0x00)              // root NODE_START, type=0

	// Root header: minimal (4 flags + 1 attr + 4+4+4+128 version data)
	rootData := make([]byte, 4+1+4+4+4+128)
	rootData[4] = 0x01 // attr = version info
	binary.LittleEndian.PutUint32(rootData[5:], 3) // major version
	copy(rootData[17:], "Test")
	buf = append(buf, escapeBytes(rootData)...)

	// Child node with bad attribute
	buf = append(buf, 0xFE, 0x01) // NODE_START, type=ground
	nodeData := make([]byte, 4)
	// flags (4)
	buf = append(buf, escapeBytes(nodeData)...)
	// Attr type
	buf = append(buf, 0x10) // ATTR_SERVERID
	// Attr length = 9999 (way too long)
	lenBuf := make([]byte, 2)
	binary.LittleEndian.PutUint16(lenBuf, 9999)
	buf = append(buf, escapeBytes(lenBuf)...)
	// Only 2 bytes of actual data
	buf = append(buf, 0x01, 0x00)
	buf = append(buf, 0xFF) // NODE_END
	buf = append(buf, 0xFF) // Root NODE_END

	_, err := otb.Parse(buf)
	if err == nil {
		t.Error("expected error for attribute length exceeding node data, got nil")
	}
}

// TestOTBItemGroupsExtended verifies that all item groups from the plan are
// correctly parsed (rune, teleport, splash, etc.).
func TestOTBItemGroupsExtended(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		group byte
		want  otb.ItemGroup
	}{
		{"rune", 0x06, otb.ItemGroupRune},
		{"teleport", 0x07, otb.ItemGroupTeleport},
		{"magic field", 0x08, otb.ItemGroupMagicField},
		{"writeable", 0x09, otb.ItemGroupWriteable},
		{"key", 0x0A, otb.ItemGroupKey},
		{"splash", 0x0B, otb.ItemGroupSplash},
		{"fluid container", 0x0C, otb.ItemGroupFluidContainer},
		{"door", 0x0D, otb.ItemGroupDoor},
		{"depot", 0x0E, otb.ItemGroupDepot},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			data := buildOTBFile([]testItem{
				{
					group: tc.group,
					attrs: []testAttr{makeServerIDAttr(uint16(tc.group) * 100)},
				},
			})

			items, err := otb.Parse(data)
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}

			sid := uint16(tc.group) * 100
			item, ok := items[sid]
			if !ok {
				t.Fatalf("item ServerID=%d not found", sid)
			}
			if item.Group != tc.want {
				t.Errorf("Group = %d, want %d", item.Group, tc.want)
			}
		})
	}
}

// TestOTBItemCombatAttributes verifies that armor and light
// attributes are parsed correctly.
func TestOTBItemCombatAttributes(t *testing.T) {
	t.Parallel()

	data := buildOTBFile([]testItem{
		{
			group: 0x05, // armor
			attrs: []testAttr{
				makeServerIDAttr(700),
				makeArmorAttr(12),
			},
		},
		{
			group: 0x01, // ground (with light)
			attrs: []testAttr{
				makeServerIDAttr(800),
				makeLightAttr(7, 215),
			},
		},
	})

	items, err := otb.Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	// Armor
	armor := items[700]
	if armor.Armor != 12 {
		t.Errorf("armor Armor = %d, want 12", armor.Armor)
	}

	// Light
	light := items[800]
	if light.LightLevel != 7 {
		t.Errorf("light LightLevel = %d, want 7", light.LightLevel)
	}
	if light.LightColor != 215 {
		t.Errorf("light LightColor = %d, want 215", light.LightColor)
	}
}

// TestOTBClientIDFromAttribute verifies that ATTR_CLIENTID in
// the attribute stream sets the ClientID correctly.
func TestOTBClientIDFromAttribute(t *testing.T) {
	t.Parallel()

	data := buildOTBFile([]testItem{
		{
			group: 0x01,
			attrs: []testAttr{
				makeServerIDAttr(900),
				makeClientIDAttr(999),
			},
		},
	})

	items, err := otb.Parse(data)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	item := items[900]
	if item.ClientID != 999 {
		t.Errorf("ClientID = %d, want 999", item.ClientID)
	}
}
