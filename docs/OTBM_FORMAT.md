# OpenTibia Map (OTBM) Format

This document provides a comprehensive overview of the OpenTibia Map (OTBM) format. It complements the OTB format and is used specifically for parsing the world map (tiles, items, spawns, houses, and towns). This guide is aimed at contributors working on the `internal/iomap` package.

## File Structure Overview

Like the OTB format, an OTBM file is a generic tree of nodes prefixed by a 4-byte file identifier. All numerical data is encoded in **Little Endian** byte order.

1. **File Identifier** (4 bytes): Typically `0x00 0x00 0x00 0x00` (ignored/skipped by the parser).
2. **Node Tree Data**: A hierarchical stream of bytes utilizing the exact same escape logic and control bytes as OTB files.

## Special Control Bytes

The OTBM parser relies on the exact same three special control bytes as the OTB parser:

* `0xFE` (`NODE_START`): Indicates the start of a new node.
* `0xFF` (`NODE_END`): Indicates the end of the current node.
* `0xFD` (`ESCAPE`): The escape character used to literalize data bytes that happen to have the same value as the control bytes (e.g., parsing `[0xFD, 0xFE]` as a literal `0xFE` value).

## Node Tree Hierarchy

Every OTBM file starts with a single **Map Header Node** containing global map dimensions, which then typically houses a **Map Data Node**. The Map Data Node acts as the container for all geographical layout nodes (Tile Areas, Towns, Waypoints).

### Node Types

The `internal/iomap` package defines the following OTBM node types (each is a `uint8` byte immediately following a `NODE_START`):

| Constant | Hex Value | Description |
|---|---|---|
| `otbmMapHeader` | `0x00` | The root node of the file containing map dimensions. |
| `otbmMapData` | `0x02` | Container node for areas, towns, and waypoints. |
| `otbmTileArea` | `0x04` | A 256x256 coordinate sector containing individual Tiles. |
| `otbmTile` | `0x05` | An individual map square coordinate. |
| `otbmItem` | `0x06` | An item placed on a tile (or inside a container item). |
| `otbmTowns` | `0x0C` | Container node for Town definitions. |
| `otbmTown` | `0x0D` | A single town definition (temple coordinates). |
| `otbmHouseTile` | `0x0E` | An individual map square that belongs to a house. |
| `otbmWaypoints` | `0x0F` | Container node for Waypoint definitions. |
| `otbmWaypoint` | `0x10` | A specific named coordinate marker. |

---

## Detailed Node Structures

### 1. Map Header (`0x00`)

The very first node in the file.

* **Header Properties (16 bytes)**:
  * `Version` (4 bytes, `uint32`)
  * `Width` (2 bytes, `uint16`)
  * `Height` (2 bytes, `uint16`)
  * `MajorItemsVersion` (4 bytes, `uint32`)
  * `MinorItemsVersion` (4 bytes, `uint32`)
* **Children**: Expects an `otbmMapData` node.

### 2. Map Data (`0x02`)

Usually the sole child of the Map Header.

* **Attributes**: Sequential attributes encoded as `[Type (1 byte)] [Length (2 bytes)] [Data]`.
  * `attrDescription` (`0x01`): Text description of the map.
  * `attrExtFile` (`0x02`): External file reference string.
  * `attrSpawnFile` (`0x0B` / 11): String filename for the spawn XML.
  * `attrHouseFile` (`0x0D` / 13): String filename for the house XML.
* **Children**: Expects `otbmTileArea`, `otbmTowns`, and `otbmWaypoints` nodes.

### 3. Tile Area (`0x04`)

Map data is partitioned into sector nodes to optimize geographical parsing.

* **Area Properties (5 bytes)**:
  * `BaseX` (2 bytes, `uint16`): The base X coordinate for tiles in this area.
  * `BaseY` (2 bytes, `uint16`): The base Y coordinate for tiles in this area.
  * `BaseZ` (1 byte, `uint8`): The Z limit/floor for this area.
* **Children**: Contains `otbmTile` or `otbmHouseTile` nodes.

### 4. Tiles (`0x05`) and House Tiles (`0x0E`)

A specific coordinate square on the map.

* **Tile Properties (2 bytes)**:
  * `OffsetX` (1 byte, `uint8`): Added to the Tile Area's `BaseX` for final X coordinate.
  * `OffsetY` (1 byte, `uint8`): Added to the Tile Area's `BaseY` for final Y coordinate.
* **House Tile Specifics**: If the node is `otbmHouseTile`, the properties are 6 bytes long. The extra 4 bytes comprise the `HouseID` (`uint32`).
* **Tile Attributes**:
  * `attrTileFlags` (`0x03`): Followed by 4 bytes (`uint32`) of flags.
  * `attrItem` (`0x09`): Inline item. Followed by item ID (`uint16`) and item attributes (no per-attribute length prefix).
* **Children**: Contains `otbmItem` nodes.

### 5. Items (`0x06`)

Items placed on tiles. Note that items can be heavily nested (items inside container items).

* **Item Properties (2 bytes)**:
  * `ID` (2 bytes, `uint16`): The server ID of the item matching OTB definitions.
* **Item Attributes** (no per-attribute length prefix — every type has a known fixed or variable size):
  * `attrCount` (`0x0F` / 15): 1 byte (`uint8`). The stack count/subtype.
  * `attrRuneCharges` (`0x0C` / 12): 1 byte (`uint8`). Rune charges.
  * `attrActionID` (`0x04`): 2 bytes (`uint16`). Scripting Action ID.
  * `attrUniqueID` (`0x05`): 2 bytes (`uint16`). Scripting Unique ID.
  * `attrText` (`0x06`): Variable length string (`uint16` length prefix + data). Written text on items.
  * `attrDesc` (`0x07`): Variable length string (`uint16` length prefix + data). Item description.
  * `attrWrittenBy` (`0x13` / 19): Variable length string (`uint16` length prefix + data). Author name.
  * `attrTeleDest` (`0x08`): 5 bytes. Teleport destination: x(`uint16`) + y(`uint16`) + z(`uint8`).
  * `attrDepotID` (`0x0A` / 10): 2 bytes (`uint16`). Depot ID.
  * `attrCharges` (`0x16` / 22): 2 bytes (`uint16`). Item charges.
  * `attrHouseDoorID` (`0x0E` / 14): 1 byte (`uint8`). House door ID.
  * `attrDecayingState` (`0x11` / 17): 1 byte (`uint8`). Decaying state.
  * `attrDuration` (`0x10` / 16): 4 bytes (`uint32`). Duration in milliseconds.
  * `attrWrittenDate` (`0x12` / 18): 4 bytes (`uint32`). Written date timestamp.
  * `attrSleeperGUID` (`0x14` / 20): 4 bytes (`uint32`). Sleeper GUID for beds.
  * `attrSleepStart` (`0x15` / 21): 4 bytes (`uint32`). Sleep start timestamp.
* **Children**: Nested `otbmItem` nodes (e.g., items within a bag or chest).

### Example Walkthrough: Parsing a Map Data Node

Imagine the parser traverses into the `otbmMapData` node and encounters the following unescaped properties and child nodes:

```hex
01 10 00 54 68 69 73 20 69 73 20 61 20 74 65 73 74 20 6D 61 70 FE 04 64 00 64 00 07 FE 05 02 02 03 01 00 00 00 FE 06 64 00 FF FF FF
```

Let's dissect this OTBM branch:

1. **`01 10 00 54 68 ...`**: Attributes on the `otbmMapData` node!
   * `01`: Type `0x01` is `attrDescription`.
   * `10 00`: Length is 16 bytes.
   * `54...70`: ASCII decoded as "This is a test map".
2. **`FE 04`**: Start of a child node. `04` is a **Tile Area**.
3. **`64 00 64 00 07`**: Tile Area properties.
   * `BaseX` = 100 (`0x0064`)
   * `BaseY` = 100 (`0x0064`)
   * `BaseZ` = 7
4. **`FE 05`**: Start of a nested child node. `05` is a **Tile**.
5. **`02 02`**: Tile coordinate offsets. `OffsetX` = 2, `OffsetY` = 2.
   * *Actual Map Coordinate* = (100+2, 100+2, 7) = **(102, 102, 7)**.
6. **`03 01 00 00 00`**: Tile Attribute!
   * `03`: Type `0x03` is `attrTileFlags`.
   * `01 00 00 00`: Tile flag bitmask = `1`.
7. **`FE 06`**: Start of a nested child node. `06` is an **Item**.
8. **`64 00`**: Item properties. Item Server ID = 100 (`0x0064`).
9. **`FF`**: End of Item Node.
10. **`FF`**: End of Tile Node.
11. **`FF`**: End of Tile Area Node.

---

## Attribute Type Constants Reference

All attribute type constants come from the C++ enums in `src/iomap.h` (`OTBM_ATTR_*`) and `src/item.h` (`ATTR_*`). The following table lists every constant implemented in the `internal/iomap` package:

| Constant Identifier | Value | Data Size | Description |
| --- | --- | --- | --- |
| `attrDescription` | 1 | Variable (uint16 len + data) | Map or item description string. |
| `attrExtFile` | 2 | Variable (uint16 len + data) | External file reference string. |
| `attrTileFlags` | 3 | 4 bytes (uint32) | Tile flag bitmask (protection zone, no-logout, etc.). |
| `attrActionID` | 4 | 2 bytes (uint16) | Scripting Action ID for items. |
| `attrUniqueID` | 5 | 2 bytes (uint16) | Scripting Unique ID for items. |
| `attrText` | 6 | Variable (uint16 len + data) | Written text on items (books, signs). |
| `attrDesc` | 7 | Variable (uint16 len + data) | Item description. |
| `attrTeleDest` | 8 | 5 bytes (uint16+uint16+uint8) | Teleport destination (x, y, z). |
| `attrItem` | 9 | Variable | Inline item in tile props: item ID (uint16) followed by item attributes. |
| `attrDepotID` | 10 | 2 bytes (uint16) | Depot ID. |
| `attrSpawnFile` | 11 | Variable (uint16 len + data) | Spawn XML filename (map data attribute). |
| `attrRuneCharges` | 12 | 1 byte (uint8) | Rune charges. |
| `attrHouseFile` | 13 | Variable (uint16 len + data) | House XML filename (map data attribute). |
| `attrHouseDoorID` | 14 | 1 byte (uint8) | House door ID. |
| `attrCount` | 15 | 1 byte (uint8) | Item stack count / subtype. |
| `attrDuration` | 16 | 4 bytes (uint32) | Duration in milliseconds. |
| `attrDecayingState` | 17 | 1 byte (uint8) | Decaying state. |
| `attrWrittenDate` | 18 | 4 bytes (uint32) | Written date timestamp. |
| `attrWrittenBy` | 19 | Variable (uint16 len + data) | Author name string. |
| `attrSleeperGUID` | 20 | 4 bytes (uint32) | Sleeper GUID for bed items. |
| `attrSleepStart` | 21 | 4 bytes (uint32) | Sleep start timestamp. |
| `attrCharges` | 22 | 2 bytes (uint16) | Item charges. |

### Important: Item Attributes Have No Per-Attribute Length Prefix

Unlike OTB item attributes (which include a `uint16` length field per attribute), OTBM item attributes have **no per-attribute length prefix**. Each attribute type has a known, fixed data size (or a `uint16` string length prefix for variable-length strings). This means **every attribute type that appears in a real OTBM file must be explicitly handled** by the parser — encountering an unknown attribute type makes it impossible to determine the data size, breaking stream alignment for all subsequent attributes.

## How the `internal/iomap` Parser Works

1. **Initial Read**: The 4-byte file identifier is skipped.
2. **Node Tree Extraction (`parseNodes`)**: Identical to the OTB parser — traverses the stream handling `NODE_START`, `NODE_END`, and `ESCAPE` bytes recursively. Produces a clean tree of `node` structs with unescaped `props` data.
3. **Map Header Parsing (`LoadMap`)**: Validates the root node as `otbmMapHeader` and reads the 16-byte header (version, width, height, item versions).
4. **Map Data Attributes (`parseMapDataAttrs`)**: Parses string attributes (description, spawn file, house file, ext file) from the `otbmMapData` node props.
5. **Tile Area Processing (`parseTileArea`)**: Reads the 5-byte area base coordinates, then processes each child tile.
6. **Tile Parsing (`parseTile`)**: Reads offset coordinates, tile attributes (flags, inline items), and item children.
7. **Item Parsing (`parseItem` / `parseItemAttrs`)**: Reads the item ID, then streams through item attributes. `parseItemAttrs` is a shared function used for both regular item nodes and inline items (via `attrItem` in tile props).

### Contribution Tips & Pitfalls

* **Attribute type values** come from the C++ enums in `src/iomap.h` and `src/item.h`. Always verify against the C++ source when adding new attributes.
* **No length prefix on item attributes**: If a new attribute type appears in real OTBM files, you must add a handler to `parseItemAttrs` — the parser cannot skip unknown types without knowing their data size.
* **Inline items** (`attrItem` in tile props) use the same attribute parsing logic as regular item nodes via the shared `parseItemAttrs` function.
* **TDD Approach**: `internal/iomap/iomap_test.go` has builders for creating synthetic OTBM files. Use these when adding support for new attribute types or node types.
