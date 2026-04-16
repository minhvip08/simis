# Redis Stream Consumer Groups - Usage Examples

## Example 1: Simple Message Queue

This example shows how to use consumer groups as a simple message queue with multiple workers.

### Setup

```bash
# Connect to Redis
redis-cli

# Create stream entries (simulating messages)
XADD orders * product "laptop" quantity "2" price "1000"
XADD orders * product "mouse" quantity "5" price "25"
XADD orders * product "keyboard" quantity "3" price "80"
XADD orders * product "monitor" quantity "1" price "300"

# Create consumer group starting from latest new messages
XGROUP CREATE orders order_processors $
```

### Worker 1 Processing

```bash
# Worker 1 reads new orders
XREADGROUP GROUP order_processors worker1 COUNT 2 STREAMS orders >

# Response shows 2 orders
# Process these orders...
# Mark as processed
XACK orders order_processors 1526569495631-0 1526569496625-0
```

### Worker 2 Processing

```bash
# Worker 2 reads next batch
XREADGROUP GROUP order_processors worker2 COUNT 2 STREAMS orders >

# Response shows next 2 orders
# Process and acknowledge
XACK orders order_processors 1526569497631-0 1526569498625-0
```

### Monitor Queue Status

```bash
# Check how many orders are pending
XPENDING orders order_processors

# See detailed info
XPENDING orders order_processors 0 + 100

# See what worker1 is still processing
XPENDING orders order_processors 0 + 100 worker1
```

## Example 2: E-commerce Order Processing Pipeline

This example shows a more complex scenario with different stages.

### Stage 1: Order Validation Group

```redis
# Create validation group
XGROUP CREATE orders validate_group $ MKSTREAM

# Validator reads orders
XREADGROUP GROUP validate_group validator1 STREAMS orders >

# After validation, acknowledge
XACK orders validate_group order_id_1 order_id_2
```

### Stage 2: Payment Processing Group

```redis
# Create payment group
XGROUP CREATE orders payment_group $

# Payment processor reads already-validated orders
XREADGROUP GROUP payment_group payment_worker1 STREAMS orders >

# Process payments...
# Acknowledge after successful payment
XACK orders payment_group order_id_1
```

### Stage 3: Shipping Group

```redis
# Create shipping group
XGROUP CREATE orders shipping_group $

# Shipping worker reads paid orders
XREADGROUP GROUP shipping_group shipper1 STREAMS orders >

# Generate shipping label...
# Acknowledge after shipping
XACK orders shipping_group order_id_1
```

## Example 3: Handling Failures and Retries

### Scenario: Worker Crashes

```redis
# Multiple workers reading from same group
XREADGROUP GROUP processor_group worker1 COUNT 10 STREAMS tasks >
XREADGROUP GROUP processor_group worker2 COUNT 10 STREAMS tasks >

# Worker1 crashes without ACKing entries
# Check pending entries
XPENDING processor_group tasks

# Another worker (or monitoring system) detects stuck tasks
# After 1 hour, claim the tasks from worker1
XCLAIM processor_group tasks worker2 3600000 task_id_1 task_id_2

# Or remove dead worker entirely
XGROUP DELCONSUMER processor_group tasks worker1
```

### Auto-Retry Pattern

```bash
# In your application:

while true:
    # Try to read new messages
    entries = XREADGROUP(GROUP=mygroup, CONSUMER=worker1, STREAMS=mystream, ID=">")
    
    for entry_id, data in entries:
        try:
            # Process entry
            process(data)
            # Success - acknowledge
            XACK(mystream, mygroup, entry_id)
        except Exception as e:
            # Don't acknowledge - leave in pending
            # Will be retried when requested
            log_error(entry_id, e)
    
    # Check for stuck tasks (older than 1 hour)
    pending = XPENDING(mystream, mygroup)
    for task in pending.stuck_tasks:
        # Claim and retry
        XCLAIM(mystream, mygroup, worker1, 3600000, task.id)
```

## Example 4: Batch Processing System

### Setup: Processing User Events

```redis
# Create event stream
XADD events * event_type "user_login" user_id "123"
XADD events * event_type "user_purchase" user_id "456" amount "99.99"
XADD events * event_type "user_logout" user_id "123"
XADD events * event_type "user_register" user_id "789" email "new@example.com"

# Create processing group
XGROUP CREATE events analytics_group 0
```

### Batch Processing

```bash
#!/bin/bash

while true:
    # Read batch of events
    batch_size=100
    events = XREADGROUP(
        GROUP='analytics_group',
        CONSUMER='processor1',
        COUNT=batch_size,
        STREAMS='events',
        ID='>'
    )
    
    if events is empty:
        sleep(1)
        continue
    
    # Process entire batch in one transaction
    for event in events:
        process_analytics(event)
    
    # Acknowledge all processed
    XACK(events, ids=['id1', 'id2', ..., 'id100'])
    
    # Generate report every N batches
    if batch_count % 10 == 0:
        generate_analytics_report()
```

## Example 5: Load Balancing Across Consumers

### Scenario: Uneven Load Distribution

```redis
# Check current load
XPENDING mystream mygroup

# Example output:
# 1) (integer) 100          # Total pending
# 2) "first-id"
# 3) "last-id"
# 4) 1) ["worker1", 60]     # worker1 has 60 pending (overloaded)
#    2) ["worker2", 40]     # worker2 has 40 pending

# Rebalance by claiming some tasks from worker1
XCLAIM mystream mygroup worker2 0 task_51 task_52 task_53 task_54 task_55

# Now distribution is more even
# worker1: 55, worker2: 45
```

## Example 6: Monitoring and Alerting

### Health Check Script

```bash
#!/bin/bash

check_stream_health() {
    stream=$1
    group=$2
    alert_threshold=$3  # max pending count before alert
    
    # Get pending stats
    stats = XPENDING($stream, $group)
    
    pending_count = stats[0]
    consumers = stats[3]
    
    # Check overall backlog
    if pending_count > alert_threshold:
        ALERT("Queue backlog too high: $pending_count")
    fi
    
    # Check for stuck consumers
    for consumer_name, pending in consumers:
        idle_time = get_idle_time($stream, $group, consumer_name)
        
        if idle_time > 30_minutes:  # No progress in 30 min
            ALERT("Consumer $consumer_name may be stuck")
            
            # Optionally auto-remediate
            XGROUP DELCONSUMER $stream $group $consumer_name
        fi
    done
}

# Run checks every minute
while true:
    check_stream_health("tasks", "workers", 1000)
    sleep 60
done
```

## Example 7: Priority Queue with Multiple Groups

### Setup: Different Priority Streams

```redis
# Separate streams for different priorities
XADD tasks_high * type "urgent" task "fix_bug"
XADD tasks_medium * type "normal" task "feature"
XADD tasks_low * type "low" task "cleanup"

# Create separate groups for each priority
XGROUP CREATE tasks_high high_priority_group $
XGROUP CREATE tasks_medium normal_priority_group $
XGROUP CREATE tasks_low low_priority_group $
```

### Processing with Priorities

```bash
def process_tasks():
    # Try high priority first
    high_priority_tasks = XREADGROUP(
        GROUP='high_priority_group',
        CONSUMER='worker',
        COUNT=5,
        STREAMS='tasks_high', '>'
    )
    if high_priority_tasks:
        for task in high_priority_tasks:
            process(task)
            XACK('tasks_high', 'high_priority_group', task.id)
    
    # Then medium
    medium_priority_tasks = XREADGROUP(
        GROUP='normal_priority_group',
        CONSUMER='worker',
        COUNT=5,
        STREAMS='tasks_medium', '>'
    )
    if medium_priority_tasks:
        for task in medium_priority_tasks:
            process(task)
            XACK('tasks_medium', 'normal_priority_group', task.id)
    
    # Finally low priority
    low_priority_tasks = XREADGROUP(
        GROUP='low_priority_group',
        CONSUMER='worker',
        COUNT=5,
        STREAMS='tasks_low', '>'
    )
    if low_priority_tasks:
        for task in low_priority_tasks:
            process(task)
            XACK('tasks_low', 'low_priority_group', task.id)
```

## Testing the Examples

You can test these examples using the Redis CLI:

```bash
# Terminal 1 - Add some test data
redis-cli
XADD mystream * field value
XGROUP CREATE mystream mygroup $

# Terminal 2 - Consumer 1
redis-cli
XREADGROUP GROUP mygroup consumer1 STREAMS mystream >

# Terminal 3 - Consumer 2
redis-cli
XREADGROUP GROUP mygroup consumer2 STREAMS mystream >

# Terminal 4 - Monitor
redis-cli
XPENDING mystream mygroup
```

## Best Practices

1. **Always ACK successfully processed entries** - Free up memory and tracking overhead

2. **Set reasonable COUNT limits** - Don't read unlimited entries
   ```redis
   XREADGROUP GROUP mygroup consumer COUNT 100 STREAMS mystream >
   ```

3. **Monitor pending entries regularly** - Detect problems early
   ```redis
   XPENDING mystream mygroup
   ```

4. **Implement timeouts** - Claim stuck entries after reasonable time
   ```redis
   XCLAIM mystream mygroup consumer2 1800000 stuck_id  # 30 min timeout
   ```

5. **Log failures** - Track why entries aren't being ACKed

6. **Use blocking reads for low-latency** - Reduce CPU usage
   ```redis
   XREADGROUP GROUP mygroup consumer BLOCK 0 STREAMS mystream >
   ```

7. **Clean up dead consumers** - Remove crashed workers
   ```redis
   XGROUP DELCONSUMER mystream mygroup dead_worker
   ```
