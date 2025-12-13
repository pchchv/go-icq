package wire

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
