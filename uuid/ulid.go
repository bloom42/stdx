package uuid

import (
	"crypto/rand"
	"time"
)

func NewUlid() UUID {
	ulid, err := NewUlidErr()
	if err != nil {
		panic(err)
	}
	return ulid
}

func NewUlidErr() (UUID, error) {
	var ulid UUID

	err := ulidSetRandom(&ulid)
	if err != nil {
		return Nil, err
	}

	ulidSetTime(&ulid, time.Now())
	return ulid, nil
}

func ulidSetTime(ulid *UUID, t time.Time) {
	timestamp := uint64(t.UnixNano() / int64(time.Millisecond))
	(*ulid)[0] = byte(timestamp >> 40)
	(*ulid)[1] = byte(timestamp >> 32)
	(*ulid)[2] = byte(timestamp >> 24)
	(*ulid)[3] = byte(timestamp >> 16)
	(*ulid)[4] = byte(timestamp >> 8)
	(*ulid)[5] = byte(timestamp)
	// var x, y byte
	// timestamp := uint64(t.UnixNano() / int64(time.Millisecond))
	// // Backups [6] and [7] bytes to override them with their original values later.
	// x, y, ulid[6], ulid[7] = ulid[6], ulid[7], x, y
	// binary.LittleEndian.PutUint64(ulid[:], timestamp)
	// // Truncates at the 6th byte as designed in the original spec (48 bytes).
	// ulid[6], ulid[7] = x, y
}

func ulidSetRandom(ulid *UUID) (err error) {
	_, err = rand.Read(ulid[6:])
	return
}
