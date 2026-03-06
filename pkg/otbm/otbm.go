// Package otbm implements the OTBM map file format parser.
package otbm

import (
	"errors"
	"fmt"

	"github.com/MutterPedro/otserver/pkg/propstream"
)

// Control bytes for OTBM node tree encoding.
const (
	NodeStartByte byte = 0xFE
	NodeEndByte   byte = 0xFF
	EscapeByte    byte = 0xFD
)

// OTBM node types.
// Values match the C++ enum in iomap.h.
const (
	OTBMMapHeader byte = 0x00
	OTBMMapData   byte = 0x02
	OTBMTileArea  byte = 0x04
	OTBMTile      byte = 0x05
	OTBMItem      byte = 0x06
	OTBMTowns     byte = 0x0C
	OTBMTown      byte = 0x0D
	OTBMHouseTile byte = 0x0E
	OTBMWaypoints byte = 0x0F
	OTBMWaypoint  byte = 0x10
)

// OTBM attribute types.
// Values match the C++ enums in iomap.h (OTBM_ATTR_*) and item.h (ATTR_*).
const (
	AttrDescription   byte = 1
	AttrExtFile       byte = 2
	AttrTileFlags     byte = 3
	AttrActionID      byte = 4
	AttrUniqueID      byte = 5
	AttrText          byte = 6
	AttrDesc          byte = 7
	AttrTeleDest      byte = 8
	AttrItem          byte = 9
	AttrDepotID       byte = 10
	AttrSpawnFile     byte = 11
	AttrRuneCharges   byte = 12
	AttrHouseFile     byte = 13
	AttrHouseDoorID   byte = 14
	AttrCount         byte = 15
	AttrDuration      byte = 16
	AttrDecayingState byte = 17
	AttrWrittenDate   byte = 18
	AttrWrittenBy     byte = 19
	AttrSleeperGUID   byte = 20
	AttrSleepStart    byte = 21
	AttrCharges       byte = 22
)

// Position represents a 3D coordinate in the game world.
type Position struct {
	X uint16
	Y uint16
	Z uint8
}

// RawItem holds the raw data for an item as loaded from the map file.
type RawItem struct {
	ID       uint16
	Count    uint8
	ActionID uint16
	UniqueID uint16
	Text     string
	SubItems []RawItem
}

// Tile represents a single map tile with its position, flags, items, and optional house ID.
type Tile struct {
	Position Position
	Flags    uint32
	Items    []RawItem
	HouseID  uint32
}

// Town represents a town entry in the map.
type Town struct {
	ID     uint32
	Name   string
	Temple Position
}

// Waypoint represents a named waypoint on the map.
type Waypoint struct {
	Name     string
	Position Position
}

// Map holds the fully parsed OTBM map data.
type Map struct {
	Width     uint16
	Height    uint16
	Tiles     map[Position]*Tile
	Towns     []Town
	Waypoints []Waypoint
	SpawnFile string
	HouseFile string
}

// node represents a parsed node in the OTBM tree.
type node struct {
	nodeType byte
	props    []byte
	children []*node
}

// parseNodes parses the raw byte stream (after the 4-byte file identifier) into
// a tree of nodes, handling escape bytes and NODE_START/NODE_END markers.
func parseNodes(data []byte) (*node, error) {
	if len(data) == 0 {
		return nil, errors.New("otbm: empty node data")
	}

	pos := 0
	if data[pos] != NodeStartByte {
		return nil, errors.New("otbm: expected NODE_START at beginning of node tree")
	}
	pos++

	root, newPos, err := readNode(data, pos)
	if err != nil {
		return nil, err
	}
	pos = newPos

	if pos != len(data) {
		return nil, fmt.Errorf("otbm: trailing data after root node (%d extra bytes)", len(data)-pos)
	}

	return root, nil
}

// readNode reads a single node starting at data[pos]. The caller has already
// consumed the NODE_START byte. pos points to the node type byte.
func readNode(data []byte, pos int) (*node, int, error) {
	if pos >= len(data) {
		return nil, 0, errors.New("otbm: unexpected end of data reading node type")
	}

	n := &node{nodeType: data[pos]}
	pos++

	for pos < len(data) {
		b := data[pos]
		switch b {
		case NodeStartByte:
			pos++
			child, newPos, err := readNode(data, pos)
			if err != nil {
				return nil, 0, err
			}
			n.children = append(n.children, child)
			pos = newPos
		case NodeEndByte:
			pos++
			return n, pos, nil
		case EscapeByte:
			pos++
			if pos >= len(data) {
				return nil, 0, errors.New("otbm: dangling escape byte at end of data")
			}
			n.props = append(n.props, data[pos])
			pos++
		default:
			n.props = append(n.props, b)
			pos++
		}
	}

	return nil, 0, errors.New("otbm: unexpected end of data, missing NODE_END")
}

// LoadMap parses an OTBM binary file and returns a Map.
func LoadMap(data []byte) (*Map, error) {
	if len(data) < 4 {
		return nil, errors.New("otbm: data too short for file identifier")
	}

	root, err := parseNodes(data[4:])
	if err != nil {
		return nil, err
	}

	if root.nodeType != OTBMMapHeader {
		return nil, fmt.Errorf("otbm: expected OTBM_MAP_HEADER (0x%02X), got 0x%02X", OTBMMapHeader, root.nodeType)
	}

	// Parse root header: version(4) + width(2) + height(2) + majorItems(4) + minorItems(4) = 16 bytes
	ps := propstream.NewPropStream(root.props)
	if err := ps.Skip(4); err != nil { // skip version
		return nil, fmt.Errorf("otbm: map header too short: %w", err)
	}

	m := &Map{
		Tiles: make(map[Position]*Tile),
	}

	m.Width, err = ps.ReadUint16()
	if err != nil {
		return nil, fmt.Errorf("otbm: reading map width: %w", err)
	}
	m.Height, err = ps.ReadUint16()
	if err != nil {
		return nil, fmt.Errorf("otbm: reading map height: %w", err)
	}

	// Find the OTBM_MAP_DATA child node.
	var mapDataNode *node
	for _, child := range root.children {
		if child.nodeType == OTBMMapData {
			mapDataNode = child
			break
		}
	}

	if mapDataNode == nil {
		return m, nil
	}

	// Parse map data attributes from props (spawn file, house file, description).
	if err := parseMapDataAttrs(mapDataNode.props, m); err != nil {
		return nil, err
	}

	// Process children of OTBM_MAP_DATA.
	for _, child := range mapDataNode.children {
		switch child.nodeType {
		case OTBMTileArea:
			if err := parseTileArea(child, m); err != nil {
				return nil, err
			}
		case OTBMTowns:
			if err := parseTowns(child, m); err != nil {
				return nil, err
			}
		case OTBMWaypoints:
			if err := parseWaypoints(child, m); err != nil {
				return nil, err
			}
		}
	}

	return m, nil
}

// parseMapDataAttrs parses sequential attributes from the OTBM_MAP_DATA node props.
func parseMapDataAttrs(props []byte, m *Map) error {
	ps := propstream.NewPropStream(props)
	for ps.Remaining() > 0 {
		attrType, err := ps.ReadUint8()
		if err != nil {
			return fmt.Errorf("otbm: reading map data attr type: %w", err)
		}

		switch attrType {
		case AttrDescription, AttrExtFile, AttrSpawnFile, AttrHouseFile:
			str, err := ps.ReadString()
			if err != nil {
				return fmt.Errorf("otbm: reading map data attr string: %w", err)
			}

			switch attrType {
			case AttrSpawnFile:
				m.SpawnFile = str
			case AttrHouseFile:
				m.HouseFile = str
			}
		default:
			// Unknown attribute; we cannot determine its length so we stop parsing.
			return nil
		}
	}
	return nil
}

// parseTileArea parses an OTBM_TILE_AREA node and its tile children.
func parseTileArea(n *node, m *Map) error {
	ps := propstream.NewPropStream(n.props)

	baseX, err := ps.ReadUint16()
	if err != nil {
		return fmt.Errorf("otbm: reading tile area base X: %w", err)
	}
	baseY, err := ps.ReadUint16()
	if err != nil {
		return fmt.Errorf("otbm: reading tile area base Y: %w", err)
	}
	baseZ, err := ps.ReadUint8()
	if err != nil {
		return fmt.Errorf("otbm: reading tile area base Z: %w", err)
	}

	for _, child := range n.children {
		switch child.nodeType {
		case OTBMTile:
			tile, err := parseTile(child, baseX, baseY, baseZ, 0)
			if err != nil {
				return err
			}
			m.Tiles[tile.Position] = tile
		case OTBMHouseTile:
			tile, err := parseHouseTile(child, baseX, baseY, baseZ)
			if err != nil {
				return err
			}
			m.Tiles[tile.Position] = tile
		}
	}

	return nil
}

// parseTile parses an OTBM_TILE node.
func parseTile(n *node, baseX, baseY uint16, baseZ uint8, houseID uint32) (*Tile, error) {
	ps := propstream.NewPropStream(n.props)

	offsetX, err := ps.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("otbm: reading tile offset X: %w", err)
	}
	offsetY, err := ps.ReadUint8()
	if err != nil {
		return nil, fmt.Errorf("otbm: reading tile offset Y: %w", err)
	}

	tile := &Tile{
		Position: Position{
			X: baseX + uint16(offsetX),
			Y: baseY + uint16(offsetY),
			Z: baseZ,
		},
		HouseID: houseID,
	}

	// Parse tile attributes from remaining props.
	for ps.Remaining() > 0 {
		attrType, err := ps.ReadUint8()
		if err != nil {
			return nil, fmt.Errorf("otbm: reading tile attr type: %w", err)
		}

		switch attrType {
		case AttrTileFlags:
			tile.Flags, err = ps.ReadUint32()
			if err != nil {
				return nil, fmt.Errorf("otbm: reading tile flags: %w", err)
			}
		case AttrItem:
			itemID, err := ps.ReadUint16()
			if err != nil {
				return nil, fmt.Errorf("otbm: reading inline item ID: %w", err)
			}
			item := RawItem{ID: itemID}
			if err := parseItemAttrs(ps, &item); err != nil {
				return nil, err
			}
			tile.Items = append(tile.Items, item)
		default:
			// Unknown attribute with no length prefix; stop parsing props.
			return tile, nil
		}
	}

	// Parse item children.
	for _, child := range n.children {
		if child.nodeType == OTBMItem {
			item, err := parseItem(child)
			if err != nil {
				return nil, err
			}
			tile.Items = append(tile.Items, item)
		}
	}

	return tile, nil
}

// parseHouseTile parses an OTBM_HOUSETILE node.
func parseHouseTile(n *node, baseX, baseY uint16, baseZ uint8) (*Tile, error) {
	if len(n.props) < 6 {
		return nil, fmt.Errorf("otbm: house tile props too short: got %d bytes, need at least 6", len(n.props))
	}

	// House ID is at offset 2 (after the two offset bytes that parseTile will read).
	ps := propstream.NewPropStream(n.props[2:])
	houseID, err := ps.ReadUint32()
	if err != nil {
		return nil, fmt.Errorf("otbm: reading house ID: %w", err)
	}
	return parseTile(n, baseX, baseY, baseZ, houseID)
}

// parseItem parses an OTBM_ITEM node into a RawItem.
func parseItem(n *node) (RawItem, error) {
	ps := propstream.NewPropStream(n.props)

	id, err := ps.ReadUint16()
	if err != nil {
		return RawItem{}, fmt.Errorf("otbm: reading item ID: %w", err)
	}

	item := RawItem{ID: id}

	if err := parseItemAttrs(ps, &item); err != nil {
		return RawItem{}, err
	}

	// Parse sub-items from children.
	for _, child := range n.children {
		if child.nodeType == OTBMItem {
			subItem, err := parseItem(child)
			if err != nil {
				return RawItem{}, err
			}
			item.SubItems = append(item.SubItems, subItem)
		}
	}

	return item, nil
}

// parseItemAttrs parses item attributes from the PropStream.
// Item attributes have no per-attribute length prefix, so every type that
// appears in the binary must be handled to keep the stream aligned.
func parseItemAttrs(ps *propstream.PropStream, item *RawItem) error {
	for ps.Remaining() > 0 {
		attrType, err := ps.ReadUint8()
		if err != nil {
			return fmt.Errorf("otbm: reading item attr type: %w", err)
		}

		switch attrType {
		case AttrCount, AttrRuneCharges:
			item.Count, err = ps.ReadUint8()
			if err != nil {
				return fmt.Errorf("otbm: reading item count: %w", err)
			}
		case AttrActionID:
			item.ActionID, err = ps.ReadUint16()
			if err != nil {
				return fmt.Errorf("otbm: reading item action ID: %w", err)
			}
		case AttrUniqueID:
			item.UniqueID, err = ps.ReadUint16()
			if err != nil {
				return fmt.Errorf("otbm: reading item unique ID: %w", err)
			}
		case AttrText, AttrDesc, AttrWrittenBy:
			str, err := ps.ReadString()
			if err != nil {
				return fmt.Errorf("otbm: reading item string attr: %w", err)
			}
			if attrType == AttrText {
				item.Text = str
			}
		case AttrTeleDest:
			if err := ps.Skip(5); err != nil { // x(2) + y(2) + z(1)
				return fmt.Errorf("otbm: skipping teleport destination: %w", err)
			}
		case AttrDepotID, AttrCharges:
			if err := ps.Skip(2); err != nil {
				return fmt.Errorf("otbm: skipping uint16 attr: %w", err)
			}
		case AttrHouseDoorID, AttrDecayingState:
			if err := ps.Skip(1); err != nil {
				return fmt.Errorf("otbm: skipping uint8 attr: %w", err)
			}
		case AttrDuration, AttrWrittenDate, AttrSleeperGUID, AttrSleepStart:
			if err := ps.Skip(4); err != nil {
				return fmt.Errorf("otbm: skipping uint32 attr: %w", err)
			}
		default:
			// Unknown attribute with no length prefix; stop parsing.
			return nil
		}
	}
	return nil
}

// parseTowns parses the OTBM_TOWNS container node and its OTBM_TOWN children.
func parseTowns(n *node, m *Map) error {
	for _, child := range n.children {
		if child.nodeType != OTBMTown {
			continue
		}

		town, err := parseTown(child)
		if err != nil {
			return err
		}
		m.Towns = append(m.Towns, town)
	}
	return nil
}

// parseTown parses a single OTBM_TOWN node.
func parseTown(n *node) (Town, error) {
	// Props: id(4) + namelen(2) + name(namelen) + templeX(2) + templeY(2) + templeZ(1)
	if len(n.props) < 4 {
		return Town{}, errors.New("otbm: town props too short for ID")
	}

	ps := propstream.NewPropStream(n.props)

	id, err := ps.ReadUint32()
	if err != nil {
		return Town{}, fmt.Errorf("otbm: reading town ID: %w", err)
	}

	name, err := ps.ReadString()
	if err != nil {
		return Town{}, fmt.Errorf("otbm: reading town name: %w", err)
	}

	templeX, err := ps.ReadUint16()
	if err != nil {
		return Town{}, fmt.Errorf("otbm: reading town temple X: %w", err)
	}

	templeY, err := ps.ReadUint16()
	if err != nil {
		return Town{}, fmt.Errorf("otbm: reading town temple Y: %w", err)
	}

	templeZ, err := ps.ReadUint8()
	if err != nil {
		return Town{}, fmt.Errorf("otbm: reading town temple Z: %w", err)
	}

	return Town{
		ID:   id,
		Name: name,
		Temple: Position{
			X: templeX,
			Y: templeY,
			Z: templeZ,
		},
	}, nil
}

// parseWaypoints parses the OTBM_WAYPOINTS container node and its OTBM_WAYPOINT children.
func parseWaypoints(n *node, m *Map) error {
	for _, child := range n.children {
		if child.nodeType != OTBMWaypoint {
			continue
		}

		wp, err := parseWaypointNode(child)
		if err != nil {
			return err
		}
		m.Waypoints = append(m.Waypoints, wp)
	}
	return nil
}

// parseWaypointNode parses a single OTBM_WAYPOINT node.
func parseWaypointNode(n *node) (Waypoint, error) {
	// Props: namelen(2) + name(namelen) + x(2) + y(2) + z(1)
	ps := propstream.NewPropStream(n.props)

	name, err := ps.ReadString()
	if err != nil {
		return Waypoint{}, fmt.Errorf("otbm: reading waypoint name: %w", err)
	}

	x, err := ps.ReadUint16()
	if err != nil {
		return Waypoint{}, fmt.Errorf("otbm: reading waypoint X: %w", err)
	}

	y, err := ps.ReadUint16()
	if err != nil {
		return Waypoint{}, fmt.Errorf("otbm: reading waypoint Y: %w", err)
	}

	z, err := ps.ReadUint8()
	if err != nil {
		return Waypoint{}, fmt.Errorf("otbm: reading waypoint Z: %w", err)
	}

	return Waypoint{
		Name: name,
		Position: Position{
			X: x,
			Y: y,
			Z: z,
		},
	}, nil
}
