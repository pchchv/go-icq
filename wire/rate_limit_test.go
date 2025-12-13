package wire

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewRateLimitClasses(t *testing.T) {
	input := [5]RateClass{
		{ID: 1, WindowSize: 10},
		{ID: 2, WindowSize: 20},
		{ID: 3, WindowSize: 30},
		{ID: 4, WindowSize: 40},
		{ID: 5, WindowSize: 50},
	}

	classes := NewRateLimitClasses(input)

	assert.Equal(t, input, classes.All())
}

func TestRateLimitClasses_Get(t *testing.T) {
	classes := DefaultRateLimitClasses()

	// Test Get() returns correct class for each ID
	for i := 1; i <= 5; i++ {
		id := RateLimitClassID(i)
		class := classes.Get(id)
		assert.Equal(t, id, class.ID)
		assert.Equal(t, classes.All()[i-1], class)
	}
}

func TestRateLimitClasses_All(t *testing.T) {
	classes := DefaultRateLimitClasses()

	// Test All() returns exactly 5 classes with correct IDs
	all := classes.All()
	assert.Len(t, all, 5)
	for i, class := range all {
		expectedID := RateLimitClassID(i + 1)
		assert.Equal(t, expectedID, class.ID, "class ID mismatch at index %d", i)
	}
}

func TestSNACRateLimits_RateClassLookup(t *testing.T) {
	limits := DefaultSNACRateLimits()

	testCases := []struct {
		foodGroup uint16
		subGroup  uint16
		expected  RateLimitClassID
		found     bool
	}{
		{Chat, ChatUsersJoined, 1, true},
		{Chat, ChatChannelMsgToHost, 2, true},
		{0xFFFF, 0x0001, 0, false},
		{Chat, 0xFFFF, 0, false},
	}

	for _, tc := range testCases {
		classID, ok := limits.RateClassLookup(tc.foodGroup, tc.subGroup)
		assert.Equal(t, tc.found, ok)
		assert.Equal(t, tc.expected, classID)
	}
}

func TestSNACRateLimits_All(t *testing.T) {
	limits := DefaultSNACRateLimits()

	seen := map[uint16]map[uint16]RateLimitClassID{}
	for entry := range limits.All() {
		if _, ok := seen[entry.FoodGroup]; !ok {
			seen[entry.FoodGroup] = map[uint16]RateLimitClassID{}
		}
		seen[entry.FoodGroup][entry.SubGroup] = entry.RateLimitClass
	}

	// Spot-check a few values
	require.Contains(t, seen, ICBM)
	assert.Equal(t, RateLimitClassID(3), seen[ICBM][ICBMChannelMsgToHost])
	assert.Equal(t, RateLimitClassID(1), seen[ICBM][ICBMChannelMsgToClient])

	require.Contains(t, seen, Locate)
	assert.Equal(t, RateLimitClassID(4), seen[Locate][LocateSetDirInfo])
	assert.Equal(t, RateLimitClassID(3), seen[Locate][LocateUserInfoQuery])
}

func TestSNACRateLimits_All_YieldStopsEarly(t *testing.T) {
	limits := DefaultSNACRateLimits()

	count := 0
	limits.All()(func(entry struct {
		FoodGroup      uint16
		SubGroup       uint16
		RateLimitClass RateLimitClassID
	}) bool {
		count++
		// stop iteration after first item to trigger `if !yield(...) { return }`
		return false
	})

	// Should only yield one entry
	assert.Equal(t, 1, count)
}
