# SimiS Architecture Documentation

## Overview

SimiS (Simple In-Memory Storage) is a Redis-like in-memory data store built in Go. It implements a concurrent, event-driven architecture with support for master-slave replication, command queueing, transactions, and blocking operations.

## High-Level Architecture

```
┌─────────────┐
│   Client    │
└──────┬──────┘
       │ TCP Connection
       ▼
┌─────────────────────────────────────┐
│          Server Layer               │
│  ┌─────────────┬─────────────┐     │
│  │   Master    │    Slave    │     │
│  │   Server    │   Server    │     │
│  └──────┬──────┴──────┬──────┘     │
│         │             │            │
└─────────┼─────────────┼────────────┘
          │             │
          ▼             ▼
┌─────────────────────────────────────┐
│       Connection Handler            │
│  ┌───────────────────────────┐     │
│  │   Session Management      │     │
│  │   Request Parsing         │     │
│  │   Response Handling       │     │
│  └────────────┬──────────────┘     │
└───────────────┼─────────────────────┘
                │
                ▼
┌─────────────────────────────────────┐
│          Router Layer               │
│  ┌───────────────────────────┐     │
│  │  Command Routing          │     │
│  │  PubSub Mode Checking     │     │
│  │  Authentication           │     │
│  └────────────┬──────────────┘     │
└───────────────┼─────────────────────┘
                │
                ▼
┌─────────────────────────────────────┐
│       Command Queue                 │
│  ┌───────────────────────────┐     │
│  │  Command Channel          │     │
│  │  Transaction Channel      │     │
│  └────────────┬──────────────┘     │
└───────────────┼─────────────────────┘
                │
                ▼
┌─────────────────────────────────────┐
│         Executor Layer              │
│  ┌─────────────────────────────┐   │
│  │   Command Executor          │   │
│  │   Blocking Manager          │   │
│  │   Transaction Executor      │   │
│  └─────────────┬───────────────┘   │
└─────────────────┼───────────────────┘
                  │
                  ▼
┌─────────────────────────────────────┐
│         Handler Layer               │
│  ┌─────────────────────────────┐   │
│  │   Command Handlers          │   │
│  │   (PING, SET, GET, etc.)    │   │
│  └─────────────────────────────┘   │
└─────────────────────────────────────┘
```

## Core Components

### 1. Main Entry Point (`app/main.go`)

**Responsibilities:**
- Parse command-line flags (port, role, master address, data directory)
- Initialize global configuration
- Start the command queue (singleton pattern)
- Launch the executor in a goroutine
- Create and run the appropriate server (Master or Slave)

**Key Operations:**
```go
parseFlags() → Initialize Config → Start Queue → Start Executor → Run Server
```

### 2. Server Layer (`app/server/`)

#### Server Interface
Defines the contract for server implementations with `Start()` and `Run()` methods.

#### Master Server (`master.go`)
**Responsibilities:**
- Listen on TCP port for incoming connections
- Accept client connections
- Spawn goroutines to handle each connection concurrently
- Manage replica connections

**Flow:**
1. Bind to TCP address
2. Accept connections in a loop
3. For each connection: `go Handle(connection)`

#### Slave Server (`slave.go`)
**Responsibilities:**
- Connect to master server
- Receive replication commands
- Handle client connections (currently under development)

### 3. Connection Layer (`app/connection/`)

#### RedisConnection Structure
Represents a single client connection with the following state:
- **Connection ID**: UUID for unique identification
- **Network Connection**: Underlying TCP connection
- **Transaction State**: `inTransaction`, `queuedCommands`
- **Replication State**: `isMaster`, `suppressResponse`
- **PubSub State**: `inPubSubMode`
- **Authentication**: `username`, `authenticated`

**Key Methods:**
- `SendResponse()`: Write response to client
- `EnqueueCommand()`: Queue command in transaction
- `IsInTransaction()`: Check transaction state
- `IsInPubSubMode()`: Check PubSub state

#### Command Structure
```go
type Command struct {
    Command  string        // Command name (e.g., "SET", "GET")
    Args     []string      // Command arguments
    Response chan string   // Channel for async response
}
```

### 4. Request Processing Pipeline

#### Router (`app/server/router.go`)

**Responsibilities:**
- Route incoming commands to appropriate handlers
- Enforce PubSub mode restrictions
- Handle authentication checks (TODO)
- Manage transaction contexts (TODO)

**Routing Logic:**
1. Check authentication status
2. Validate PubSub mode constraints
3. Handle special cases (PING in PubSub)
4. Route to data command execution

#### Command Parser (`app/command/parser.go`)

**Responsibilities:**
- Parse RESP (Redis Serialization Protocol) format
- Support single and multiple command parsing
- Handle incomplete buffers and streaming data

**Key Functions:**
- `ParseRequest()`: Parse single RESP command
- `ParseMultipleRequests()`: Parse multiple pipelined commands
- Returns: command name, arguments, and any leftover buffer data

### 5. Command Queue (`app/command/queue.go`)

**Pattern:** Singleton with thread-safe initialization

**Responsibilities:**
- Decouple request handling from command execution
- Provide buffered channels for commands and transactions
- Enable asynchronous command processing

**Channels:**
- `cmdQueue`: Buffered channel for individual commands
- `transactions`: Buffered channel for transaction batches

**Benefits:**
- Non-blocking request handling
- Concurrent command execution
- Transaction isolation

### 6. Executor Layer (`app/executor/`)

#### Main Executor (`executor.go`)

**Responsibilities:**
- Consume commands from the queue
- Delegate to appropriate handlers
- Manage blocking operations
- Coordinate transactions

**Execution Flow:**
```go
while true:
    select:
        case cmd from cmdQueue:
            executeCommand(cmd)
        case trans from transactionQueue:
            executeTransaction(trans)
```

**Command Execution:**
1. Lookup handler in registry
2. Execute handler with command
3. Handle errors
4. Send response through command's channel

#### Blocking Command Manager (`blocking.go`)

**Responsibilities:**
- Manage blocking operations (BLPOP, BRPOP, etc.)
- Track blocked clients by key
- Wake up blocked clients when keys become available
- Handle timeout logic

**Key Features:**
- Key-based blocking queues
- Timeout management
- Wake-up notifications

#### Transaction Executor (`transaction.go`)

**Responsibilities:**
- Execute MULTI/EXEC transactions atomically
- Queue commands within transaction context
- Rollback on errors (TODO)
- Ensure ACID properties

### 7. Handler Layer (`app/handler/`)

**Pattern:** Strategy pattern with handler registry

**Handler Interface:**
```go
type Handler interface {
    Execute(cmd *Command) *ExecutionResult
}
```

**ExecutionResult Structure:**
- `Response`: Serialized response string
- `Error`: Execution error (if any)
- `BlockingTimeout`: Timeout for blocking commands
- `WaitingKeys`: Keys to wait on for blocking operations
- `ModifiedKeys`: Keys modified by the command (for replication)

**Handler Registry:**
```go
var Handlers = map[string]Handler{
    "PING": &PingHandler{},
    "SET":  &SetHandler{},
    "GET":  &GetHandler{},
    // ... more handlers
}
```

**Benefits:**
- Easy to add new commands
- Command-specific logic isolation
- Testable handler implementations

### 8. Configuration Management (`app/config/`)

**Singleton Pattern** for global configuration access

**Configuration Options:**
- `Port`: Server listening port
- `Host`: Server host address
- `Role`: Master or Slave
- `MasterAddress`: Master server address (for slaves)
- `Dir`: Working directory for persistence
- `FileName`: RDB file name

**Role Types:**
- `Master`: Primary server handling writes and replication
- `Slave`: Replica server for read scaling and high availability

### 9. Utility Layers

#### RESP Protocol Handler (`app/utils/resp.go`)
- Parse RESP arrays, strings, integers, errors
- Encode responses in RESP format
- Handle nested arrays and bulk strings

#### Logger (`app/logger/`)
- Structured logging
- Different log levels (Info, Warn, Error, Debug)
- Context-aware logging

#### Error Handling (`app/error/`)
- Custom error types
- Error code definitions
- Consistent error responses

#### Constants (`app/constants/`)
- Command definitions
- Queue sizes
- Protocol constants
- Default values

## Data Flow

### Read Command (e.g., GET)

```
Client → Connection → Parser → Router → Queue → Executor → Handler → Storage → Response
```

1. Client sends RESP-encoded command
2. Connection handler reads raw bytes
3. Parser decodes RESP format
4. Router validates and routes command
5. Command enqueued to queue
6. Executor dequeues and dispatches
7. Handler executes business logic
8. Response sent through command channel
9. Connection sends RESP response to client

### Write Command (e.g., SET)

```
Client → ... → Handler → Storage → Replication → Response
```

Same flow as read, with additional:
- Modify in-memory storage
- Propagate to slaves (if master)
- Track modified keys for replication

### Blocking Command (e.g., BLPOP)

```
Client → ... → Handler → BlockingManager (wait) → Wake on key event → Response
```

1. Command executed but key is empty
2. Register client in blocking manager
3. Block with timeout
4. On key modification, wake blocked clients
5. Retry operation and respond

### Transaction (MULTI/EXEC)

```
Client → MULTI → Queue Commands → EXEC → Atomic Execution → Response
```

1. MULTI: Enter transaction mode
2. Commands queued in connection state
3. EXEC: Create transaction batch
4. Executor runs commands atomically
5. All-or-nothing execution
6. Return array of results

## Concurrency Model

### Goroutine Architecture

1. **Main Goroutine**: Server listener accepting connections
2. **Connection Goroutines**: One per client connection (request handling)
3. **Executor Goroutine**: Single goroutine consuming from queue
4. **Background Workers**: Blocking timeout handlers, replication sync

### Thread Safety

- **Command Queue**: Thread-safe channels for communication
- **Shared State**: Protected by mutexes or atomic operations
- **Executor**: Single-threaded execution ensures no race conditions
- **Connection State**: Each connection has isolated state

### Benefits

- Scalable concurrent client handling
- Ordered command execution prevents race conditions
- Lock-free command queueing
- Non-blocking I/O operations

## Replication Architecture (Master-Slave)

### Master Responsibilities
- Handle write operations
- Propagate changes to slaves
- Track connected slaves
- Handle slave synchronization

### Slave Responsibilities
- Connect to master
- Receive replication stream
- Replay commands locally
- Serve read requests (eventually consistent)

### Replication Flow
1. Slave connects to master
2. Full synchronization (RDB transfer)
3. Incremental replication (command stream)
4. Heartbeat/PING for connection health

## Persistence Architecture (TODO)

### RDB (Redis Database) Format
- Periodic snapshots of in-memory data
- Binary format for efficient storage
- Configurable snapshot intervals

### Loading Process
1. Server startup
2. Check for RDB file
3. Deserialize data structures
4. Restore to memory
5. Start accepting connections

## Extensibility Points

### Adding New Commands
1. Create handler implementing `Handler` interface
2. Register in `handler.Handlers` map
3. Implement `Execute()` method
4. Return `ExecutionResult` with response

### Adding New Server Types
1. Implement `Server` interface
2. Define `Start()` and `Run()` methods
3. Update factory in `server.NewServer()`

### Custom Storage Backends
- Abstract storage interface
- Pluggable implementations
- In-memory, persistent, hybrid strategies

## Design Patterns Used

1. **Singleton**: Configuration, Command Queue
2. **Factory**: Server creation based on role
3. **Strategy**: Command handlers
4. **Producer-Consumer**: Command queue and executor
5. **Observer**: Blocking command notifications
6. **Command Pattern**: Command structure with response channel

## Performance Considerations

### Optimization Strategies
- **Connection Pooling**: Reuse network connections
- **Buffered Channels**: Reduce goroutine blocking
- **Lock-Free Operations**: Use channels instead of mutexes
- **Memory Pooling**: Reuse byte buffers for parsing
- **Pipeline Support**: Batch multiple commands

### Bottlenecks
- Single executor goroutine (intentional for ordering)
- Network I/O for replication
- RDB loading on startup

### Scalability
- Horizontal: Add slave replicas for reads
- Vertical: Increase memory and CPU
- Sharding: Distribute keys across instances (future)

## Security Considerations

### Authentication (TODO)
- Username/password authentication
- ACL (Access Control Lists)
- Per-command permissions

### Network Security
- TLS/SSL support (TODO)
- IP whitelisting (TODO)
- Rate limiting (TODO)

## Error Handling Strategy

### Error Categories
1. **Parse Errors**: Invalid RESP format
2. **Command Errors**: Unknown command, wrong args
3. **Execution Errors**: Type errors, out of range
4. **Network Errors**: Connection drops, timeouts
5. **Replication Errors**: Sync failures, lag

### Error Responses
- RESP error format: `-ERR message\r\n`
- Descriptive error messages
- Connection graceful shutdown on fatal errors

## Testing Strategy

### Unit Tests
- Handler execution logic
- RESP parsing
- Command queueing
- Configuration management

### Integration Tests
- Client-server communication
- Transaction execution
- Blocking operations
- Replication sync

### Load Tests
- Concurrent client connections
- High throughput commands
- Memory usage under load

## Future Enhancements

1. **Persistence**: Complete RDB and AOF implementation
2. **Pub/Sub**: Full publish-subscribe support
3. **Cluster Mode**: Distributed hash slots
4. **Lua Scripting**: EVAL/EVALSHA commands
5. **Streams**: Time-series data structures
6. **TLS Support**: Encrypted connections
7. **Monitoring**: INFO, MONITOR commands
8. **Advanced Data Structures**: HyperLogLog, Geo, etc.

## Dependencies

- **Standard Library**: net, bufio, sync, etc.
- **github.com/google/uuid**: UUID generation for connections
- **Minimal External Dependencies**: Keep lightweight and portable

## Deployment Considerations

### Configuration
- Command-line flags for runtime config
- Config file support (future)
- Environment variables (future)

### Monitoring
- Logging for debugging and auditing
- Metrics collection (TODO)
- Health check endpoints (TODO)

### Operations
- Graceful shutdown (TODO)
- Hot reload configuration (TODO)
- Backup and restore procedures

---

**Document Version**: 1.0  
**Last Updated**: January 5, 2026  
**Maintainers**: SimiS Development Team
