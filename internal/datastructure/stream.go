package datastructure

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"time"

	radix "github.com/hashicorp/go-immutable-radix/v2"
)

type Entry struct {
	ID     StreamID
	Fields map[string]string
}

// PendingEntry represents an entry that has been delivered to a consumer but not yet acknowledged
type PendingEntry struct {
	ID             StreamID
	Consumer       string
	DeliveredTime  int64 // Unix milliseconds when delivered
	DeliveryCount  int   // Number of times this entry has been delivered
}

// ConsumerGroup represents a consumer group within a stream
type ConsumerGroup struct {
	Name             string
	LastDeliveredID  StreamID            // Last entry ID delivered to consumers in this group
	Consumers        map[string]*Consumer // Map of consumer name to consumer info
	PendingEntries   map[string]*PendingEntry // Map of entry ID (as string) to pending entry
}

// Consumer represents a consumer within a consumer group
type Consumer struct {
	Name              string
	LastSeenTime      int64 // Unix milliseconds when last seen
	PendingEntriesNum int   // Number of pending entries for this consumer
}

type Stream struct {
	Index           *radix.Tree[int]
	Entries         *ListPack
	lastStreamID    *StreamID
	ConsumerGroups  map[string]*ConsumerGroup // Map of group name to consumer group
}

func (e *Entry) String() string {
	return fmt.Sprintf("ID: %s, Fields: %v", e.ID.String(), e.Fields)
}

func (s *Stream) String() string {
	entries := make([]string, 0, s.Entries.Len())
	for i := 0; i < s.Entries.Len(); i++ {
		entryBytes, err := s.Entries.Get(i)
		if err != nil {
			return fmt.Sprintf("Error getting entry: %v", err)
		}
		entry, err := DecodeEntry(entryBytes)
		if err != nil {
			return fmt.Sprintf("Error decoding entry: %v", err)
		}
		entries = append(entries, entry.String())
	}
	return fmt.Sprintf("lastStreamID: %s, Entries: %v", s.lastStreamID, entries)
}

func NewStream() *Stream {
	return &Stream{
		Entries:        NewListPack(),
		Index:          radix.New[int](),
		ConsumerGroups: make(map[string]*ConsumerGroup),
	}
}

// GetLastStreamID returns the last stream ID, or nil if the stream is empty.
// Returns a copy of the StreamID to avoid exposing internal state.
func (s *Stream) GetLastStreamID() *StreamID {
	return s.lastStreamID
}

func (s *Stream) Kind() RType {
	return RStream
}

func AsStream(v RValue) (*Stream, bool) {
	if sv, ok := v.(*Stream); ok {
		return sv, true
	}
	return nil, false
}

func (s *Stream) CreateEntry(id string, values ...string) (*Entry, error) {
	lastID := s.lastStreamID

	entryID, err := GenerateNextID(lastID, id)
	if err != nil {
		return nil, err
	}

	fields := make(map[string]string)
	for i := 0; i < len(values); i += 2 {
		fields[values[i]] = values[i+1]
	}

	entry := Entry{ID: entryID, Fields: fields}

	return &entry, nil
}

func (s *Stream) AppendEntry(entry *Entry) error {
	// Validate that the entry ID is greater than the last ID
	if s.lastStreamID != nil {
		if entry.ID.Compare(*s.lastStreamID) <= 0 {
			return ErrNotGreaterThanTop
		}
	}

	s.Entries.Append(EncodeEntry(*entry))
	id := entry.ID
	s.lastStreamID = &id
	s.Index, _, _ = s.Index.Insert(entry.ID.EncodeToBytes(), s.Entries.Len()-1)
	return nil
}

// RangeScan returns entries in the range [start, end] (inclusive).
func (s *Stream) RangeScan(start, end StreamID) ([]*Entry, error) {
	fmt.Printf("range scanning stream from %s to %s\n", start.String(), end.String())
	// Check if stream is empty
	if s.lastStreamID == nil {
		return []*Entry{}, nil
	}

	// Check if end is the maximum ID (represents end of stream)
	isMaxEnd := end.Ms == math.MaxUint64 && end.Seq == math.MaxUint64

	startIter := s.Index.Root().Iterator()
	startIter.SeekLowerBound(start.EncodeToBytes())

	_, startIndex, ok := startIter.Next()
	if !ok {
		return []*Entry{}, nil
	}
	fmt.Printf("start index: %d\n", startIndex)

	var endIndex int
	if isMaxEnd {
		// For maximum end ID, look up the last stream ID in the index
		endIter := s.Index.Root().Iterator()
		endIter.SeekLowerBound(s.lastStreamID.EncodeToBytes())
		_, idx, ok := endIter.Next()
		if !ok {
			// Fallback: use last entry index (should not happen if lastStreamID is consistent)
			endIndex = s.Entries.Len() - 1
		} else {
			endIndex = idx
		}
	} else {
		// For normal end ID, seek to it
		endIter := s.Index.Root().Iterator()
		endIter.SeekLowerBound(end.EncodeToBytes())
		_, idx, ok := endIter.Next()
		if !ok {
			return nil, errors.New("end not found")
		}
		endIndex = idx
	}

	fmt.Printf("end index: %d\n", endIndex)

	entries := make([]*Entry, 0, endIndex-startIndex+1)
	for startIndex <= endIndex {
		entryBytes, err := s.Entries.Get(startIndex)
		if err != nil {
			return nil, err
		}
		entry, err := DecodeEntry(entryBytes)
		if err != nil {
			return nil, err
		}
		entries = append(entries, &entry)
		startIndex++
	}
	return entries, nil
}

func appendVarString(dst []byte, s string) []byte {
	tmp := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(tmp, uint64(len(s)))
	dst = append(dst, tmp[:n]...)
	dst = append(dst, s...)
	return dst
}

func EncodeEntry(e Entry) []byte {
	var buf []byte
	// encode ID
	buf = append(buf, e.ID.EncodeToBytes()...)
	// encode field count
	count := uint64(len(e.Fields))
	tmp := make([]byte, binary.MaxVarintLen64)
	n := binary.PutUvarint(tmp, count)
	buf = append(buf, tmp[:n]...)
	// encode each field
	for k, v := range e.Fields {
		buf = appendVarString(buf, k)
		buf = appendVarString(buf, v)
	}
	return buf
}

func DecodeEntry(b []byte) (Entry, error) {
	var e Entry
	pos := 0

	// ID - encoded as 16 fixed bytes (8 bytes Ms + 8 bytes Seq)
	if len(b) < 16 {
		return e, errors.New("invalid entry: too short for ID")
	}
	var err error
	e.ID, err = DecodeFromBytes(b[pos : pos+16])
	if err != nil {
		return e, err
	}
	pos += 16

	// numFields
	count, n2 := binary.Uvarint(b[pos:])
	if n2 <= 0 {
		return e, errors.New("invalid field count")
	}
	pos += n2

	e.Fields = make(map[string]string, count)
	for i := uint64(0); i < count; i++ {
		k, nk, err := readVarString(b, pos)
		if err != nil {
			return e, err
		}
		pos += nk
		v, nv, err := readVarString(b, pos)
		if err != nil {
			return e, err
		}
		pos += nv
		e.Fields[k] = v
	}
	return e, nil
}

func readVarString(b []byte, pos int) (string, int, error) {
	length, n := binary.Uvarint(b[pos:])
	if n <= 0 {
		return "", 0, errors.New("invalid length varint")
	}
	start := pos + n
	end := start + int(length)
	if end > len(b) {
		return "", 0, errors.New("truncated data")
	}
	return string(b[start:end]), n + int(length), nil
}

// Consumer Group Methods

// CreateGroup creates a new consumer group for the stream.
// Returns an error if the group already exists.
func (s *Stream) CreateGroup(groupName string, startID string) error {
	if _, exists := s.ConsumerGroups[groupName]; exists {
		return errors.New("BUSYGROUP Consumer group '" + groupName + "' already exists")
	}

	var lastDeliveredID StreamID
	if startID == "$" {
		// $ means start from the end of the stream
		if s.lastStreamID != nil {
			lastDeliveredID = *s.lastStreamID
		} else {
			lastDeliveredID = StreamID{Ms: 0, Seq: 0}
		}
	} else if startID == "0" || startID == "0-0" {
		// 0 means start from the beginning
		lastDeliveredID = StreamID{Ms: 0, Seq: 0}
	} else {
		// Parse the provided ID
		id, err := ParseStreamID(startID)
		if err != nil {
			return err
		}
		lastDeliveredID = id
	}

	s.ConsumerGroups[groupName] = &ConsumerGroup{
		Name:            groupName,
		LastDeliveredID: lastDeliveredID,
		Consumers:       make(map[string]*Consumer),
		PendingEntries:  make(map[string]*PendingEntry),
	}
	return nil
}

// GetGroup returns a consumer group by name.
func (s *Stream) GetGroup(groupName string) (*ConsumerGroup, error) {
	group, exists := s.ConsumerGroups[groupName]
	if !exists {
		return nil, errors.New("NOGROUP No such consumer group '" + groupName + "' for stream '" + "' |" + "'")
	}
	return group, nil
}

// DestroyGroup removes a consumer group.
func (s *Stream) DestroyGroup(groupName string) error {
	if _, exists := s.ConsumerGroups[groupName]; !exists {
		return errors.New("NOGROUP No such consumer group '" + groupName + "'")
	}
	delete(s.ConsumerGroups, groupName)
	return nil
}

// SetGroupID sets the last delivered ID for a group.
func (s *Stream) SetGroupID(groupName, newID string) error {
	group, err := s.GetGroup(groupName)
	if err != nil {
		return err
	}

	if newID == "$" {
		// $ means set to the end of the stream
		if s.lastStreamID != nil {
			group.LastDeliveredID = *s.lastStreamID
		} else {
			group.LastDeliveredID = StreamID{Ms: 0, Seq: 0}
		}
	} else {
		id, err := ParseStreamID(newID)
		if err != nil {
			return err
		}
		group.LastDeliveredID = id
	}
	return nil
}

// DeleteConsumer removes a consumer from a group and moves its pending entries back to the group queue.
func (s *Stream) DeleteConsumer(groupName, consumerName string) (int, error) {
	group, err := s.GetGroup(groupName)
	if err != nil {
		return 0, err
	}

	_, exists := group.Consumers[consumerName]
	if !exists {
		return 0, nil
	}

	// Count pending entries for this consumer
	pendingCount := 0
	for _, pending := range group.PendingEntries {
		if pending.Consumer == consumerName {
			pendingCount++
			// Reset consumer to empty (entry stays in pending)
			pending.Consumer = ""
		}
	}

	delete(group.Consumers, consumerName)
	return pendingCount, nil
}

// ReadGroupNewEntries reads new entries for a consumer group that haven't been delivered yet.
// Returns entries that come after the group's LastDeliveredID.
func (s *Stream) ReadGroupNewEntries(groupName, consumerName string, count int) ([]*Entry, error) {
	group, err := s.GetGroup(groupName)
	if err != nil {
		return nil, err
	}

	// Update consumer last seen time
	if _, exists := group.Consumers[consumerName]; !exists {
		group.Consumers[consumerName] = &Consumer{
			Name:              consumerName,
			LastSeenTime:      time.Now().UnixMilli(),
			PendingEntriesNum: 0,
		}
	} else {
		group.Consumers[consumerName].LastSeenTime = time.Now().UnixMilli()
	}

	// Scan entries after LastDeliveredID
	startID := group.LastDeliveredID
	startID.Seq++ // Get entries strictly after this ID

	endID := StreamID{Ms: math.MaxUint64, Seq: math.MaxUint64}

	entries, err := s.RangeScan(startID, endID)
	if err != nil {
		return nil, err
	}

	// Limit the number of entries
	if count > 0 && len(entries) > count {
		entries = entries[:count]
	}

	// Add delivered entries to pending list
	now := time.Now().UnixMilli()
	for _, entry := range entries {
		idStr := entry.ID.String()
		group.PendingEntries[idStr] = &PendingEntry{
			ID:            entry.ID,
			Consumer:      consumerName,
			DeliveredTime: now,
			DeliveryCount: 1,
		}
		group.Consumers[consumerName].PendingEntriesNum++
		// Update group's last delivered ID
		if entry.ID.Compare(group.LastDeliveredID) > 0 {
			group.LastDeliveredID = entry.ID
		}
	}

	return entries, nil
}

// AckEntries acknowledges entries for a consumer group, removing them from the pending list.
func (s *Stream) AckEntries(groupName string, entryIDs []string) (int, error) {
	group, err := s.GetGroup(groupName)
	if err != nil {
		return 0, err
	}

	ackedCount := 0
	for _, idStr := range entryIDs {
		if pending, exists := group.PendingEntries[idStr]; exists {
			if pending.Consumer != "" && group.Consumers[pending.Consumer] != nil {
				group.Consumers[pending.Consumer].PendingEntriesNum--
			}
			delete(group.PendingEntries, idStr)
			ackedCount++
		}
	}
	return ackedCount, nil
}

// GetPendingEntries returns pending entries for a group or consumer.
// If consumerName is empty, returns all pending entries for the group.
func (s *Stream) GetPendingEntries(groupName, consumerName string) ([]*PendingEntry, error) {
	group, err := s.GetGroup(groupName)
	if err != nil {
		return nil, err
	}

	var results []*PendingEntry
	for _, pending := range group.PendingEntries {
		if consumerName == "" || pending.Consumer == consumerName {
			results = append(results, pending)
		}
	}
	return results, nil
}

// ClaimEntries claims pending entries from other consumers.
// Returns the claimed entries and updates their consumer and delivery info.
func (s *Stream) ClaimEntries(groupName, consumerName string, minIdleTime int64, entryIDs []string) ([]*Entry, error) {
	group, err := s.GetGroup(groupName)
	if err != nil {
		return nil, err
	}

	// Ensure consumer exists
	if _, exists := group.Consumers[consumerName]; !exists {
		group.Consumers[consumerName] = &Consumer{
			Name:              consumerName,
			LastSeenTime:      time.Now().UnixMilli(),
			PendingEntriesNum: 0,
		}
	}

	now := time.Now().UnixMilli()
	var claimedEntries []*Entry

	for _, idStr := range entryIDs {
		pending, exists := group.PendingEntries[idStr]
		if !exists {
			continue
		}

		// Check if entry is idle long enough
		if now-pending.DeliveredTime < minIdleTime {
			continue
		}

		// Update the pending entry
		oldConsumer := pending.Consumer
		pending.Consumer = consumerName
		pending.DeliveredTime = now
		pending.DeliveryCount++

		// Update consumer counts
		if oldConsumer != "" && group.Consumers[oldConsumer] != nil {
			group.Consumers[oldConsumer].PendingEntriesNum--
		}
		group.Consumers[consumerName].PendingEntriesNum++

		// Get the actual entry
		id, _ := ParseStreamID(idStr)
		entries, _ := s.RangeScan(id, id)
		if len(entries) > 0 {
			claimedEntries = append(claimedEntries, entries[0])
		}
	}

	return claimedEntries, nil
}
