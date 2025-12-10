package wire

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFoodGroupName_HappyPath(t *testing.T) {
	assert.Equal(t, "OService", FoodGroupName(OService))
}

func TestFoodGroupName_InvalidFoodGroup(t *testing.T) {
	assert.Equal(t, "unknown", FoodGroupName(2142))
}
