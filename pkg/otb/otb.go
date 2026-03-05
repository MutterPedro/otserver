// Package otb handles parsing the OTB binary file to in memory item types.
package otb

import (
	"errors"
	"fmt"

	"github.com/MutterPedro/otserver/pkg/propstream"
)

const (
	nodeStart  byte = 0xFE
	nodeEnd    byte = 0xFF
	escapeChar byte = 0xFD
)

// Item attribute type constants from the OTB format.
// Values match the C++ itemattrib_t enum which auto-increments from 0x10.
const (
	attrServerID uint8 = 0x10
	attrClientID uint8 = 0x11
	attrName     uint8 = 0x12
	attrSpeed    uint8 = 0x14
	attrWeight   uint8 = 0x17
	attrArmor    uint8 = 0x1A
	attrLight2   uint8 = 0x2A
	attrTopOrder uint8 = 0x2B
)

// ItemGroup represents the type/group of an OTB item.
type ItemGroup uint8

// ItemGroup represents the type/group of an OTB item.
const (
	ItemGroupNone           ItemGroup = 0
	ItemGroupGround         ItemGroup = 1
	ItemGroupContainer      ItemGroup = 2
	ItemGroupWeapon         ItemGroup = 3
	ItemGroupAmmunition     ItemGroup = 4
	ItemGroupArmor          ItemGroup = 5
	ItemGroupRune           ItemGroup = 6
	ItemGroupTeleport       ItemGroup = 7
	ItemGroupMagicField     ItemGroup = 8
	ItemGroupWriteable      ItemGroup = 9
	ItemGroupKey            ItemGroup = 10
	ItemGroupSplash         ItemGroup = 11
	ItemGroupFluidContainer ItemGroup = 12
	ItemGroupDoor           ItemGroup = 13
	ItemGroupDepot          ItemGroup = 14
)

// ItemType holds the parsed data for a single OTB item.
type ItemType struct {
	ServerID   uint16
	ClientID   uint16
	Name       string
	Group      ItemGroup
	Speed      uint16
	Weight     uint32
	Armor      uint16
	Attack     uint16
	Defense    uint16
	LightLevel uint16
	LightColor uint16
}

// node represents a parsed node in the OTB tree.
type node struct {
	nodeType byte
	props    []byte
	children []*node
}

// parseNodes parses the raw byte stream (after the 4-byte file identifier) into
// a tree of nodes, handling escape bytes and NODE_START/NODE_END markers.
func parseNodes(data []byte) (*node, error) {
	pos := 0

	if pos >= len(data) || data[pos] != nodeStart {
		return nil, errors.New("otb: expected NODE_START at beginning of node tree")
	}
	pos++

	root, newPos, err := readNode(data, pos)
	if err != nil {
		return nil, err
	}
	pos = newPos

	if pos != len(data) {
		return nil, fmt.Errorf("otb: trailing data after root node (%d extra bytes)", len(data)-pos)
	}

	return root, nil
}

// readNode reads a single node starting at data[pos]. The caller has already
// consumed the NODE_START byte. pos points to the node type byte.
func readNode(data []byte, pos int) (*node, int, error) {
	if pos >= len(data) {
		return nil, 0, errors.New("otb: unexpected end of data reading node type")
	}

	n := &node{nodeType: data[pos]}
	pos++

	// Read props (unescaped data bytes) until we hit a control byte.
	for pos < len(data) {
		b := data[pos]
		switch b {
		case nodeStart:
			// Start of a child node.
			pos++ // consume NODE_START
			child, newPos, err := readNode(data, pos)
			if err != nil {
				return nil, 0, err
			}
			n.children = append(n.children, child)
			pos = newPos
		case nodeEnd:
			pos++ // consume NODE_END
			return n, pos, nil
		case escapeChar:
			pos++ // consume escape byte
			if pos >= len(data) {
				return nil, 0, errors.New("otb: dangling escape byte at end of data")
			}
			n.props = append(n.props, data[pos])
			pos++
		default:
			n.props = append(n.props, b)
			pos++
		}
	}

	return nil, 0, errors.New("otb: unexpected end of data, missing NODE_END")
}

// Parse parses an OTB binary file and returns a map of items indexed by ServerID.
func Parse(data []byte) (map[uint16]*ItemType, error) {
	if len(data) < 4 {
		return nil, errors.New("otb: data too short for file identifier")
	}

	// Skip the 4-byte file identifier.
	root, err := parseNodes(data[4:])
	if err != nil {
		return nil, err
	}

	// Parse root node header.
	if err := parseRootHeader(root.props); err != nil {
		return nil, err
	}

	// Parse each child node as an item.
	items := make(map[uint16]*ItemType)
	for _, child := range root.children {
		item, err := parseItemNode(child)
		if err != nil {
			return nil, err
		}
		items[item.ServerID] = item
	}

	return items, nil
}

// parseRootHeader validates the root node's header data.
func parseRootHeader(props []byte) error {
	// Root header: flags(4) + attr(1) + majorVersion(4) + minorVersion(4) + buildNumber(4) + CSDVersion(128) = 145 bytes
	const rootHeaderSize = 4 + 1 + 4 + 4 + 4 + 128
	if len(props) < rootHeaderSize {
		return fmt.Errorf("otb: root header too short: got %d bytes, need %d", len(props), rootHeaderSize)
	}
	return nil
}

// parseItemNode parses a child node into an ItemType.
func parseItemNode(n *node) (*ItemType, error) {
	item := &ItemType{
		Group: ItemGroup(n.nodeType),
	}

	ps := propstream.NewPropStream(n.props)

	// Item props start with: flags(4), then attributes follow.
	if err := ps.Skip(4); err != nil {
		return nil, fmt.Errorf("otb: item node props too short: %w", err)
	}

	// Parse attributes: each is type(1) + length(2) + data(length).
	for ps.Remaining() > 0 {
		attrType, err := ps.ReadUint8()
		if err != nil {
			return nil, fmt.Errorf("otb: reading item attr type: %w", err)
		}

		attrLen, err := ps.ReadUint16()
		if err != nil {
			return nil, fmt.Errorf("otb: reading item attr length: %w", err)
		}

		attrData, err := ps.ReadBytes(int(attrLen))
		if err != nil {
			return nil, fmt.Errorf("otb: attribute length %d exceeds remaining data: %w", attrLen, err)
		}

		ads := propstream.NewPropStream(attrData)

		switch attrType {
		case attrServerID:
			item.ServerID, err = ads.ReadUint16()
			if err != nil {
				return nil, fmt.Errorf("otb: reading ATTR_SERVERID: %w", err)
			}
		case attrClientID:
			item.ClientID, err = ads.ReadUint16()
			if err != nil {
				return nil, fmt.Errorf("otb: reading ATTR_CLIENTID: %w", err)
			}
		case attrName:
			item.Name = string(attrData)
		case attrSpeed:
			item.Speed, err = ads.ReadUint16()
			if err != nil {
				return nil, fmt.Errorf("otb: reading ATTR_SPEED: %w", err)
			}
		case attrWeight:
			item.Weight, err = ads.ReadUint32()
			if err != nil {
				return nil, fmt.Errorf("otb: reading ATTR_WEIGHT: %w", err)
			}
		case attrArmor:
			item.Armor, err = ads.ReadUint16()
			if err != nil {
				return nil, fmt.Errorf("otb: reading ATTR_ARMOR: %w", err)
			}
		case attrLight2:
			item.LightLevel, err = ads.ReadUint16()
			if err != nil {
				return nil, fmt.Errorf("otb: reading ATTR_LIGHT2 level: %w", err)
			}
			item.LightColor, err = ads.ReadUint16()
			if err != nil {
				return nil, fmt.Errorf("otb: reading ATTR_LIGHT2 color: %w", err)
			}
		case attrTopOrder:
			// Parsed but not stored; skip.
		}
	}

	return item, nil
}
