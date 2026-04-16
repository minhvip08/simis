# Redis Stream Consumer Groups Implementation

This document describes the consumer group functionality implemented for Redis Streams in Simis.

## Overview

Consumer groups enable multiple consumers to work together and process stream entries in parallel. Each consumer group tracks which entries have been delivered to consumers and which have been acknowledged.

## Data Structures

### ConsumerGroup
```go
type ConsumerGroup struct {
    Name            string                    // Group name
    LastDeliveredID StreamID                  // Last entry ID delivered to any consumer
    Consumers       map[string]*Consumer      // Map of consumer name to consumer info
    PendingEntries  map[string]*PendingEntry // Entries waiting for acknowledgment
}
```

### Consumer
```go
type Consumer struct {
    Name              string // Consumer name
    LastSeenTime      int64  // Unix milliseconds of last activity
    PendingEntriesNum int    // Count of pending entries for this consumer
}
```

### PendingEntry
```go
type PendingEntry struct {
    ID            StreamID // Entry ID
    Consumer      string   // Current consumer claiming the entry
    DeliveredTime int64    // Unix milliseconds when delivered
    DeliveryCount int      // Number of delivery attempts
}
```

## Commands Implemented

### XGROUP - Consumer Group Management

#### Create a Consumer Group
```
XGROUP CREATE key group id [MKSTREAM]
```
Creates a new consumer group for a stream. The `id` parameter can be:
- `$` - Start consuming new messages from now
- `0` or `0-0` - Start from the beginning
- A specific stream ID (e.g., `1526569495631-0`)

Options:
- `MKSTREAM` - Create the stream if it doesn't exist

Example:
```
XGROUP CREATE mystream mygroup $
XGROUP CREATE mystream mygroup 0 MKSTREAM
```

#### Destroy a Consumer Group
```
XGROUP DESTROY key group
```
Removes a consumer group and all its associated data.

Example:
```
XGROUP DESTROY mystream mygroup
```

#### Delete a Consumer
```
XGROUP DELCONSUMER key group consumer
```
Removes a consumer from a group. Returns the number of pending entries that were reassigned.

Example:
```
XGROUP DELCONSUMER mystream mygroup consumer1
```

#### Set the Group's Last Delivered ID
```
XGROUP SETID key group id
```
Updates the last delivered ID for a consumer group, which changes where new entries will be read from.

Example:
```
XGROUP SETID mystream mygroup 1526569495631-0
```

### XREADGROUP - Read from a Consumer Group

```
XREADGROUP GROUP group consumer [COUNT count] [BLOCK milliseconds] STREAMS key [key ...] id [id ...]
```

Reads entries from a stream as a member of a consumer group.

Parameters:
- `GROUP group consumer` - Consumer group and consumer name
- `COUNT count` - Maximum number of entries to return
- `BLOCK milliseconds` - Block if no entries are available
- `STREAMS key ... id ...` - Stream keys and starting positions
  - `>` - Read only new entries (not yet delivered)
  - Entry ID or `0` - Read specific or all pending entries

Example:
```
XREADGROUP GROUP mygroup consumer1 COUNT 10 STREAMS mystream >
XREADGROUP GROUP mygroup consumer2 STREAMS mystream 0
```

### XACK - Acknowledge Entries

```
XACK key group id [id ...]
```

Marks entries as processed by removing them from the group's pending list. Returns the number of entries acknowledged.

Example:
```
XACK mystream mygroup 1526569495631-0 1526569496625-0
```

### XPENDING - Get Pending Entries

```
XPENDING key group [start end count] [consumer]
```

Shows pending entries for a consumer group.

#### Summary Format (no range specified)
```
XPENDING mystream mygroup
```
Returns: `[count, start_id, end_id, [[consumer_name, pending_count], ...]]`

#### Detailed Format (with range)
```
XPENDING mystream mygroup 0 + 10
XPENDING mystream mygroup 0 + 10 consumer1
```

Returns array of: `[id, consumer, idle_time_ms, delivery_count]`

### XCLAIM - Claim Pending Entries

```
XCLAIM key group consumer min-idle-time id [id ...] [options]
```

Allows a consumer to claim pending entries from other consumers. This is useful for handling dead consumers.

Parameters:
- `min-idle-time` - Minimum idle time (ms) before entry can be claimed
- `id [id ...]` - IDs of entries to claim

Options (parsed but not fully implemented):
- `IDLE ms` - Set idle time
- `TIME ms-unix-time` - Set delivery time
- `RETRYCOUNT count` - Set delivery count
- `FORCE` - Claim even if not idle long enough
- `JUSTID` - Return only IDs

Example:
```
XCLAIM mystream mygroup consumer1 3600000 1526569495631-0 1526569496625-0
```

## Usage Example

### Setup
```
XADD mystream * field1 value1
XADD mystream * field2 value2
XADD mystream * field3 value3

XGROUP CREATE mystream workers $
```

### Consumer Reading
```
# Consumer1 reads new entries
XREADGROUP GROUP workers consumer1 COUNT 2 STREAMS mystream >

# Process entries...

# Acknowledge processed entries
XACK mystream workers 1526569495631-0
```

### Monitor Pending Entries
```
XPENDING mystream workers
XPENDING mystream workers 0 + 100
```

### Claim Stuck Entries
```
# If consumer1 dies, another consumer can claim its work
XCLAIM mystream workers consumer2 3600000 1526569495631-0
```

## Implementation Notes

1. **Thread Safety**: Consumer group operations are protected by the stream's access through the KVStore.

2. **Pending Entries**: Entries remain in the pending list until explicitly acknowledged with XACK.

3. **Delivery Tracking**: The `DeliveryCount` tracks how many times an entry has been delivered, useful for identifying problematic entries.

4. **Consumer Cleanup**: When a consumer is deleted, its pending entries are not automatically acknowledged; they return to the group queue for reassignment.

5. **Blocking Reads**: XREADGROUP supports blocking when no new entries are available, similar to XREAD.

## Future Enhancements

- Full XCLAIM options support (IDLE, TIME, RETRYCOUNT, FORCE)
- Consumer group persistence in RDB files
- Consumer activity monitoring and auto-cleanup
- Dead letter queue handling for entries with high delivery counts
- Consumer group events/notifications
