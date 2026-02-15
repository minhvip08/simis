# SimiS - Simple In-Memory Storage

SimiS is a lightweight, high-performance in-memory data store similar to Redis, built with Go. It implements the Redis protocol (RESP2) and supports a wide range of data structures and operations.

## Features

### Core Data Structures
- **Strings**: Store and retrieve text or binary data
- **Lists**: Ordered collections with push/pop operations
- **Sorted Sets**: Score-ordered collections with rank-based queries
- **Streams**: Append-only log data structure
- **Geospatial**: Location-based data with radius search

### Advanced Features
- **Transactions**: MULTI/EXEC for atomic command execution
- **Pub/Sub**: Publish-subscribe messaging pattern
- **Master-Slave Replication**: Data replication across multiple instances
- **Persistence**: RDB snapshots for data durability
- **Blocking Operations**: Wait for data availability (BLPOP)
- **Authentication**: ACL-based user authentication
- **Configuration**: Runtime configuration management

## Installation

### Prerequisites
- Go 1.25.0 or higher

### Build from Source
```bash
git clone https://github.com/minhvip08/simis.git
cd simis
go build -o simis ./cmd/server
```

## Usage

### Starting the Server

#### Basic Usage
```bash
./simis
```

#### With Custom Port
```bash
./simis --port 6380
```

#### With Persistence
```bash
./simis --dir /var/lib/simis --dbfilename dump.rdb
```

#### As a Replica
```bash
./simis --replicaof "127.0.0.1 6379"
```

### Command Line Options
- `--port`: Port to listen on (default: 6379)
- `--dir`: Working directory for RDB files (default: /tmp/redis-data)
- `--dbfilename`: Name of the RDB file (default: temp.rdb)
- `--replicaof`: Master server address for replication (format: "host port")

## Supported Commands

### String Commands
- `GET key` - Get the value of a key
- `SET key value [EX seconds] [PX milliseconds]` - Set key to hold string value with optional expiration
- `INCR key` - Increment the integer value of a key
- `ECHO message` - Echo the given string

### List Commands
- `LPUSH key element [element ...]` - Prepend elements to a list
- `RPUSH key element [element ...]` - Append elements to a list
- `LPOP key [count]` - Remove and get the first elements in a list
- `LRANGE key start stop` - Get a range of elements from a list
- `LLEN key` - Get the length of a list
- `BLPOP key [key ...] timeout` - Remove and get the first element in a list, or block until one is available

### Sorted Set Commands
- `ZADD key score member [score member ...]` - Add members to a sorted set
- `ZRANGE key start stop [WITHSCORES]` - Return a range of members by index
- `ZRANK key member` - Determine the index of a member
- `ZCARD key` - Get the number of members in a sorted set
- `ZSCORE key member` - Get the score of a member
- `ZREM key member [member ...]` - Remove members from a sorted set

### Stream Commands
- `XADD key ID field value [field value ...]` - Append a new entry to a stream
- `XRANGE key start end [COUNT count]` - Return a range of elements in a stream
- `XREAD [COUNT count] [BLOCK milliseconds] STREAMS key [key ...] ID [ID ...]` - Read data from streams

### Geospatial Commands
- `GEOADD key longitude latitude member [longitude latitude member ...]` - Add geospatial items
- `GEOPOS key member [member ...]` - Get positions of members
- `GEODIST key member1 member2 [unit]` - Get distance between members
- `GEOSEARCH key FROMLONLAT longitude latitude BYRADIUS radius unit` - Search for members within a radius

### Transaction Commands
- `MULTI` - Mark the start of a transaction block
- `EXEC` - Execute all commands in a transaction
- `DISCARD` - Discard all commands in a transaction

### Pub/Sub Commands
- `SUBSCRIBE channel [channel ...]` - Subscribe to channels
- `UNSUBSCRIBE [channel [channel ...]]` - Unsubscribe from channels
- `PUBLISH channel message` - Post a message to a channel
- `PING [message]` - Ping the server (works in pub/sub mode)

### Replication Commands
- `REPLCONF` - Configure replication parameters
- `PSYNC replicationid offset` - Synchronize with master
- `WAIT numreplicas timeout` - Wait for replication acknowledgment

### Persistence Commands
- `BGSAVE` - Asynchronously save the dataset to disk

### Server Commands
- `PING [message]` - Ping the server
- `INFO [section]` - Get server information
- `CONFIG GET parameter` - Get configuration parameter
- `KEYS pattern` - Find all keys matching a pattern
- `TYPE key` - Determine the type stored at key
- `AUTH username password` - Authenticate to the server

## Architecture

### Components

#### Command Execution Flow
1. **Connection Handler**: Manages client connections and RESP protocol parsing
2. **Router**: Dispatches commands based on connection state (auth, transaction, pub/sub)
3. **Command Queue**: Buffers commands for sequential execution
4. **Executor**: Processes commands, manages blocking operations and transactions
5. **Handlers**: Individual command implementations
6. **Store**: Thread-safe in-memory data storage

#### Key Design Patterns
- **Singleton Pattern**: Config, Store, PubSub Manager
- **Command Pattern**: Each Redis command is a handler
- **Observer Pattern**: Pub/Sub implementation
- **State Pattern**: Connection states (transaction, pub/sub, authenticated)

### Concurrency Model
- Single-threaded command execution via command queue
- Concurrent connection handling with goroutines
- Thread-safe data structures with mutex protection
- Atomic operations for counters and flags

## Protocol

SimiS implements the Redis Serialization Protocol (RESP2):
- Simple Strings: `+OK\r\n`
- Errors: `-ERR message\r\n`
- Integers: `:1000\r\n`
- Bulk Strings: `$6\r\nfoobar\r\n`
- Arrays: `*2\r\n$3\r\nfoo\r\n$3\r\nbar\r\n`

## Client Compatibility

SimiS is compatible with standard Redis clients:
- redis-cli
- redis-py (Python)
- node-redis (Node.js)
- go-redis (Go)
- jedis (Java)

## Performance Considerations

- **In-Memory Storage**: All data kept in RAM for fast access
- **Single-Threaded Execution**: Avoids lock contention for data operations
- **Efficient Serialization**: Fast RESP2 protocol parsing
- **Concurrent Connections**: Multiple clients handled concurrently
- **Background Persistence**: Non-blocking RDB snapshots

## Development

### Project Structure
```
simis/
├── cmd/
│   ├── client/         # CLI client implementation
│   └── server/         # Server entry point
├── internal/
│   ├── acl/           # Access control lists
│   ├── command/       # Command queue and parsing
│   ├── config/        # Configuration management
│   ├── connection/    # Connection state management
│   ├── constants/     # Constant definitions
│   ├── datastructure/ # Data structure implementations
│   ├── error/         # Error definitions
│   ├── executor/      # Command execution engine
│   ├── geo/           # Geospatial utilities
│   ├── handler/       # Command handlers
│   ├── logger/        # Logging utilities
│   ├── pubsub/        # Pub/Sub manager
│   ├── rdb/           # RDB persistence
│   ├── replication/   # Replication manager
│   ├── server/        # Server core logic
│   ├── store/         # Key-value store
│   └── utils/         # Utility functions
└── docs/              # Documentation
```

### Building
```bash
# Build server
go build -o simis ./cmd/server

# Build client
go build -o simis-cli ./cmd/client

# Run tests
go test ./...

# Run with race detector
go run -race ./cmd/server
```

### Testing
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/store/...
```



