// Package entity defines core domain types for the game server.
package entity

import (
	"errors"
	"fmt"
)

// ErrDifferentFloor is returned when computing distance between positions on different Z levels.
var ErrDifferentFloor = errors.New("positions are on different floors")

// MaxFloor represents the maximum floor number in the game.
const MaxFloor = 15

// Position represents a coordinate on the Tibia game map.
type Position struct {
	X uint16
	Y uint16
	Z uint8
}

// IsValid reports whether the position is a valid map coordinate.
// A position is valid when X and Y are both non-zero and Z is in the range 0–MaxFloor.
func (p Position) IsValid() bool {
	return p.X > 0 && p.Y > 0 && p.Z <= MaxFloor
}

// Distance returns the Chebyshev (chessboard) distance between two positions.
// It returns ErrDifferentFloor if the positions are on different Z levels.
func (p Position) Distance(other Position) (int, error) {
	if p.Z != other.Z {
		return 0, ErrDifferentFloor
	}

	dx := int(p.X) - int(other.X)
	if dx < 0 {
		dx = -dx
	}

	dy := int(p.Y) - int(other.Y)
	if dy < 0 {
		dy = -dy
	}

	if dx > dy {
		return dx, nil
	}
	return dy, nil
}

// IsAdjacentTo reports whether the two positions are directly adjacent
// (including diagonals) on the same floor. A position is not adjacent to itself.
func (p Position) IsAdjacentTo(other Position) bool {
	if p.Z != other.Z {
		return false
	}

	dx := int(p.X) - int(other.X)
	if dx < 0 {
		dx = -dx
	}

	dy := int(p.Y) - int(other.Y)
	if dy < 0 {
		dy = -dy
	}

	return dx <= 1 && dy <= 1 && (dx+dy) > 0
}

// Translate returns a new Position offset by (dx, dy, dz).
// Overflow and underflow wrap around, matching C++ unsigned integer behavior.
func (p Position) Translate(dx, dy int, dz int8) Position {
	return Position{
		X: uint16(int(p.X) + dx),
		Y: uint16(int(p.Y) + dy),
		Z: uint8(int8(p.Z) + dz),
	}
}

// Direction represents a cardinal or diagonal facing direction.
type Direction uint8

// Direction constants
const (
	DirectionNorth     Direction = iota // 0
	DirectionEast                       // 1
	DirectionSouth                      // 2
	DirectionWest                       // 3
	DirectionSouthWest                  // 4
	DirectionSouthEast                  // 5
	DirectionNorthWest                  // 6
	DirectionNorthEast                  // 7
)

// Opposite returns the direction that faces the opposite way.
func (d Direction) Opposite() Direction {
	switch d {
	case DirectionNorth:
		return DirectionSouth
	case DirectionSouth:
		return DirectionNorth
	case DirectionEast:
		return DirectionWest
	case DirectionWest:
		return DirectionEast
	case DirectionSouthWest:
		return DirectionNorthEast
	case DirectionNorthEast:
		return DirectionSouthWest
	case DirectionSouthEast:
		return DirectionNorthWest
	case DirectionNorthWest:
		return DirectionSouthEast
	default:
		return d
	}
}

// IsValid reports whether the direction is a recognized direction value (0–7).
func (d Direction) IsValid() bool {
	return d <= DirectionNorthEast
}

// ---------------------------------------------------------------------------
// ReturnValue
// ---------------------------------------------------------------------------

// ReturnValue represents a result code returned by game actions.
// Values must match C++ enums.h exactly for wire protocol compatibility.
type ReturnValue uint8

// ReturnValue constants
const (
	RetNoError                                     ReturnValue = 0
	RetNotPossible                                 ReturnValue = 1
	RetNotEnoughRoom                               ReturnValue = 2
	RetPlayerIsPzLocked                            ReturnValue = 3
	RetPlayerIsNotInvited                          ReturnValue = 4
	RetCannotThrow                                 ReturnValue = 5
	RetThereIsNoWay                                ReturnValue = 6
	RetDestinationOutOfReach                       ReturnValue = 7
	RetCreatureBlock                               ReturnValue = 8
	RetNotMoveable                                 ReturnValue = 9
	RetDropTwoHandedItem                           ReturnValue = 10
	RetBothHandsNeedToBeFree                       ReturnValue = 11
	RetCanOnlyUseOneWeapon                         ReturnValue = 12
	RetNeedExchange                                ReturnValue = 13
	RetCannotBeDressed                             ReturnValue = 14
	RetPutThisObjectInYourHand                     ReturnValue = 15
	RetPutThisObjectInBothHands                    ReturnValue = 16
	RetTooFarAway                                  ReturnValue = 17
	RetFirstGoDownstairs                           ReturnValue = 18
	RetFirstGoUpstairs                             ReturnValue = 19
	RetContainerNotEnoughRoom                      ReturnValue = 20
	RetNotEnoughCapacity                           ReturnValue = 21
	RetCannotPickup                                ReturnValue = 22
	RetThisIsImpossible                            ReturnValue = 23
	RetDepotIsFull                                 ReturnValue = 24
	RetCreatureDoesNotExist                        ReturnValue = 25
	RetCannotUseThisObject                         ReturnValue = 26
	RetPlayerWithThisNameIsNotOnline               ReturnValue = 27
	RetNotRequiredLevelToUseRune                   ReturnValue = 28
	RetYouAreAlreadyTrading                        ReturnValue = 29
	RetThisPlayerIsAlreadyTrading                  ReturnValue = 30
	RetYouMayNotLogoutDuringAFight                 ReturnValue = 31
	RetDirectPlayerShoot                           ReturnValue = 32
	RetNotEnoughLevel                              ReturnValue = 33
	RetNotEnoughMagicLevel                         ReturnValue = 34
	RetNotEnoughMana                               ReturnValue = 35
	RetNotEnoughSoul                               ReturnValue = 36
	RetYouAreExhausted                             ReturnValue = 37
	RetPlayerIsNotReachable                        ReturnValue = 38
	RetCreatureIsNotReachable                      ReturnValue = 39
	RetActionNotPermittedInProtectionZone          ReturnValue = 40
	RetYouMayNotAttackThisPlayer                   ReturnValue = 41
	RetYouMayNotAttackAPersonInProtectionZone      ReturnValue = 42
	RetYouMayNotAttackAPersonWhileInProtectionZone ReturnValue = 43
	RetYouMayNotAttackThisCreature                 ReturnValue = 44
	RetYouCanOnlyUseItOnCreatures                  ReturnValue = 45
	RetCreatureIsNotReachable2                     ReturnValue = 46
	RetTurnSecureModeToAttackUnmarkedPlayers       ReturnValue = 47
	RetYouNeedPremiumAccount                       ReturnValue = 48
	RetYouNeedToLearnThisSpell                     ReturnValue = 49
	RetYourVocationCannotUseThisSpell              ReturnValue = 50
	RetYouNeedAWeaponToUseThisSpell                ReturnValue = 51
	RetPlayerIsPzLockedLeavePvpZone                ReturnValue = 52
	RetPlayerIsPzLockedEnterPvpZone                ReturnValue = 53
	RetActionNotPermittedInANoPvpZone              ReturnValue = 54
	RetYouCannotLogoutHere                         ReturnValue = 55
	RetYouNeedAMagicItemToCastSpell                ReturnValue = 56
	RetCannotConjureItemHere                       ReturnValue = 57
	RetYouNeedToSplitYourSpears                    ReturnValue = 58
	RetNameIsTooAmbiguous                          ReturnValue = 59
	RetCanOnlyUseOneShield                         ReturnValue = 60
	RetNoPartyMembers                              ReturnValue = 61
	RetNotInvited                                  ReturnValue = 62
)

// String returns a human-readable name for the ReturnValue.
func (r ReturnValue) String() string {
	switch r {
	case RetNoError:
		return "RetNoError"
	case RetNotPossible:
		return "RetNotPossible"
	case RetNotEnoughRoom:
		return "RetNotEnoughRoom"
	case RetPlayerIsPzLocked:
		return "RetPlayerIsPzLocked"
	case RetPlayerIsNotInvited:
		return "RetPlayerIsNotInvited"
	case RetCannotThrow:
		return "RetCannotThrow"
	case RetThereIsNoWay:
		return "RetThereIsNoWay"
	case RetDestinationOutOfReach:
		return "RetDestinationOutOfReach"
	case RetCreatureBlock:
		return "RetCreatureBlock"
	case RetNotMoveable:
		return "RetNotMoveable"
	case RetDropTwoHandedItem:
		return "RetDropTwoHandedItem"
	case RetBothHandsNeedToBeFree:
		return "RetBothHandsNeedToBeFree"
	case RetCanOnlyUseOneWeapon:
		return "RetCanOnlyUseOneWeapon"
	case RetNeedExchange:
		return "RetNeedExchange"
	case RetCannotBeDressed:
		return "RetCannotBeDressed"
	case RetPutThisObjectInYourHand:
		return "RetPutThisObjectInYourHand"
	case RetPutThisObjectInBothHands:
		return "RetPutThisObjectInBothHands"
	case RetTooFarAway:
		return "RetTooFarAway"
	case RetFirstGoDownstairs:
		return "RetFirstGoDownstairs"
	case RetFirstGoUpstairs:
		return "RetFirstGoUpstairs"
	case RetContainerNotEnoughRoom:
		return "RetContainerNotEnoughRoom"
	case RetNotEnoughCapacity:
		return "RetNotEnoughCapacity"
	case RetCannotPickup:
		return "RetCannotPickup"
	case RetThisIsImpossible:
		return "RetThisIsImpossible"
	case RetDepotIsFull:
		return "RetDepotIsFull"
	case RetCreatureDoesNotExist:
		return "RetCreatureDoesNotExist"
	case RetCannotUseThisObject:
		return "RetCannotUseThisObject"
	case RetPlayerWithThisNameIsNotOnline:
		return "RetPlayerWithThisNameIsNotOnline"
	case RetNotRequiredLevelToUseRune:
		return "RetNotRequiredLevelToUseRune"
	case RetYouAreAlreadyTrading:
		return "RetYouAreAlreadyTrading"
	case RetThisPlayerIsAlreadyTrading:
		return "RetThisPlayerIsAlreadyTrading"
	case RetYouMayNotLogoutDuringAFight:
		return "RetYouMayNotLogoutDuringAFight"
	case RetDirectPlayerShoot:
		return "RetDirectPlayerShoot"
	case RetNotEnoughLevel:
		return "RetNotEnoughLevel"
	case RetNotEnoughMagicLevel:
		return "RetNotEnoughMagicLevel"
	case RetNotEnoughMana:
		return "RetNotEnoughMana"
	case RetNotEnoughSoul:
		return "RetNotEnoughSoul"
	case RetYouAreExhausted:
		return "RetYouAreExhausted"
	case RetPlayerIsNotReachable:
		return "RetPlayerIsNotReachable"
	case RetCreatureIsNotReachable:
		return "RetCreatureIsNotReachable"
	case RetActionNotPermittedInProtectionZone:
		return "RetActionNotPermittedInProtectionZone"
	case RetYouMayNotAttackThisPlayer:
		return "RetYouMayNotAttackThisPlayer"
	case RetYouMayNotAttackAPersonInProtectionZone:
		return "RetYouMayNotAttackAPersonInProtectionZone"
	case RetYouMayNotAttackAPersonWhileInProtectionZone:
		return "RetYouMayNotAttackAPersonWhileInProtectionZone"
	case RetYouMayNotAttackThisCreature:
		return "RetYouMayNotAttackThisCreature"
	case RetYouCanOnlyUseItOnCreatures:
		return "RetYouCanOnlyUseItOnCreatures"
	case RetCreatureIsNotReachable2:
		return "RetCreatureIsNotReachable2"
	case RetTurnSecureModeToAttackUnmarkedPlayers:
		return "RetTurnSecureModeToAttackUnmarkedPlayers"
	case RetYouNeedPremiumAccount:
		return "RetYouNeedPremiumAccount"
	case RetYouNeedToLearnThisSpell:
		return "RetYouNeedToLearnThisSpell"
	case RetYourVocationCannotUseThisSpell:
		return "RetYourVocationCannotUseThisSpell"
	case RetYouNeedAWeaponToUseThisSpell:
		return "RetYouNeedAWeaponToUseThisSpell"
	case RetPlayerIsPzLockedLeavePvpZone:
		return "RetPlayerIsPzLockedLeavePvpZone"
	case RetPlayerIsPzLockedEnterPvpZone:
		return "RetPlayerIsPzLockedEnterPvpZone"
	case RetActionNotPermittedInANoPvpZone:
		return "RetActionNotPermittedInANoPvpZone"
	case RetYouCannotLogoutHere:
		return "RetYouCannotLogoutHere"
	case RetYouNeedAMagicItemToCastSpell:
		return "RetYouNeedAMagicItemToCastSpell"
	case RetCannotConjureItemHere:
		return "RetCannotConjureItemHere"
	case RetYouNeedToSplitYourSpears:
		return "RetYouNeedToSplitYourSpears"
	case RetNameIsTooAmbiguous:
		return "RetNameIsTooAmbiguous"
	case RetCanOnlyUseOneShield:
		return "RetCanOnlyUseOneShield"
	case RetNoPartyMembers:
		return "RetNoPartyMembers"
	case RetNotInvited:
		return "RetNotInvited"
	default:
		return fmt.Sprintf("ReturnValue(%d)", r)
	}
}

// ---------------------------------------------------------------------------
// Outfit
// ---------------------------------------------------------------------------

// Outfit represents a creature's visual appearance.
type Outfit struct {
	LookType   uint16
	LookHead   uint8
	LookBody   uint8
	LookLegs   uint8
	LookFeet   uint8
	LookAddons uint8
	LookMount  uint16
}

// ---------------------------------------------------------------------------
// StackPos
// ---------------------------------------------------------------------------

// StackPos represents an item's position within a tile stack.
type StackPos uint8

// StackPosAny is a sentinel value meaning any stack position.
const StackPosAny StackPos = 255

// ---------------------------------------------------------------------------
// Slot
// ---------------------------------------------------------------------------

// Slot represents an inventory equipment slot.
type Slot uint8

// Slot constants
const (
	SlotFirst Slot = iota // 0
	SlotHead              // 1
	SlotNeck              // 2
	SlotBack              // 3
	SlotBody              // 4
	SlotRight             // 5
	SlotLeft              // 6
	SlotLegs              // 7
	SlotFeet              // 8
	SlotRing              // 9
	SlotAmmo              // 10
	SlotLast              // 11
)

// ---------------------------------------------------------------------------
// SpeakType
// ---------------------------------------------------------------------------

// SpeakType represents the type of speech in the Tibia protocol.
type SpeakType uint8

// SpeakType constants
const (
	SpeakSay         SpeakType = 0x01
	SpeakWhisper     SpeakType = 0x02
	SpeakYell        SpeakType = 0x03
	SpeakPrivateFrom SpeakType = 0x04
	SpeakPrivateTo   SpeakType = 0x05
	SpeakChannelO    SpeakType = 0x06
	SpeakChannelY    SpeakType = 0x07
	SpeakChannelR1   SpeakType = 0x08
	SpeakPrivatePN   SpeakType = 0x09
	SpeakBroadcast   SpeakType = 0x0A
	SpeakChannelR2   SpeakType = 0x0B
	SpeakMonsterSay  SpeakType = 0x0C
	SpeakMonsterYell SpeakType = 0x0D
)
