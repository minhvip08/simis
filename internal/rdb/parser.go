package rdb

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strconv"
	"time"

	errs "github.com/minhvip08/simis/internal/error"
)

// RDBEntry represents a single key/value entry parsed from an RDB file.
// For this stage we only support simple string values with an optional expiry.
type RDBEntry struct {
	Key      string
	Value    string
	ExpireAt *time.Time
}

// rdbReader wraps a byte slice and keeps track of the current offset.
type rdbReader struct {
	data []byte
	pos  int
}

func newRDBReader(b []byte) *rdbReader {
	return &rdbReader{data: b}
}

func (r *rdbReader) readByte() (byte, error) {
	if r.pos >= len(r.data) {
		return 0, errs.ErrUnexpectedEOF
	}
	b := r.data[r.pos]
	r.pos++
	return b, nil
}

func (r *rdbReader) readN(n int) ([]byte, error) {
	if r.pos+n > len(r.data) {
		return nil, errs.ErrUnexpectedEOF
	}
	out := r.data[r.pos : r.pos+n]
	r.pos += n
	return out, nil
}

func (r *rdbReader) readUint32LE() (uint32, error) {
	b, err := r.readN(4)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(b), nil
}

func (r *rdbReader) readUint64LE() (uint64, error) {
	b, err := r.readN(8)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint64(b), nil
}

// readLength decodes a "length-encoded" integer according to the RDB spec.
// It returns (length, isEncoded, encodingType, error).
// When isEncoded is true, encodingType is one of:
//
//	0 -> 8-bit integer, 1 -> 16-bit integer, 2 -> 32-bit integer, 3 -> LZF (unsupported here).
func (r *rdbReader) readLength() (int, bool, byte, error) {
	first, err := r.readByte()
	if err != nil {
		return 0, false, 0, err
	}

	typ := (first & 0xC0) >> 6 // top two bits
	switch typ {
	case 0: // 00, 6-bit length
		return int(first & 0x3F), false, 0, nil
	case 1: // 01, 14-bit length
		next, err := r.readByte()
		if err != nil {
			return 0, false, 0, err
		}
		// remaining 6 bits in first, plus next byte, big-endian
		val := (int(first&0x3F) << 8) | int(next)
		return val, false, 0, nil
	case 2: // 10, 32-bit length (big-endian)
		buf, err := r.readN(4)
		if err != nil {
			return 0, false, 0, err
		}
		val := int(binary.BigEndian.Uint32(buf))
		return val, false, 0, nil
	case 3: // 11, "special" encoding (integers, LZF, etc.)
		encType := first & 0x3F
		return 0, true, encType, nil
	default:
		return 0, false, 0, fmt.Errorf("unknown length encoding type: %d", typ)
	}
}

// readString decodes a "string-encoded" value according to the RDB spec.
// For this challenge we support:
//   - plain strings (length prefix with 00,01,10 modes), and
//   - 8/16/32-bit integer encodings.
//
// LZF-compressed strings (encoding 3) are not supported and will return ErrUnsupportedType.
func (r *rdbReader) readString() (string, error) {
	length, isEncoded, encType, err := r.readLength()
	if err != nil {
		return "", err
	}

	if !isEncoded {
		// regular string: 'length' bytes follow
		buf, err := r.readN(length)
		if err != nil {
			return "", err
		}
		return string(buf), nil
	}

	// Encoded string (integer or LZF)
	switch encType {
	case 0: // 8-bit integer
		b, err := r.readByte()
		if err != nil {
			return "", err
		}
		return strconv.Itoa(int(int8(b))), nil
	case 1: // 16-bit integer (little-endian)
		buf, err := r.readN(2)
		if err != nil {
			return "", err
		}
		val := int16(binary.LittleEndian.Uint16(buf))
		return strconv.Itoa(int(val)), nil
	case 2: // 32-bit integer (little-endian)
		buf, err := r.readN(4)
		if err != nil {
			return "", err
		}
		val := int32(binary.LittleEndian.Uint32(buf))
		return strconv.Itoa(int(val)), nil
	case 3: // LZF-compressed string (not used in this challenge)
		return "", errs.ErrUnsupportedType
	default:
		return "", fmt.Errorf("unknown string encoding type: %d", encType)
	}
}

// ParseRDB parses the given RDB binary data and returns all key/value entries
// for string-typed values found in the first database.
// It intentionally focuses on the subset of the RDB format required for this challenge.
func ParseRDB(data []byte) ([]RDBEntry, error) {
	r := newRDBReader(data)

	// 1. Header: "REDIS" + 4-digit version (e.g. "0011")
	if len(data) < 9 {
		return nil, errs.ErrInvalidRDBHeader
	}
	magic := string(data[:5])
	if magic != "REDIS" {
		return nil, errs.ErrInvalidRDBHeader
	}
	// We don't currently enforce the version number; skip 9 bytes total.
	r.pos = 9

	entries := make([]RDBEntry, 0)
	var currentExpire *time.Time
	var estimatedSize int // Will be set when we encounter 0xFB

	for {
		b, err := r.readByte()
		if err != nil {
			// We expect a clean EOF signaled by 0xFF, not a raw EOF.
			if errors.Is(err, errs.ErrUnexpectedEOF) {
				return nil, err
			}
			return nil, err
		}

		switch b {
		case 0xFA:
			// Metadata subsection: name (string), value (string), both string-encoded.
			if _, err := r.readString(); err != nil {
				return nil, err
			}
			if _, err := r.readString(); err != nil {
				return nil, err
			}

		case 0xFE:
			// Start of a database section. Next is the DB index (length-encoded).
			if _, _, _, err := r.readLength(); err != nil {
				return nil, err
			}

		case 0xFB:
			// Hash table sizes for keys and expires (both length-encoded).
			kvSize, _, _, err := r.readLength() // key-value hash table size
			if err != nil {
				return nil, err
			}
			if _, _, _, err := r.readLength(); err != nil { // expire hash table size
				return nil, err
			}
			// Pre-allocate entries slice if we haven't already and size is reasonable
			if estimatedSize == 0 && kvSize > 0 && kvSize < 1000000 {
				entries = make([]RDBEntry, 0, kvSize)
				estimatedSize = kvSize
			}

		case 0xFC:
			// Expire time in milliseconds (8-byte unsigned long, little-endian).
			ms, err := r.readUint64LE()
			if err != nil {
				return nil, err
			}
			t := time.Unix(0, int64(ms)*int64(time.Millisecond))
			currentExpire = &t

		case 0xFD:
			// Expire time in seconds (4-byte unsigned int, little-endian).
			secs, err := r.readUint32LE()
			if err != nil {
				return nil, err
			}
			t := time.Unix(int64(secs), 0)
			currentExpire = &t

		case 0xFF:
			// End-of-file section: next 8 bytes are CRC64 checksum, which we ignore here.
			// Make sure we can read the checksum bytes without error.
			if _, err := r.readN(8); err != nil {
				return nil, err
			}
			return entries, nil

		default:
			// Otherwise, this byte is the value type flag for a key.
			valueType := b

			// Only handle simple string values (valueType == 0).
			if valueType != 0 {
				return nil, errs.ErrUnsupportedType
			}

			// Key and value are both string-encoded.
			key, err := r.readString()
			if err != nil {
				return nil, err
			}
			val, err := r.readString()
			if err != nil {
				return nil, err
			}

			entry := RDBEntry{
				Key:   key,
				Value: val,
			}
			if currentExpire != nil {
				// Create a copy of the time value, not a pointer reference
				expireCopy := *currentExpire
				entry.ExpireAt = &expireCopy
			}
			entries = append(entries, entry)

			// Reset expire so it only applies to the key that immediately follows
			// the expire opcode, matching the RDB specification.
			currentExpire = nil
		}
	}
}

// LoadRDBIntoStore is a convenience function that parses RDB data and loads
// all parsed entries into the provided KV store.
// This can be used to restore data from an RDB file or snapshot.
func LoadRDBIntoStore(data []byte, store RDBStore) error {
	entries, err := ParseRDB(data)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		store.StoreRDBEntry(entry.Key, entry.Value, entry.ExpireAt)
	}

	return nil
}

// RDBStore is an interface that defines what's needed to load RDB entries.
// This allows the parser to remain decoupled from the actual store implementation.
type RDBStore interface {
	StoreRDBEntry(key, value string, expireAt *time.Time)
}

// KVStoreReader is an interface for reading from the KV store to generate RDB files
type KVStoreReader interface {
	Range(fn func(key string, val interface{}) bool)
}

// rdbWriter helps build RDB binary data
type rdbWriter struct {
	data []byte
}

func newRDBWriter() *rdbWriter {
	return &rdbWriter{data: make([]byte, 0, 1024)}
}

func (w *rdbWriter) writeByte(b byte) {
	w.data = append(w.data, b)
}

func (w *rdbWriter) writeBytes(b []byte) {
	w.data = append(w.data, b...)
}

func (w *rdbWriter) writeUint32LE(val uint32) {
	buf := make([]byte, 4)
	binary.LittleEndian.PutUint32(buf, val)
	w.writeBytes(buf)
}

func (w *rdbWriter) writeUint64LE(val uint64) {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, val)
	w.writeBytes(buf)
}

// writeLength encodes a length value according to RDB spec
func (w *rdbWriter) writeLength(length int) {
	if length < 64 {
		// 00pppppp - length in lower 6 bits
		w.writeByte(byte(length))
	} else if length < 16384 {
		// 01pppppp qqqqqqqq - 14-bit length
		w.writeByte(byte(0x40 | (length >> 8)))
		w.writeByte(byte(length & 0xFF))
	} else {
		// 10000000 followed by 4-byte big-endian length
		w.writeByte(0x80)
		buf := make([]byte, 4)
		binary.BigEndian.PutUint32(buf, uint32(length))
		w.writeBytes(buf)
	}
}

// writeString encodes a string according to RDB spec
func (w *rdbWriter) writeString(s string) {
	w.writeLength(len(s))
	w.writeBytes([]byte(s))
}

// RedisObject interface for type assertion
type RedisObject interface {
	Get() interface{}
	GetTTL() *time.Time
}

// StringValue interface for type checking
type StringValue interface {
	Get() string
}

// GenerateRDB creates an RDB binary dump from a KV store
func GenerateRDB(store KVStoreReader) ([]byte, error) {
	w := newRDBWriter()

	// Magic string "REDIS" followed by 4-byte version (e.g., "0011")
	w.writeBytes([]byte("REDIS0011"))

	// Auxiliary fields (optional metadata) - we'll skip for simplicity

	// Database selector - DB 0
	w.writeByte(0xFE) // SELECTDB opcode
	w.writeByte(0x00) // Database number 0

	// Key-value pairs
	type entry struct {
		key      string
		value    string
		expireAt *time.Time
	}

	entries := make([]entry, 0)

	// Collect all non-expired entries from the store
	now := time.Now()
	store.Range(func(key string, val interface{}) bool {
		// Type assertion to get RedisObject
		redisObj, ok := val.(RedisObject)
		if !ok {
			return true // continue iteration
		}

		// Skip expired keys
		ttl := redisObj.GetTTL()
		if ttl != nil && ttl.Before(now) {
			return true // skip this key
		}

		// Only support string values for now
		rval := redisObj.Get()
		if strVal, ok := rval.(StringValue); ok {
			entries = append(entries, entry{
				key:      key,
				value:    strVal.Get(),
				expireAt: ttl,
			})
		}
		return true // continue iteration
	})

	// Resize DB hint (FB opcode) - helps Redis pre-allocate memory
	if len(entries) > 0 {
		w.writeByte(0xFB)
		w.writeLength(len(entries)) // number of keys
		w.writeLength(0)            // number of keys with expiry (we'll calculate on the fly)
	}

	// Write each key-value pair
	for _, e := range entries {
		// Write expiry if present and not expired
		if e.expireAt != nil && e.expireAt.After(now) {
			// EXPIRETIME_MS (0xFC) - expiry time in milliseconds
			w.writeByte(0xFC)
			expiryMs := uint64(e.expireAt.UnixMilli())
			w.writeUint64LE(expiryMs)
		}

		// Value type - 0x00 for string
		w.writeByte(0x00)

		// Write key and value
		w.writeString(e.key)
		w.writeString(e.value)
	}

	// End of RDB file
	w.writeByte(0xFF) // EOF opcode

	// 8-byte CRC64 checksum (optional - we'll use zeros for simplicity)
	for i := 0; i < 8; i++ {
		w.writeByte(0x00)
	}

	return w.data, nil
}
