package types

import (
	"crypto/rand"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
)

// NewUUIDv7 generates a time-ordered UUIDv7
func NewUUIDv7() pgtype.UUID {
	var uuid [16]byte
	now := time.Now().UTC()
	millis := uint64(now.UnixMilli())

	// Top 48 bits: timestamp in milliseconds
	uuid[0] = byte(millis >> 40)
	uuid[1] = byte(millis >> 32)
	uuid[2] = byte(millis >> 24)
	uuid[3] = byte(millis >> 16)
	uuid[4] = byte(millis >> 8)
	uuid[5] = byte(millis)

	// Next 16 bits: version 7 + 12 bits random
	var randBytes [10]byte
	_, _ = rand.Read(randBytes[:])

	uuid[6] = 0x70 | (randBytes[0] & 0x0f)
	uuid[7] = randBytes[1]

	// Next 16 bits: variant 2 (RFC 4122) + random
	uuid[8] = 0x80 | (randBytes[2] & 0x3f)
	copy(uuid[9:], randBytes[3:10])

	return pgtype.UUID{
		Bytes: uuid,
		Valid: true,
	}
}

// StringToUUID converts a standard UUID string to pgtype.UUID
func StringToUUID(s string) (pgtype.UUID, error) {
	var u pgtype.UUID
	err := u.Scan(s)
	if err != nil {
		return pgtype.UUID{}, fmt.Errorf("invalid UUID %q: %w", s, err)
	}
	return u, nil
}

// UUIDToString converts pgtype.UUID to string
func UUIDToString(u pgtype.UUID) string {
	if !u.Valid {
		return ""
	}
	src := u.Bytes
	return fmt.Sprintf("%x-%x-%x-%x-%x", src[0:4], src[4:6], src[6:8], src[8:10], src[10:16])
}

// UUIDToBytesSlice converts pgtype.UUID to 16 bytes
func UUIDToBytesSlice(u pgtype.UUID) []byte {
	if !u.Valid {
		return nil
	}
	res := make([]byte, 16)
	copy(res, u.Bytes[:])
	return res
}

// UnixMilliFromUUIDv7 extracts the Unix millisecond timestamp from a UUIDv7
func UnixMilliFromUUIDv7(u pgtype.UUID) int64 {
	if !u.Valid {
		return 0
	}
	var millis uint64
	millis = uint64(u.Bytes[0]) << 40
	millis |= uint64(u.Bytes[1]) << 32
	millis |= uint64(u.Bytes[2]) << 24
	millis |= uint64(u.Bytes[3]) << 16
	millis |= uint64(u.Bytes[4]) << 8
	millis |= uint64(u.Bytes[5])
	return int64(millis)
}

// TimeToTimestamptz converts time.Time to pgtype.Timestamptz
func TimeToTimestamptz(t time.Time) pgtype.Timestamptz {
	if t.IsZero() {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{
		Time:  t.UTC(),
		Valid: true,
	}
}

// NullTimeToTimestamptz converts *time.Time to pgtype.Timestamptz
func NullTimeToTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil || t.IsZero() {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{
		Time:  t.UTC(),
		Valid: true,
	}
}

// TimestamptzToNullTime converts pgtype.Timestamptz to *time.Time
func TimestamptzToNullTime(ts pgtype.Timestamptz) *time.Time {
	if !ts.Valid {
		return nil
	}
	t := ts.Time.UTC()
	return &t
}
