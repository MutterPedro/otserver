# OpenTibia Binary (OTB) Format

This document provides a comprehensive overview of the OpenTibia Binary (OTB) format. It is aimed at contributors to the `forgottenserver-go` project who want to customize the `pkg/otb` parser, add new attributes, or simply understand the binary structure used for item definitions.

## File Structure Overview

An OTB file defines a list of items and their properties. The format is a hierarchical, node-based structure prefixed by a simple file identifier. Note that data values use **Little Endian** byte order.

The top-level structure of the file looks like:

1. **File Identifier** (4 bytes): Typically `0x00 0x00 0x00 0x00` (ignored/skipped by the parser).
2. **Node Tree Data**: A stream of bytes that defines the root node and its children.

## Special Control Bytes

The OTB parser relies on three special control bytes to represent the node tree:

* `0xFE` (`NODE_START`): Indicates the start of a new node.
* `0xFF` (`NODE_END`): Indicates the end of the current node.
* `0xFD` (`ESCAPE`): The escape character used for data bytes that happen to have the same value as the control bytes.

### Escape Decoding (Crucial for Parsing)

Because item data (e.g., numerical values or strings) can contain bytes equal to `0xFE`, `0xFF`, or `0xFD`, those bytes must be escaped to prevent the parser from confusing them with structural markers.
When the parser encounters `0xFD`, the **next byte** is read literally as data, regardless of its value.

For example, if an item's server ID is `0x00FE` (254), its little-endian representation is `[0xFE, 0x00]`. In the OTB file, the `0xFE` byte will be escaped as `[0xFD, 0xFE]`.

## Node Tree Hierarchy

Every OTB file consists of a single **Root Node**. The root node acts as a container for all the individual item definitions (which are child nodes of the root).

### 1. Root Node

The hierarchy begins with a `NODE_START` (`0xFE`). The root node has:

* **Node Type**: `0x00` (A generic root type identifier)
* **Root Header Properties**: 145 bytes containing global file information.

The 145-byte root header breaks down as follows:

* `Flags` (4 bytes, `uint32`)
* `Attribute Type` (1 byte, usually `0x01` indicating version data)
* `Major Version` (4 bytes, `uint32`)
* `Minor Version` (4 bytes, `uint32`)
* `Build Number` (4 bytes, `uint32`)
* `CSDVersion` (128 bytes, zero-padded string)

Inside the root node's scope (before its corresponding `NODE_END`), all subsequent `NODE_START` sequences signify individual **Item Nodes**.

### 2. Item Nodes

An item node describes a single item. It begins with a `NODE_START` (`0xFE`).

* **Node Type (Item Group)**: The type byte immediately following `NODE_START` represents the item's grouping (e.g., weapon, armor, etc.).
* **Item Header Properties**: 4 bytes.
  * `Flags` (4 bytes, `uint32`, specific options for the item).
* **Item Attributes**: The remainder of the node consists of consecutive item attributes until a `NODE_END` (`0xFF`) is encountered.

Note: The `ClientID` is **not** part of the fixed header. It comes from the `ATTR_CLIENTID` (`0x11`) attribute in the attribute stream, matching the C++ `Items::loadFromOtb()` implementation.

#### Item Groups (Node Types)

The `pkg/otb` parser defines the following item groups (represented by a `uint8` value in the node type):

* `0` - None
* `1` - Ground
* `2` - Container
* `3` - Weapon
* `4` - Ammunition
* `5` - Armor
* `6` - Rune
* `7` - Teleport
* `8` - Magic Field
* `9` - Writeable
* `10` - Key
* `11` - Splash
* `12` - Fluid Container
* `13` - Door
* `14` - Depot

### 3. Item Attributes

Within an item node, attributes stream sequentially following the 4-byte flags header. Each attribute is defined by:

1. `Type` (1 byte, `uint8`): The identifier of the attribute.
2. `Length` (2 bytes, `uint16`, little-endian): The byte length of the upcoming attribute data.
3. `Data` (`Length` bytes): The unescaped value of the attribute.

#### Attribute Type Constants

The attribute type values are defined by the C++ `itemattrib_t` enum which auto-increments from `0x10`. The following table lists all constants implemented in `forgottenserver-go` (`pkg/otb/otb.go`):

| Constant Identifier | Hex Value | Length | Type | Description |
| --- | --- | --- | --- | --- |
| `attrServerID` | `0x10` | 2 bytes | `uint16` | The unique server-side identifier of the item. |
| `attrClientID` | `0x11` | 2 bytes | `uint16` | The graphical client ID mapping. |
| `attrName` | `0x12` | Variable | Raw bytes | The name of the item. The attribute data is the raw string bytes (length = `attrLen`). |
| `attrSpeed` | `0x14` | 2 bytes | `uint16` | The speed modifier (for ground items). |
| `attrWeight` | `0x17` | 4 bytes | `uint32` | The weight of the item in hundredths of an ounce. |
| `attrArmor` | `0x1A` | 2 bytes | `uint16` | The armor value. |
| `attrLight2` | `0x2A` | 4 bytes | `uint16,uint16` | Light emitted by the item. Contains Light Level (2 bytes) and Light Color (2 bytes). |
| `attrTopOrder` | `0x2B` | 1 byte | `uint8` | The always-on-top draw order. Parsed but not stored. |

The full C++ enum (`itemattrib_t` in `src/itemloader.h`) defines many more attribute types (slots, magic fields, deprecated weapon/armor variants, etc.). Unknown attribute types are silently skipped by the parser using the `attrLen` field to advance past the data.

### Example Walkthrough: Parsing a Sword

Imagine we extract the following raw bytes from an OTB file, representing a **"Wooden Sword"**:

```hex
FE 03 00 00 00 00 10 02 00 C8 00 11 02 00 64 00 12 0C 00 57 6F 6F 64 65 6E 20 53 77 6F 72 64 FF
```

Let's dissect this byte stream step-by-step:

1. **`FE` (NODE_START)**: A new item definition begins here.
2. **`03`**: Item Group `3` means this is a **Weapon**.
3. **`00 00 00 00`**: `Flags` (4 bytes). No special flags set.
4. **`10 02 00 C8 00`**: First attribute.
   * `10`: Type `0x10` is `attrServerID`.
   * `02 00`: Length is 2 bytes.
   * `C8 00`: Data is `200` (`0x00C8`). The item's **Server ID is 200**.
5. **`11 02 00 64 00`**: Second attribute.
   * `11`: Type `0x11` is `attrClientID`.
   * `02 00`: Length is 2 bytes.
   * `64 00`: Data is `100` (`0x0064`). The item's **Client ID is 100**.
6. **`12 0C 00 57 6F 6F 64 65 6E 20 53 77 6F 72 64`**: Third attribute.
   * `12`: Type `0x12` is `attrName`.
   * `0C 00`: Length is 12 bytes.
   * `57...64`: ASCII decoded, these 12 bytes read **"Wooden Sword"**.
7. **`FF` (NODE_END)**: The item definition is complete! The parser loops back to expect the next `NODE_START`.

## How the `pkg/otb` Parser Works

Contributors working on `pkg/otb/otb.go` should understand the parsing pipeline:

1. **Initial Read**: The 4-byte prefix is truncated and discarded.
2. **Node Tree Extraction (`parseNodes`)**: The hierarchical node tree is traversed. Crucially, that phase **simultaneously handles escape byte processing**. As it recursively reads `readNode()`, if it encounters `0xFD`, it skips the escape byte and consumes the following literal byte. The output is a cleanly structured tree of `node` structs containing unescaped data in `props`.
3. **Item Mapping (`Parse`)**: The tree structure is interpreted. The root node's 145-byte unescaped properties are validated.
4. **Item Building (`parseItemNode`)**: For every child branch (item node), the first 4 unescaped property bytes are the item flags (skipped), and a scanner iterates through the remaining stream to process item attributes one by one. Valid items are populated into a dictionary, indexed by their extracted `ServerID`.

### Contribution Tips & Pitfalls

* Whenever adding a parser feature for a new property, trace to ensure you extract the exact correct length bytes dictated by `attrLen`.
* Endianness errors: Ensure that integer byte extractions cast with `binary.LittleEndian.Uint16` or `Uint32`.
* Avoid adding decoding logic directly into `parseItemNode`. The lower-level tree builder `parseNodes` fundamentally extracts actual bytes and handles the `0xFD` un-escaping phase.
* **TDD Approach**: `pkg/otb/otb_test.go` has robust synthetic builders (like `buildOTBFile`) which allow you to serialize mock items and test the parser thoroughly. Take advantage of `testItem`, `testAttr`, and custom builders inside `otb_test.go` when adding logic for new Attribute IDs or Item Groups!
* **Attribute values**: The attribute type constants come from the C++ `itemattrib_t` enum in `src/itemloader.h` which auto-increments from `0x10`. Always verify against the C++ source when adding new attributes.
