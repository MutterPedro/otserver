package otbm_test

import (
	"encoding/binary"
	"testing"

	"github.com/MutterPedro/otserver/pkg/otbm"
)

// escapeBytes applies OTB/OTBM escape encoding.
func escapeBytes(data []byte) []byte {
	var out []byte
	for _, b := range data {
		if b == otbm.EscapeByte || b == otbm.NodeStartByte || b == otbm.NodeEndByte {
			out = append(out, otbm.EscapeByte, b)
		} else {
			out = append(out, b)
		}
	}
	return out
}

func appendU16(buf []byte, v uint16) []byte {
	b := make([]byte, 2)
	binary.LittleEndian.PutUint16(b, v)
	return append(buf, b...)
}

func appendU32(buf []byte, v uint32) []byte {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, v)
	return append(buf, b...)
}

// buildOTBMFile constructs a minimal OTBM file with the given tile areas, towns, and waypoints.
type tileEntry struct {
	offsetX uint8
	offsetY uint8
	houseID uint32 // 0 = normal tile, >0 = house tile
	items   []itemEntry
}

type itemEntry struct {
	id uint16
}

type tileAreaEntry struct {
	baseX uint16
	baseY uint16
	baseZ uint8
	tiles []tileEntry
}

type townEntry struct {
	id      uint32
	name    string
	templeX uint16
	templeY uint16
	templeZ uint8
}

type waypointEntry struct {
	name string
	x    uint16
	y    uint16
	z    uint8
}

func buildOTBMFile(areas []tileAreaEntry, towns []townEntry, waypoints []waypointEntry, spawnFile, houseFile string) []byte {
	var buf []byte

	// OTBM files start with 4 zero bytes (file identifier)
	buf = append(buf, 0x00, 0x00, 0x00, 0x00)

	// Root node: OTBM_MAP_HEADER
	buf = append(buf, otbm.NodeStartByte, otbm.OTBMMapHeader)

	// Header props: version(4) + width(2) + height(2) + majorItems(4) + minorItems(4)
	var headerProps []byte
	headerProps = appendU32(headerProps, 2) // version = 2
	headerProps = appendU16(headerProps, 512) // width
	headerProps = appendU16(headerProps, 512) // height
	headerProps = appendU32(headerProps, 3)   // major items version
	headerProps = appendU32(headerProps, 0)   // minor items version
	buf = append(buf, escapeBytes(headerProps)...)

	// OTBM_MAP_DATA child
	buf = append(buf, otbm.NodeStartByte, otbm.OTBMMapData)

	// Map data attributes (description, spawn file, house file)
	if spawnFile != "" {
		var attrBuf []byte
		attrBuf = append(attrBuf, otbm.AttrSpawnFile)
		attrBuf = appendU16(attrBuf, uint16(len(spawnFile)))
		attrBuf = append(attrBuf, spawnFile...)
		buf = append(buf, escapeBytes(attrBuf)...)
	}
	if houseFile != "" {
		var attrBuf []byte
		attrBuf = append(attrBuf, otbm.AttrHouseFile)
		attrBuf = appendU16(attrBuf, uint16(len(houseFile)))
		attrBuf = append(attrBuf, houseFile...)
		buf = append(buf, escapeBytes(attrBuf)...)
	}

	// Tile areas
	for _, area := range areas {
		buf = append(buf, otbm.NodeStartByte, otbm.OTBMTileArea)

		// TileArea props: baseX(2) + baseY(2) + baseZ(1)
		var areaProps []byte
		areaProps = appendU16(areaProps, area.baseX)
		areaProps = appendU16(areaProps, area.baseY)
		areaProps = append(areaProps, area.baseZ)
		buf = append(buf, escapeBytes(areaProps)...)

		// Tiles
		for _, tile := range area.tiles {
			if tile.houseID > 0 {
				buf = append(buf, otbm.NodeStartByte, otbm.OTBMHouseTile)
				var tileProps []byte
				tileProps = append(tileProps, tile.offsetX, tile.offsetY)
				tileProps = appendU32(tileProps, tile.houseID)
				buf = append(buf, escapeBytes(tileProps)...)
			} else {
				buf = append(buf, otbm.NodeStartByte, otbm.OTBMTile)
				var tileProps []byte
				tileProps = append(tileProps, tile.offsetX, tile.offsetY)
				buf = append(buf, escapeBytes(tileProps)...)
			}

			// Items on tile
			for _, item := range tile.items {
				buf = append(buf, otbm.NodeStartByte, otbm.OTBMItem)
				var itemProps []byte
				itemProps = appendU16(itemProps, item.id)
				buf = append(buf, escapeBytes(itemProps)...)
				buf = append(buf, otbm.NodeEndByte) // item NODE_END
			}

			buf = append(buf, otbm.NodeEndByte) // tile NODE_END
		}

		buf = append(buf, otbm.NodeEndByte) // tile area NODE_END
	}

	// Towns
	if len(towns) > 0 {
		buf = append(buf, otbm.NodeStartByte, otbm.OTBMTowns)
		for _, town := range towns {
			buf = append(buf, otbm.NodeStartByte, otbm.OTBMTown)
			var townProps []byte
			townProps = appendU32(townProps, town.id)
			// Town name: uint16 len + string
			townProps = appendU16(townProps, uint16(len(town.name)))
			townProps = append(townProps, town.name...)
			// Temple pos: x(2) + y(2) + z(1)
			townProps = appendU16(townProps, town.templeX)
			townProps = appendU16(townProps, town.templeY)
			townProps = append(townProps, town.templeZ)
			buf = append(buf, escapeBytes(townProps)...)
			buf = append(buf, otbm.NodeEndByte) // town NODE_END
		}
		buf = append(buf, otbm.NodeEndByte) // towns NODE_END
	}

	// Waypoints
	if len(waypoints) > 0 {
		buf = append(buf, otbm.NodeStartByte, otbm.OTBMWaypoints)
		for _, wp := range waypoints {
			buf = append(buf, otbm.NodeStartByte, otbm.OTBMWaypoint)
			var wpProps []byte
			wpProps = appendU16(wpProps, uint16(len(wp.name)))
			wpProps = append(wpProps, wp.name...)
			wpProps = appendU16(wpProps, wp.x)
			wpProps = appendU16(wpProps, wp.y)
			wpProps = append(wpProps, wp.z)
			buf = append(buf, escapeBytes(wpProps)...)
			buf = append(buf, otbm.NodeEndByte) // waypoint NODE_END
		}
		buf = append(buf, otbm.NodeEndByte) // waypoints NODE_END
	}

	buf = append(buf, otbm.NodeEndByte) // map data NODE_END
	buf = append(buf, otbm.NodeEndByte) // root NODE_END
	return buf
}

// TestOTBMParseTileArea verifies that a tile area with two tiles at known
// positions is parsed correctly.
func TestOTBMParseTileArea(t *testing.T) {
	t.Parallel()

	data := buildOTBMFile(
		[]tileAreaEntry{
			{
				baseX: 100, baseY: 200, baseZ: 7,
				tiles: []tileEntry{
					{offsetX: 0, offsetY: 0},
					{offsetX: 3, offsetY: 5},
				},
			},
		},
		nil, nil, "", "",
	)

	m, err := otbm.LoadMap(data)
	if err != nil {
		t.Fatalf("LoadMap: %v", err)
	}

	if len(m.Tiles) != 2 {
		t.Fatalf("len(Tiles) = %d, want 2", len(m.Tiles))
	}

	// Check first tile position: base(100,200,7) + offset(0,0)
	tile1, ok := m.Tiles[otbm.Position{X: 100, Y: 200, Z: 7}]
	if !ok {
		t.Error("tile at (100,200,7) not found")
	} else if tile1.Position.X != 100 || tile1.Position.Y != 200 || tile1.Position.Z != 7 {
		t.Errorf("tile1 position = %+v, want (100,200,7)", tile1.Position)
	}

	// Check second tile position: base(100,200,7) + offset(3,5)
	tile2, ok := m.Tiles[otbm.Position{X: 103, Y: 205, Z: 7}]
	if !ok {
		t.Error("tile at (103,205,7) not found")
	} else if tile2.Position.X != 103 || tile2.Position.Y != 205 || tile2.Position.Z != 7 {
		t.Errorf("tile2 position = %+v, want (103,205,7)", tile2.Position)
	}
}

// TestOTBMParseTileWithItem verifies that a tile containing an item is parsed correctly.
func TestOTBMParseTileWithItem(t *testing.T) {
	t.Parallel()

	data := buildOTBMFile(
		[]tileAreaEntry{
			{
				baseX: 50, baseY: 50, baseZ: 7,
				tiles: []tileEntry{
					{
						offsetX: 1, offsetY: 2,
						items: []itemEntry{{id: 2160}}, // gold coin
					},
				},
			},
		},
		nil, nil, "", "",
	)

	m, err := otbm.LoadMap(data)
	if err != nil {
		t.Fatalf("LoadMap: %v", err)
	}

	tile, ok := m.Tiles[otbm.Position{X: 51, Y: 52, Z: 7}]
	if !ok {
		t.Fatal("tile at (51,52,7) not found")
	}

	if len(tile.Items) != 1 {
		t.Fatalf("len(tile.Items) = %d, want 1", len(tile.Items))
	}
	if tile.Items[0].ID != 2160 {
		t.Errorf("item ID = %d, want 2160", tile.Items[0].ID)
	}
}

// TestOTBMParseTown verifies that town nodes are parsed correctly.
func TestOTBMParseTown(t *testing.T) {
	t.Parallel()

	data := buildOTBMFile(
		nil,
		[]townEntry{
			{id: 1, name: "Thais", templeX: 500, templeY: 600, templeZ: 7},
			{id: 2, name: "Carlin", templeX: 700, templeY: 800, templeZ: 7},
		},
		nil, "", "",
	)

	m, err := otbm.LoadMap(data)
	if err != nil {
		t.Fatalf("LoadMap: %v", err)
	}

	if len(m.Towns) != 2 {
		t.Fatalf("len(Towns) = %d, want 2", len(m.Towns))
	}

	if m.Towns[0].ID != 1 || m.Towns[0].Name != "Thais" {
		t.Errorf("Towns[0] = %+v, want {ID:1, Name:Thais, ...}", m.Towns[0])
	}
	if m.Towns[0].Temple.X != 500 || m.Towns[0].Temple.Y != 600 || m.Towns[0].Temple.Z != 7 {
		t.Errorf("Towns[0].Temple = %+v, want (500,600,7)", m.Towns[0].Temple)
	}
	if m.Towns[1].ID != 2 || m.Towns[1].Name != "Carlin" {
		t.Errorf("Towns[1] = %+v, want {ID:2, Name:Carlin, ...}", m.Towns[1])
	}
}

// TestOTBMHouseTile verifies that OTBM_HOUSETILE nodes have their HouseID set.
func TestOTBMHouseTile(t *testing.T) {
	t.Parallel()

	data := buildOTBMFile(
		[]tileAreaEntry{
			{
				baseX: 300, baseY: 400, baseZ: 7,
				tiles: []tileEntry{
					{offsetX: 0, offsetY: 0, houseID: 42},
				},
			},
		},
		nil, nil, "", "",
	)

	m, err := otbm.LoadMap(data)
	if err != nil {
		t.Fatalf("LoadMap: %v", err)
	}

	tile, ok := m.Tiles[otbm.Position{X: 300, Y: 400, Z: 7}]
	if !ok {
		t.Fatal("tile at (300,400,7) not found")
	}
	if tile.HouseID != 42 {
		t.Errorf("HouseID = %d, want 42", tile.HouseID)
	}
}

// TestOTBMTruncated verifies that truncated input returns an error, not a panic.
func TestOTBMTruncated(t *testing.T) {
	t.Parallel()

	data := buildOTBMFile(
		[]tileAreaEntry{
			{
				baseX: 100, baseY: 200, baseZ: 7,
				tiles: []tileEntry{
					{offsetX: 0, offsetY: 0},
				},
			},
		},
		nil, nil, "", "",
	)

	truncations := []int{0, 4, 5, 10, len(data) / 2, len(data) - 1}
	for _, cut := range truncations {
		if cut > len(data) {
			continue
		}
		truncated := data[:cut]
		_, err := otbm.LoadMap(truncated)
		if err == nil {
			t.Errorf("expected error for data truncated at %d bytes, got nil", cut)
		}
	}
}

// TestOTBMEmptyFile verifies that an empty file returns an error.
func TestOTBMEmptyFile(t *testing.T) {
	t.Parallel()

	_, err := otbm.LoadMap([]byte{})
	if err == nil {
		t.Error("expected error for empty file, got nil")
	}

	_, err = otbm.LoadMap(nil)
	if err == nil {
		t.Error("expected error for nil input, got nil")
	}
}

// TestOTBMSpawnAndHouseFiles verifies that map attributes (spawn file, house file) are parsed.
func TestOTBMSpawnAndHouseFiles(t *testing.T) {
	t.Parallel()

	data := buildOTBMFile(nil, nil, nil, "spawns.xml", "houses.xml")

	m, err := otbm.LoadMap(data)
	if err != nil {
		t.Fatalf("LoadMap: %v", err)
	}

	if m.SpawnFile != "spawns.xml" {
		t.Errorf("SpawnFile = %q, want %q", m.SpawnFile, "spawns.xml")
	}
	if m.HouseFile != "houses.xml" {
		t.Errorf("HouseFile = %q, want %q", m.HouseFile, "houses.xml")
	}
}

// TestOTBMParseWaypoints verifies that waypoint nodes are parsed correctly.
func TestOTBMParseWaypoints(t *testing.T) {
	t.Parallel()

	data := buildOTBMFile(
		nil,
		nil,
		[]waypointEntry{
			{name: "depot", x: 100, y: 200, z: 7},
		},
		"", "",
	)

	m, err := otbm.LoadMap(data)
	if err != nil {
		t.Fatalf("LoadMap: %v", err)
	}

	if len(m.Waypoints) != 1 {
		t.Fatalf("len(Waypoints) = %d, want 1", len(m.Waypoints))
	}
	wp := m.Waypoints[0]
	if wp.Name != "depot" {
		t.Errorf("Waypoint.Name = %q, want %q", wp.Name, "depot")
	}
	if wp.Position.X != 100 || wp.Position.Y != 200 || wp.Position.Z != 7 {
		t.Errorf("Waypoint.Position = %+v, want (100,200,7)", wp.Position)
	}
}

// TestOTBMMaxDepth verifies that deeply nested items (containers within containers)
// don't cause stack overflow. The parser should either handle them or return an error.
func TestOTBMMaxDepth(t *testing.T) {
	t.Parallel()

	// Build a file with a deeply nested container chain:
	// tile -> item -> item -> item -> ... (50 levels deep)
	var buf []byte
	buf = append(buf, 0x00, 0x00, 0x00, 0x00) // identifier

	// Root node: OTBM_MAP_HEADER
	buf = append(buf, otbm.NodeStartByte, otbm.OTBMMapHeader)
	var headerProps []byte
	headerProps = appendU32(headerProps, 2)   // version
	headerProps = appendU16(headerProps, 256) // width
	headerProps = appendU16(headerProps, 256) // height
	headerProps = appendU32(headerProps, 3)   // major items
	headerProps = appendU32(headerProps, 0)   // minor items
	buf = append(buf, escapeBytes(headerProps)...)

	// OTBM_MAP_DATA
	buf = append(buf, otbm.NodeStartByte, otbm.OTBMMapData)
	// OTBM_TILE_AREA
	buf = append(buf, otbm.NodeStartByte, otbm.OTBMTileArea)
	var areaProps []byte
	areaProps = appendU16(areaProps, 100)
	areaProps = appendU16(areaProps, 100)
	areaProps = append(areaProps, 7)
	buf = append(buf, escapeBytes(areaProps)...)

	// OTBM_TILE
	buf = append(buf, otbm.NodeStartByte, otbm.OTBMTile)
	buf = append(buf, escapeBytes([]byte{0, 0})...) // offset x=0, y=0

	// Nest 50 items deep
	const depth = 50
	for i := 0; i < depth; i++ {
		buf = append(buf, otbm.NodeStartByte, otbm.OTBMItem) // OTBM_ITEM
		var itemProps []byte
		itemProps = appendU16(itemProps, uint16(2000+i))
		buf = append(buf, escapeBytes(itemProps)...)
	}
	// Close all 50 items
	for i := 0; i < depth; i++ {
		buf = append(buf, otbm.NodeEndByte)
	}

	buf = append(buf, otbm.NodeEndByte) // tile
	buf = append(buf, otbm.NodeEndByte) // tile area
	buf = append(buf, otbm.NodeEndByte) // map data
	buf = append(buf, otbm.NodeEndByte) // root

	// This should not panic with a stack overflow
	m, err := otbm.LoadMap(buf)
	if err != nil {
		// An error is acceptable (if we add a max depth limit)
		return
	}

	// If no error, verify the tile exists
	tile, ok := m.Tiles[otbm.Position{X: 100, Y: 100, Z: 7}]
	if !ok {
		t.Fatal("tile at (100,100,7) not found")
	}

	// Verify the nested item chain
	if len(tile.Items) != 1 {
		t.Fatalf("len(tile.Items) = %d, want 1 (top-level item)", len(tile.Items))
	}

	// Walk down the sub-item chain
	current := tile.Items[0]
	for i := 0; i < depth-1; i++ {
		if len(current.SubItems) != 1 {
			t.Fatalf("depth %d: len(SubItems) = %d, want 1", i, len(current.SubItems))
		}
		current = current.SubItems[0]
	}
}

// TestOTBMMultipleTileAreas verifies that multiple tile areas are merged correctly.
func TestOTBMMultipleTileAreas(t *testing.T) {
	t.Parallel()

	data := buildOTBMFile(
		[]tileAreaEntry{
			{
				baseX: 100, baseY: 100, baseZ: 7,
				tiles: []tileEntry{
					{offsetX: 0, offsetY: 0},
				},
			},
			{
				baseX: 200, baseY: 200, baseZ: 6,
				tiles: []tileEntry{
					{offsetX: 1, offsetY: 1},
				},
			},
		},
		nil, nil, "", "",
	)

	m, err := otbm.LoadMap(data)
	if err != nil {
		t.Fatalf("LoadMap: %v", err)
	}

	if len(m.Tiles) != 2 {
		t.Errorf("len(Tiles) = %d, want 2", len(m.Tiles))
	}

	if _, ok := m.Tiles[otbm.Position{X: 100, Y: 100, Z: 7}]; !ok {
		t.Error("tile from first area not found at (100,100,7)")
	}
	if _, ok := m.Tiles[otbm.Position{X: 201, Y: 201, Z: 6}]; !ok {
		t.Error("tile from second area not found at (201,201,6)")
	}
}

// TestOTBMMapDimensions verifies that width and height from the header are parsed.
func TestOTBMMapDimensions(t *testing.T) {
	t.Parallel()

	data := buildOTBMFile(nil, nil, nil, "", "")

	m, err := otbm.LoadMap(data)
	if err != nil {
		t.Fatalf("LoadMap: %v", err)
	}

	if m.Width != 512 {
		t.Errorf("Width = %d, want 512", m.Width)
	}
	if m.Height != 512 {
		t.Errorf("Height = %d, want 512", m.Height)
	}
}
