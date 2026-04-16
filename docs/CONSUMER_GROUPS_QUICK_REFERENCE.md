# Redis Stream Consumer Groups - Quick Reference

## Command Summary

| Command | Syntax | Purpose |
|---------|--------|---------|
| **XGROUP CREATE** | `XGROUP CREATE key group id [MKSTREAM]` | Create consumer group |
| **XGROUP DESTROY** | `XGROUP DESTROY key group` | Delete consumer group |
| **XGROUP DELCONSUMER** | `XGROUP DELCONSUMER key group consumer` | Remove consumer from group |
| **XGROUP SETID** | `XGROUP SETID key group id` | Update group's read position |
| **XREADGROUP** | `XREADGROUP GROUP group consumer [COUNT n] [BLOCK ms] STREAMS key id` | Read entries from group |
| **XACK** | `XACK key group id [id ...]` | Acknowledge processed entries |
| **XPENDING** | `XPENDING key group [start end count] [consumer]` | View pending entries |
| **XCLAIM** | `XCLAIM key group consumer min-idle-ms id [id ...]` | Claim pending entries |

## Typical Workflow

### 1. Initialize (One-time setup)
```redis
# Create stream entries
XADD mystream * field1 value1
XADD mystream * field2 value2

# Create consumer group starting from latest
XGROUP CREATE mystream mygroup $

# Or start from beginning
XGROUP CREATE mystream mygroup 0

# Or start from specific ID
XGROUP CREATE mystream mygroup 1526569495631-0
```

### 2. Consumer Loop

```redis
# Each consumer processes entries
XREADGROUP GROUP mygroup consumer1 COUNT 10 STREAMS mystream >

# After processing successfully
XACK mystream mygroup id1 id2 id3

# Or if processing fails, entry remains pending for retry
```

### 3. Monitor Progress

```redis
# View pending entries summary
XPENDING mystream mygroup

# View pending entries for specific consumer
XPENDING mystream mygroup 0 + 100 consumer1

# Check consumer status
XPENDING mystream mygroup
```

### 4. Handle Failures

```redis
# If consumer crashes, claim its work
XCLAIM mystream mygroup consumer2 3600000 id1 id2 id3

# Or remove the dead consumer
XGROUP DELCONSUMER mystream mygroup consumer1
```

## ID Format Reference

| ID Format | Meaning |
|-----------|---------|
| `$` | Latest message (for XGROUP CREATE) |
| `0` or `0-0` | Earliest message |
| `>` | New messages not yet delivered (for XREADGROUP) |
| `1526569495631-0` | Specific entry ID (milliseconds-sequence) |

## Return Value Examples

### XREADGROUP Success
```
1) 1) "mystream"
   2) 1) 1) "1526569495631-0"
         2) 1) "field1"
            2) "value1"
            3) "field2"
            2) "value2"
```

### XPENDING Summary
```
1) (integer) 3              # 3 entries pending
2) "1526569495631-0"        # Oldest pending ID
3) "1526569496625-0"        # Newest pending ID
4) 1) 1) "consumer1"
      2) (integer) 2        # 2 pending entries for consumer1
   2) 1) "consumer2"
      2) (integer) 1        # 1 pending entry for consumer2
```

### XPENDING Detailed
```
1) 1) "1526569495631-0"     # Entry ID
   2) "consumer1"           # Current consumer
   3) (integer) 5000        # Idle time (ms)
   4) (integer) 1           # Delivery count
```

## Error Scenarios

| Error | Cause | Solution |
|-------|-------|----------|
| NOGROUP | Group doesn't exist | Use XGROUP CREATE |
| BUSYGROUP | Group already exists | Use different name or DESTROY first |
| Stream doesn't exist | Key doesn't point to stream | Create with XADD or use MKSTREAM |
| No group membership | Consumer not in group | XREADGROUP will auto-add |

## Common Patterns

### Pattern 1: Simple Task Queue
```redis
# Setup
XGROUP CREATE tasks workers $ MKSTREAM

# Producer adds tasks
XADD tasks * action "send-email" to "user@example.com"

# Worker processes
XREADGROUP GROUP workers w1 STREAMS tasks >
XACK tasks workers <id>
```

### Pattern 2: Competing Consumers
```redis
# Multiple workers reading same stream
XREADGROUP GROUP workers w1 COUNT 1 STREAMS tasks >
XREADGROUP GROUP workers w2 COUNT 1 STREAMS tasks >
XREADGROUP GROUP workers w3 COUNT 1 STREAMS tasks >

# Each gets different entries automatically
```

### Pattern 3: Retry Failed Items
```redis
# View failures
XPENDING tasks workers 0 + 1000

# Claim unprocessed after timeout (1 hour)
XCLAIM tasks workers retry_handler 3600000 <stuck_ids>

# Or remove dead consumer
XGROUP DELCONSUMER tasks workers dead_worker
```

### Pattern 4: Load Balancing
```redis
# Get status
XPENDING tasks workers

# Rebalance if consumer has too many pending
XCLAIM tasks workers balanced_w1 0 <ids_from_overloaded>
```

## Performance Tips

1. **Use COUNT limit** - Don't read unlimited entries
   ```redis
   XREADGROUP GROUP mygroup consumer COUNT 10 STREAMS mystream >
   ```

2. **ACK immediately after processing** - Free up resources
   ```redis
   # Process
   XACK mystream mygroup id
   ```

3. **Monitor pending entries** - Detect stuck consumers
   ```redis
   XPENDING mystream mygroup
   ```

4. **Clean up dead consumers** - Use DELCONSUMER
   ```redis
   XGROUP DELCONSUMER mystream mygroup dead_consumer
   ```

5. **Use blocking reads for real-time** - Reduce polling overhead
   ```redis
   XREADGROUP GROUP mygroup consumer BLOCK 0 STREAMS mystream >
   ```

## FAQ

**Q: What happens if consumer crashes without ACKing?**  
A: Entry stays pending. Other consumers can claim it with XCLAIM or read same entry again.

**Q: Can multiple consumers process the same entry?**  
A: No, only one consumer owns an entry at a time. Use XCLAIM to transfer ownership.

**Q: How to reset consumer group?**  
A: Use `XGROUP DESTROY` then `XGROUP CREATE` to restart.

**Q: What's the max pending entries?**  
A: No hard limit, but monitor with XPENDING to detect issues.

**Q: How to process entries in order?**  
A: Use single consumer instead of group, or implement ordering in application.

## Related Commands

- `XADD` - Add entries to stream
- `XREAD` - Read without consumer group
- `XRANGE` - Scan all entries
- `XLEN` - Get stream length
- `XTRIM` - Remove old entries
