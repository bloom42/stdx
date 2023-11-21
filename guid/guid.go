package guid

import (
	"crypto/rand"
	"database/sql/driver"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/bloom42/stdx/base32"
	"github.com/bloom42/stdx/crypto"
)

const (
	Size = 16

	randPoolSize = 128 * 16
)

// A GUID is a 128 bit (16 byte) Globally Unique IDentifier
type GUID [Size]byte

var (
	ErrGuidIsNotValid = errors.New("GUID is not valid")
	ErrUuidIsNotValid = errors.New("Not a valid UUID")
)

var (
	Empty GUID // empty GUID, all zeros

	poolEnabled  = true
	randomSource = rand.Reader // random function
	poolMutex    sync.Mutex
	poolPosition = randPoolSize     // protected with poolMutex
	pool         [randPoolSize]byte // protected with poolMutex
)

func NewRandom() GUID {
	guid, err := NewRandomWithErr()
	if err != nil {
		panic(err)
	}
	return guid
}

func NewRandomWithErr() (GUID, error) {
	if poolEnabled {
		return newRandomFromPool()
	} else {
		return newRandomFromReader(randomSource)
	}
}

func newRandomFromPool() (GUID, error) {
	var guid GUID
	poolMutex.Lock()
	if poolPosition == randPoolSize {
		_, err := io.ReadFull(randomSource, pool[:])
		if err != nil {
			poolMutex.Unlock()
			return Empty, err
		}
		poolPosition = 0
	}
	copy(guid[:], pool[poolPosition:(poolPosition+16)])
	poolPosition += 16
	poolMutex.Unlock()

	guid[6] = (guid[6] & 0x0f) | (0x04 << 4) // Version 4
	guid[8] = (guid[8] & 0x3f) | 0x80        // Variant is 10

	return guid, nil
}

func newRandomFromReader(reader io.Reader) (GUID, error) {
	var guid GUID
	_, err := io.ReadFull(reader, guid[:])
	if err != nil {
		return Empty, err
	}

	guid[6] = (guid[6] & 0x0f) | 0x40 // Version 4
	guid[8] = (guid[8] & 0x3f) | 0x80 // Variant is 10
	return guid, nil
}

func NewTimeBased() GUID {
	guid, err := NewTimeBasedWithErr()
	if err != nil {
		panic(err)
	}
	return guid
}

// TODO: Improve by reading only 10 bytes of random data
func NewTimeBasedWithErr() (guid GUID, err error) {
	guid, err = NewRandomWithErr()
	if err != nil {
		return
	}

	now := time.Now().UTC()
	// 48 bit timestamp
	timestamp := uint64(now.UnixNano() / int64(time.Millisecond))
	guid[0] = byte(timestamp >> 40)
	guid[1] = byte(timestamp >> 32)
	guid[2] = byte(timestamp >> 24)
	guid[3] = byte(timestamp >> 16)
	guid[4] = byte(timestamp >> 8)
	guid[5] = byte(timestamp)

	// var timestamp uint64
	// timestamp += uint64(now.Unix()) * 1e3
	// timestamp += uint64(now.Nanosecond()) / 1e6
	// binary.BigEndian.PutUint64(guid[:8], timestamp<<16)

	// var x, y byte
	// timestamp := uint64(t.UnixNano() / int64(time.Millisecond))
	// // Backups [6] and [7] bytes to override them with their original values later.
	// x, y, ulid[6], ulid[7] = ulid[6], ulid[7], x, y
	// binary.LittleEndian.PutUint64(ulid[:], timestamp)
	// // Truncates at the 6th byte as designed in the original spec (48 bytes).
	// ulid[6], ulid[7] = x, y

	// guid[6] = (guid[6] & 0x0f) | 0x07 // Version 7
	guid[6] = (guid[6] & 0x0f) | (0x07 << 4)
	guid[8] = (guid[8] & 0x3f) | 0x80 // Variant is 10

	return
}

// TODO: parse without allocs
func Parse(input string) (guid GUID, err error) {
	bytes, err := base32.DecodeString(input)
	if err != nil {
		err = ErrGuidIsNotValid
		return
	}

	if len(bytes) != Size {
		err = ErrGuidIsNotValid
		return
	}

	return GUID(bytes), nil
}

// FromBytes creates a new GUID from a byte slice. Returns an error if the slice
// does not have a length of 16. The bytes are copied from the slice.
func FromBytes(b []byte) (guid GUID, err error) {
	err = guid.UnmarshalBinary(b)
	return guid, err
}

// String returns the string form of guid
// TODO: encode without alloc
func (guid GUID) String() string {
	return base32.EncodeToString(guid[:])
}

func (guid GUID) Equal(other GUID) bool {
	return crypto.ConstantTimeCompare(guid[:], other[:])
}

func (guid GUID) Bytes() []byte {
	return guid[:]
}

// MarshalText implements encoding.TextMarshaler.
func (guid GUID) MarshalText() ([]byte, error) {
	ret := guid.String()
	return []byte(ret), nil
}

// UnmarshalText implements encoding.TextUnmarshaler.
func (guid *GUID) UnmarshalText(data []byte) error {
	id, err := Parse(string(data))
	if err != nil {
		return err
	}
	*guid = id
	return nil
}

// MarshalBinary implements encoding.BinaryMarshaler.
func (guid GUID) MarshalBinary() ([]byte, error) {
	return guid[:], nil
}

// UnmarshalBinary implements encoding.BinaryUnmarshaler.
func (guid *GUID) UnmarshalBinary(data []byte) error {
	if len(data) != 16 {
		return fmt.Errorf("invalid GUID (got %d bytes)", len(data))
	}
	copy(guid[:], data)
	return nil
}

// Scan implements sql.Scanner so GUIDs can be read from databases transparently.
// Currently, database types that map to string and []byte are supported. Please
// consult database-specific driver documentation for matching types.
func (guid *GUID) Scan(src interface{}) error {
	switch src := src.(type) {
	case nil:
		return nil

	case string:
		// if an empty GUID comes from a table, we return a null GUID
		if src == "" {
			return nil
		}

		// see Parse for required string format
		u, err := ParseUuidString(src)
		if err != nil {
			return fmt.Errorf("Scan: %v", err)
		}

		*guid = u

	case []byte:
		// if an empty GUID comes from a table, we return a null GUID
		if len(src) == 0 {
			return nil
		}

		// assumes a simple slice of bytes if 16 bytes
		// otherwise attempts to parse
		if len(src) != 16 {
			return guid.Scan(string(src))
		}
		copy((*guid)[:], src)

	default:
		return fmt.Errorf("Scan: unable to scan type %T into GUID", src)
	}

	return nil
}

// Value implements sql.Valuer so that GUIDs can be written to databases
// transparently. Currently, GUIDs map to strings. Please consult
// database-specific driver documentation for matching types.
func (guid GUID) Value() (driver.Value, error) {
	return guid.ToUuidString(), nil
}

// SetRand sets the random number generator to r, which implements io.Reader.
// If r.Read returns an error when the package requests random data then
// a panic will be issued.
//
// Calling SetRand with nil sets the random number generator to the default
// generator.
func SetRand(r io.Reader) {
	if r == nil {
		randomSource = rand.Reader
		return
	}
	randomSource = r
}

func SetPoolEnabled(enabled bool) {
	poolEnabled = enabled
}
