package entity_test

import (
	"errors"
	"testing"

	"github.com/MutterPedro/otserver/internal/entity"
)

// TestPositionIsValid verifies that Position.IsValid correctly identifies valid
// and invalid positions based on the Tibia map bounds.
// Position (0,0,0) is the "null" position and considered invalid.
// Valid positions have non-zero X and Y, and Z in range 0–15.
func TestPositionIsValid(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		pos  entity.Position
		want bool
	}{
		{"null position (0,0,0)", entity.Position{X: 0, Y: 0, Z: 0}, false},
		{"valid typical position", entity.Position{X: 100, Y: 100, Z: 7}, true},
		{"valid ground floor", entity.Position{X: 1000, Y: 2000, Z: 0}, true},
		{"valid max Z", entity.Position{X: 100, Y: 100, Z: 15}, true},
		{"invalid Z=16", entity.Position{X: 100, Y: 100, Z: 16}, false},
		{"invalid Z=255", entity.Position{X: 100, Y: 100, Z: 255}, false},
		{"X=0 only", entity.Position{X: 0, Y: 100, Z: 7}, false},
		{"Y=0 only", entity.Position{X: 100, Y: 0, Z: 7}, false},
		{"max uint16 coords", entity.Position{X: 0xFFFF, Y: 0xFFFF, Z: 15}, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := tc.pos.IsValid()
			if got != tc.want {
				t.Errorf("Position{%d,%d,%d}.IsValid() = %v, want %v",
					tc.pos.X, tc.pos.Y, tc.pos.Z, got, tc.want)
			}
		})
	}
}

// TestPositionDistance_SameFloor verifies that Distance returns the Chebyshev
// (chessboard) distance between two positions on the same floor.
func TestPositionDistance_SameFloor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a, b entity.Position
		want int
	}{
		{"same position", entity.Position{X: 100, Y: 100, Z: 7}, entity.Position{X: 100, Y: 100, Z: 7}, 0},
		{"horizontal distance 5", entity.Position{X: 100, Y: 100, Z: 7}, entity.Position{X: 105, Y: 100, Z: 7}, 5},
		{"vertical distance 3", entity.Position{X: 100, Y: 100, Z: 7}, entity.Position{X: 100, Y: 103, Z: 7}, 3},
		{"diagonal Chebyshev", entity.Position{X: 100, Y: 100, Z: 7}, entity.Position{X: 103, Y: 105, Z: 7}, 5},
		{"reversed order", entity.Position{X: 105, Y: 100, Z: 7}, entity.Position{X: 100, Y: 100, Z: 7}, 5},
		{"adjacent", entity.Position{X: 10, Y: 10, Z: 7}, entity.Position{X: 11, Y: 10, Z: 7}, 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := tc.a.Distance(tc.b)
			if err != nil {
				t.Fatalf("Distance: unexpected error: %v", err)
			}
			if got != tc.want {
				t.Errorf("Distance = %d, want %d", got, tc.want)
			}
		})
	}
}

// TestPositionDistance_DifferentFloors verifies that Distance returns an error
// when positions are on different floors.
func TestPositionDistance_DifferentFloors(t *testing.T) {
	t.Parallel()

	a := entity.Position{X: 100, Y: 100, Z: 7}
	b := entity.Position{X: 100, Y: 100, Z: 6}

	_, err := a.Distance(b)
	if err == nil {
		t.Fatal("Distance across floors: expected error, got nil")
	}

	if !errors.Is(err, entity.ErrDifferentFloor) {
		t.Errorf("Distance error = %v, want ErrDifferentFloor", err)
	}
}

// TestPositionIsAdjacentTo verifies adjacency detection including cardinal
// and diagonal neighbors, and non-adjacent positions.
func TestPositionIsAdjacentTo(t *testing.T) {
	t.Parallel()

	center := entity.Position{X: 10, Y: 10, Z: 7}

	tests := []struct {
		name string
		pos  entity.Position
		want bool
	}{
		{"same position", entity.Position{X: 10, Y: 10, Z: 7}, false},
		{"north", entity.Position{X: 10, Y: 9, Z: 7}, true},
		{"south", entity.Position{X: 10, Y: 11, Z: 7}, true},
		{"east", entity.Position{X: 11, Y: 10, Z: 7}, true},
		{"west", entity.Position{X: 9, Y: 10, Z: 7}, true},
		{"northeast diagonal", entity.Position{X: 11, Y: 9, Z: 7}, true},
		{"southwest diagonal", entity.Position{X: 9, Y: 11, Z: 7}, true},
		{"two apart horizontally", entity.Position{X: 12, Y: 10, Z: 7}, false},
		{"two apart vertically", entity.Position{X: 10, Y: 12, Z: 7}, false},
		{"different floor", entity.Position{X: 11, Y: 10, Z: 6}, false},
		{"far away", entity.Position{X: 100, Y: 100, Z: 7}, false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := center.IsAdjacentTo(tc.pos)
			if got != tc.want {
				t.Errorf("Position{10,10,7}.IsAdjacentTo(%v) = %v, want %v",
					tc.pos, got, tc.want)
			}
		})
	}
}

// TestPositionTranslate verifies that Translate returns a new position offset
// by the given deltas without modifying the original.
func TestPositionTranslate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		start    entity.Position
		dx, dy   int
		dz       int8
		expected entity.Position
	}{
		{"positive offset", entity.Position{X: 100, Y: 100, Z: 7}, 1, -1, 0, entity.Position{X: 101, Y: 99, Z: 7}},
		{"zero offset", entity.Position{X: 50, Y: 50, Z: 5}, 0, 0, 0, entity.Position{X: 50, Y: 50, Z: 5}},
		{"negative offset", entity.Position{X: 100, Y: 100, Z: 7}, -10, -20, -2, entity.Position{X: 90, Y: 80, Z: 5}},
		{"floor change up", entity.Position{X: 100, Y: 100, Z: 7}, 0, 0, -1, entity.Position{X: 100, Y: 100, Z: 6}},
		{"floor change down", entity.Position{X: 100, Y: 100, Z: 7}, 0, 0, 1, entity.Position{X: 100, Y: 100, Z: 8}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := tc.start.Translate(tc.dx, tc.dy, tc.dz)
			if got != tc.expected {
				t.Errorf("Translate(%d,%d,%d) = %v, want %v",
					tc.dx, tc.dy, tc.dz, got, tc.expected)
			}
		})
	}
}

// TestPositionTranslate_Overflow verifies that Translate wraps around at
// uint16 boundaries, matching C++ unsigned overflow behavior.
func TestPositionTranslate_Overflow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		start    entity.Position
		dx, dy   int
		dz       int8
		expected entity.Position
	}{
		{"X overflow wraps to 0", entity.Position{X: 0xFFFF, Y: 100, Z: 7}, 1, 0, 0, entity.Position{X: 0, Y: 100, Z: 7}},
		{"Y overflow wraps to 0", entity.Position{X: 100, Y: 0xFFFF, Z: 7}, 0, 1, 0, entity.Position{X: 100, Y: 0, Z: 7}},
		{"X underflow wraps to max", entity.Position{X: 0, Y: 100, Z: 7}, -1, 0, 0, entity.Position{X: 0xFFFF, Y: 100, Z: 7}},
		{"Y underflow wraps to max", entity.Position{X: 100, Y: 0, Z: 7}, 0, -1, 0, entity.Position{X: 100, Y: 0xFFFF, Z: 7}},
		{"Z overflow wraps", entity.Position{X: 100, Y: 100, Z: 255}, 0, 0, 1, entity.Position{X: 100, Y: 100, Z: 0}},
		{"Z underflow wraps", entity.Position{X: 100, Y: 100, Z: 0}, 0, 0, -1, entity.Position{X: 100, Y: 100, Z: 255}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := tc.start.Translate(tc.dx, tc.dy, tc.dz)
			if got != tc.expected {
				t.Errorf("Translate(%d,%d,%d) = %v, want %v",
					tc.dx, tc.dy, tc.dz, got, tc.expected)
			}
		})
	}

	// Verify Translate does not modify the original position (value semantics).
	original := entity.Position{X: 100, Y: 200, Z: 7}
	_ = original.Translate(50, 50, 1)
	if original.X != 100 || original.Y != 200 || original.Z != 7 {
		t.Error("Translate modified the original position (should return new value)")
	}
}

// ---------------------------------------------------------------------------
// Direction
// ---------------------------------------------------------------------------

// TestDirectionOpposite_Cardinal verifies opposite directions for the 4
// cardinal directions.
func TestDirectionOpposite_Cardinal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		d    entity.Direction
		want entity.Direction
	}{
		{"North → South", entity.DirectionNorth, entity.DirectionSouth},
		{"South → North", entity.DirectionSouth, entity.DirectionNorth},
		{"East → West", entity.DirectionEast, entity.DirectionWest},
		{"West → East", entity.DirectionWest, entity.DirectionEast},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := tc.d.Opposite()
			if got != tc.want {
				t.Errorf("Direction(%d).Opposite() = %d, want %d", tc.d, got, tc.want)
			}
		})
	}
}

// TestDirectionOpposite_Diagonal verifies opposite directions for the 4
// diagonal directions.
func TestDirectionOpposite_Diagonal(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		d    entity.Direction
		want entity.Direction
	}{
		{"SouthWest → NorthEast", entity.DirectionSouthWest, entity.DirectionNorthEast},
		{"NorthEast → SouthWest", entity.DirectionNorthEast, entity.DirectionSouthWest},
		{"SouthEast → NorthWest", entity.DirectionSouthEast, entity.DirectionNorthWest},
		{"NorthWest → SouthEast", entity.DirectionNorthWest, entity.DirectionSouthEast},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := tc.d.Opposite()
			if got != tc.want {
				t.Errorf("Direction(%d).Opposite() = %d, want %d", tc.d, got, tc.want)
			}
		})
	}
}

// TestDirectionIsValid_AllEight verifies that all 8 compass directions (0–7)
// are valid.
func TestDirectionIsValid_AllEight(t *testing.T) {
	t.Parallel()

	for i := entity.Direction(0); i <= 7; i++ {
		d := i
		t.Run("direction_"+string(rune('0'+d)), func(t *testing.T) {
			t.Parallel()

			if !d.IsValid() {
				t.Errorf("Direction(%d).IsValid() = false, want true", d)
			}
		})
	}
}

// TestDirectionIsValid_OutOfRange verifies that values 8+ are invalid.
func TestDirectionIsValid_OutOfRange(t *testing.T) {
	t.Parallel()

	tests := []entity.Direction{8, 9, 100, 255}
	for _, d := range tests {
		d := d
		t.Run("direction_"+string(rune('0'+d)), func(t *testing.T) {
			t.Parallel()

			if d.IsValid() {
				t.Errorf("Direction(%d).IsValid() = true, want false", d)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ReturnValue
// ---------------------------------------------------------------------------

// TestReturnValueZeroIsNoError asserts that the zero value of ReturnValue
// is RetNoError, which is critical for default initialization semantics.
func TestReturnValueZeroIsNoError(t *testing.T) {
	t.Parallel()

	if entity.ReturnValue(0) != entity.RetNoError {
		t.Errorf("ReturnValue(0) = %d, want RetNoError(%d)", entity.ReturnValue(0), entity.RetNoError)
	}
}

// TestReturnValueConstants verifies that key ReturnValue constants match
// the exact numeric values from C++ enums.h. Wire protocol compatibility
// depends on these values being correct.
func TestReturnValueConstants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		rv   entity.ReturnValue
		want uint8
	}{
		{"RetNoError", entity.RetNoError, 0},
		{"RetNotPossible", entity.RetNotPossible, 1},
		{"RetNotEnoughRoom", entity.RetNotEnoughRoom, 2},
		{"RetCannotThrow", entity.RetCannotThrow, 5},
		{"RetThereIsNoWay", entity.RetThereIsNoWay, 6},
		{"RetTooFarAway", entity.RetTooFarAway, 17},
		{"RetNotEnoughCapacity", entity.RetNotEnoughCapacity, 21},
		{"RetYouNeedPremiumAccount", entity.RetYouNeedPremiumAccount, 48},
		{"RetCanOnlyUseOneShield", entity.RetCanOnlyUseOneShield, 60},
		{"RetNoPartyMembers", entity.RetNoPartyMembers, 61},
		{"RetNotInvited", entity.RetNotInvited, 62},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if uint8(tc.rv) != tc.want {
				t.Errorf("%s = %d, want %d", tc.name, uint8(tc.rv), tc.want)
			}
		})
	}
}

// TestReturnValueString_KnownValues verifies that the String() method returns
// non-empty, meaningful strings for known ReturnValues.
func TestReturnValueString_KnownValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		rv   entity.ReturnValue
	}{
		{"RetNoError", entity.RetNoError},
		{"RetNotPossible", entity.RetNotPossible},
		{"RetTooFarAway", entity.RetTooFarAway},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			s := tc.rv.String()
			if s == "" {
				t.Errorf("%s.String() returned empty string", tc.name)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Outfit
// ---------------------------------------------------------------------------

// TestOutfitZeroValue verifies that the zero value of Outfit is usable
// (represents an invisible/default creature appearance).
func TestOutfitZeroValue(t *testing.T) {
	t.Parallel()

	var o entity.Outfit
	if o.LookType != 0 {
		t.Errorf("zero Outfit.LookType = %d, want 0", o.LookType)
	}
	if o.LookHead != 0 || o.LookBody != 0 || o.LookLegs != 0 || o.LookFeet != 0 {
		t.Error("zero Outfit color parts are not zero")
	}
	if o.LookAddons != 0 {
		t.Errorf("zero Outfit.LookAddons = %d, want 0", o.LookAddons)
	}
	if o.LookMount != 0 {
		t.Errorf("zero Outfit.LookMount = %d, want 0", o.LookMount)
	}
}

// ---------------------------------------------------------------------------
// StackPos
// ---------------------------------------------------------------------------

// TestStackPosAny verifies the special sentinel value.
func TestStackPosAny(t *testing.T) {
	t.Parallel()

	if entity.StackPosAny != 255 {
		t.Errorf("StackPosAny = %d, want 255", entity.StackPosAny)
	}
}

// ---------------------------------------------------------------------------
// Slot
// ---------------------------------------------------------------------------

// TestSlotConstants verifies that inventory slot constants match the expected
// C++ enum values (critical for protocol encoding).
func TestSlotConstants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		slot entity.Slot
		want uint8
	}{
		{"SlotFirst", entity.SlotFirst, 0},
		{"SlotHead", entity.SlotHead, 1},
		{"SlotNeck", entity.SlotNeck, 2},
		{"SlotBack", entity.SlotBack, 3},
		{"SlotBody", entity.SlotBody, 4},
		{"SlotRight", entity.SlotRight, 5},
		{"SlotLeft", entity.SlotLeft, 6},
		{"SlotLegs", entity.SlotLegs, 7},
		{"SlotFeet", entity.SlotFeet, 8},
		{"SlotRing", entity.SlotRing, 9},
		{"SlotAmmo", entity.SlotAmmo, 10},
		{"SlotLast", entity.SlotLast, 11},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if uint8(tc.slot) != tc.want {
				t.Errorf("%s = %d, want %d", tc.name, uint8(tc.slot), tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// SpeakType
// ---------------------------------------------------------------------------

// TestSpeakTypeConstants verifies that speak type constants match the exact
// hex values from the Tibia protocol.
func TestSpeakTypeConstants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		st   entity.SpeakType
		want uint8
	}{
		{"SpeakSay", entity.SpeakSay, 0x01},
		{"SpeakWhisper", entity.SpeakWhisper, 0x02},
		{"SpeakYell", entity.SpeakYell, 0x03},
		{"SpeakPrivateFrom", entity.SpeakPrivateFrom, 0x04},
		{"SpeakPrivateTo", entity.SpeakPrivateTo, 0x05},
		{"SpeakChannelO", entity.SpeakChannelO, 0x06},
		{"SpeakChannelY", entity.SpeakChannelY, 0x07},
		{"SpeakChannelR1", entity.SpeakChannelR1, 0x08},
		{"SpeakPrivatePN", entity.SpeakPrivatePN, 0x09},
		{"SpeakBroadcast", entity.SpeakBroadcast, 0x0A},
		{"SpeakChannelR2", entity.SpeakChannelR2, 0x0B},
		{"SpeakMonsterSay", entity.SpeakMonsterSay, 0x0C},
		{"SpeakMonsterYell", entity.SpeakMonsterYell, 0x0D},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if uint8(tc.st) != tc.want {
				t.Errorf("%s = 0x%02X, want 0x%02X", tc.name, uint8(tc.st), tc.want)
			}
		})
	}
}
