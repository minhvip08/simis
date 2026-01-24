package handler

import (
	"errors"
	"strconv"
	"strings"

	"github.com/minhvip08/simis/internal/datastructure"
	"github.com/minhvip08/simis/internal/utils"
)

type StreamKeyEntries struct {
	Key     string
	Entries []*datastructure.Entry
}

var (
	ErrInvalidArguments = errors.New("Invalid arguments")
	ErrTimeout          = errors.New("Timeout")
	ErrUserNotFound     = errors.New("User not found")
)

// Redis data type names
const (
	DataTypeNone   = "none"
	DataTypeString = "string"
	DataTypeList   = "list"
	DataTypeStream = "stream"
	DataTypeSet    = "set"
	DataTypeZSet   = "zset"
	DataTypeHash   = "hash"
)

func ToEmptyArray() string {
	return "*0\r\n"
}

/*
toStreamEntries converts a slice of stream entries to a RESP array.
Each entry is represented as an array of 2 elements:
  - Element 0: The entry ID as a bulk string
  - Element 1: An array of field key-value pairs (flattened)
*/
func toStreamEntries(entries []*datastructure.Entry) string {
	if entries == nil {
		return "*-1\r\n"
	}

	var builder strings.Builder
	builder.Grow(256) // Pre-allocate reasonable size
	builder.WriteString("*")
	builder.WriteString(strconv.Itoa(len(entries)))
	builder.WriteString("\r\n")

	for _, entry := range entries {
		formatSingleEntry(&builder, entry)
	}
	return builder.String()
}

// formatSingleEntry formats a single stream entry as RESP array [ID, [fields...]]
func formatSingleEntry(builder *strings.Builder, entry *datastructure.Entry) {
	// Each entry is an array of 2 elements: [ID, fields_array]
	builder.WriteString("*2\r\n")
	// Element 0: ID as bulk string
	builder.WriteString(utils.ToBulkString(entry.ID.String()))
	// Element 1: Fields as array (key-value pairs flattened)
	fields := make([]string, 0, len(entry.Fields)*2)
	for key, value := range entry.Fields {
		fields = append(fields, key, value)
	}
	builder.WriteString(utils.ToRespArray(fields))
}

/*
toStreamEntriesByKey converts a slice of stream key-entry pairs into a RESP array.
- The order of the input slice is preserved in the output.
- Each element in the top-level array is an array of 2 elements: [key, entries_array].
- The entries_array is formatted the same way as toStreamEntries.
- Format: *N\r\n (*2\r\n$X\r\n<key>\r\n*M\r\n<entries>...) ...
*/
func toStreamEntriesByKey(streams []StreamKeyEntries) string {
	if streams == nil {
		return "*-1\r\n"
	}

	var builder strings.Builder
	builder.Grow(512) // Pre-allocate reasonable size for multiple streams

	// Top-level array with one element per stream key
	builder.WriteString("*")
	builder.WriteString(strconv.Itoa(len(streams)))
	builder.WriteString("\r\n")

	for _, stream := range streams {
		// Each stream is an array of 2 elements: [key, entries_array]
		builder.WriteString("*2\r\n")
		// Element 0: Key as bulk string
		builder.WriteString(utils.ToBulkString(stream.Key))
		// Element 1: Entries array
		if stream.Entries == nil {
			builder.WriteString("*-1\r\n")
		} else {
			builder.WriteString("*")
			builder.WriteString(strconv.Itoa(len(stream.Entries)))
			builder.WriteString("\r\n")
			for _, entry := range stream.Entries {
				formatSingleEntry(&builder, entry)
			}
		}
	}
	return builder.String()
}
