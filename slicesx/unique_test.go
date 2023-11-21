package slicesx_test

import (
	"slices"
	"testing"

	"github.com/bloom42/stdx/slicesx"
	"github.com/bloom42/stdx/uuid"
)

func TestUniqueUUIDs(t *testing.T) {
	uuid1 := uuid.New()
	uuid2 := uuid.New()
	uuid3 := uuid.New()
	uuid4 := uuid.New()
	uuid5 := uuid.New()

	input := [][]uuid.UUID{
		{},
		{uuid1},
		{uuid1, uuid1},
		{uuid1, uuid1, uuid2},
		{uuid1, uuid1, uuid2, uuid2, uuid1, uuid3},
		{uuid1, uuid2, uuid3, uuid4, uuid5},
	}
	expected := [][]uuid.UUID{
		{},
		{uuid1},
		{uuid1},
		{uuid1, uuid2},
		{uuid1, uuid2, uuid3},
		{uuid1, uuid2, uuid3, uuid4, uuid5},
	}

	for i := range input {
		output := slicesx.Unique(input[i])
		if !slices.EqualFunc(expected[i], output, func(a, b uuid.UUID) bool {
			return a.String() == b.String()
		}) {
			t.Errorf("(%d) %#v (output) != %#v (expected)", i, output, expected[i])
		}
	}

}
