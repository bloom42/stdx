package uuid

import (
	"bytes"
	"testing"
)

func TestNewUlid(t *testing.T) {
	for i := 0; i < 10000; i += 1 {
		_ = NewUlid()
	}
}

func TestParse(t *testing.T) {
	for i := 0; i < 10000; i += 1 {
		id := NewUlid()
		parsed, err := Parse(id.String())
		if err != nil {
			t.Errorf("parsing ulid: %s", err)
		}
		if !bytes.Equal(id[:], parsed[:]) {
			t.Errorf("parsed (%s) != original ULID (%s)", parsed.String(), id.String())
		}
	}
}
