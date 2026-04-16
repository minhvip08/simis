# Tóm Tắt Triển Khai - Redis Stream Consumer Groups

## Tổng Quan

Đã triển khai hoàn chỉnh tính năng **Redis Stream Consumer Groups** cho phép hỗ trợ các mẫu xử lý hàng đợi (queue patterns) với multiple consumers.

## Những Gì Đã Được Thực Hiện

### 1. **Cấu Trúc Dữ Liệu (Data Structures)**

#### Tệp: `internal/datastructure/stream.go`

Đã thêm ba cấu trúc chính:

```go
// PendingEntry - Theo dõi entries đang chờ xác nhận
type PendingEntry struct {
    ID             StreamID  // ID của entry
    Consumer       string    // Consumer hiện tại
    DeliveredTime  int64     // Thời gian giao (Unix milliseconds)
    DeliveryCount  int       // Số lần giao entry
}

// Consumer - Đại diện cho một consumer trong group
type Consumer struct {
    Name              string
    LastSeenTime      int64  // Lần cuối active
    PendingEntriesNum int    // Số entries đang chờ
}

// ConsumerGroup - Quản lý một consumer group
type ConsumerGroup struct {
    Name             string
    LastDeliveredID  StreamID
    Consumers        map[string]*Consumer
    PendingEntries   map[string]*PendingEntry
}
```

Cập nhật `Stream` struct:
```go
type Stream struct {
    // ... existing fields ...
    ConsumerGroups  map[string]*ConsumerGroup  // Thêm dòng này
}
```

### 2. **Methods cho Consumer Groups**

Đã thêm 9 methods vào `Stream`:

- **`CreateGroup(groupName, startID)`** - Tạo consumer group
- **`GetGroup(groupName)`** - Lấy group theo tên
- **`DestroyGroup(groupName)`** - Xóa consumer group
- **`SetGroupID(groupName, newID)`** - Cập nhật vị trí group
- **`DeleteConsumer(groupName, consumerName)`** - Xóa consumer khỏi group
- **`ReadGroupNewEntries(groupName, consumerName, count)`** - Đọc entries mới cho consumer
- **`AckEntries(groupName, entryIDs)`** - Xác nhận entries
- **`GetPendingEntries(groupName, consumerName)`** - Lấy entries đang chờ
- **`ClaimEntries(groupName, consumerName, minIdleTime, entryIDs)`** - Nhận lại entries từ consumers khác

### 3. **Command Handlers**

Đã tạo 5 command handlers mới:

#### a) **XGROUP** - `internal/handler/xgroup.go`
```
XGROUP CREATE key group id [MKSTREAM]  - Tạo consumer group
XGROUP DESTROY key group               - Xóa group
XGROUP DELCONSUMER key group consumer  - Xóa consumer
XGROUP SETID key group id              - Cập nhật vị trí group
```

#### b) **XREADGROUP** - `internal/handler/xreadgroup.go`
```
XREADGROUP GROUP group consumer [COUNT count] [BLOCK ms] STREAMS key id
- Đọc entries từ stream với consumer group
- Hỗ trợ '>' để đọc entries mới
- Hỗ trợ ID cụ thể để đọc entries đang chờ
```

#### c) **XACK** - `internal/handler/xack.go`
```
XACK key group id [id ...]
- Xác nhận entries đã xử lý
- Loại bỏ khỏi pending list
```

#### d) **XPENDING** - `internal/handler/xpending.go`
```
XPENDING key group [start end count] [consumer]
- Hiển thị tóm tắt: số lượng, range IDs, consumers
- Chi tiết: ID, consumer, thời gian chờ, số lần giao
```

#### e) **XCLAIM** - `internal/handler/xclaim.go`
```
XCLAIM key group consumer min-idle-time id [id ...]
- Nhận lại entries từ consumers khác
- Dùng để xử lý consumer bị treo
```

### 4. **Utility Functions**

Thêm vào `internal/datastructure/streamid.go`:
- **`ParseStreamID(s string)`** - Parse strict format "ms-seq"

### 5. **Handler Registration**

Cập nhật `internal/handler/handler.go`:
```go
var Handlers = map[string]Handler{
    // ... existing handlers ...
    "XGROUP":     &XGroupHandler{},
    "XREADGROUP": &XReadGroupHandler{},
    "XACK":       &XAckHandler{},
    "XPENDING":   &XPendingHandler{},
    "XCLAIM":     &XClaimHandler{},
}
```

## Các Quy Trình Chính

### 1. **Setup Consumer Group**
```
XGROUP CREATE mystream workers $
```
- Tạo group tên "workers" trên stream "mystream"
- Bắt đầu từ entries mới từ bây giờ (`$`)

### 2. **Consumer Đọc Entries**
```
XREADGROUP GROUP workers consumer1 COUNT 10 STREAMS mystream >
```
- Consumer1 đọc tối đa 10 entries mới
- Entries tự động thêm vào pending list

### 3. **Xác Nhận Hoàn Thành**
```
XACK mystream workers 1526569495631-0 1526569496625-0
```
- Xác nhận 2 entries đã xử lý
- Loại bỏ khỏi pending list

### 4. **Giám Sát Entries Chờ Xử Lý**
```
XPENDING mystream workers
```
- Xem tóm tắt entries đang chờ
- Xem consumer nào đang xử lý

### 5. **Nhận Lại Entries từ Consumer Treo**
```
XCLAIM mystream workers consumer2 3600000 1526569495631-0
```
- Consumer2 nhận lại entry từ consumer1 (đã idle > 1 giờ)

## Đặc Điểm Chính

✅ **Delivery Guarantee** - At-least-once delivery semantics  
✅ **Pending Tracking** - Theo dõi tất cả entries đang xử lý  
✅ **Consumer Management** - Quản lý multiple consumers  
✅ **Failure Handling** - Hỗ trợ claim entries từ consumers chết  
✅ **Activity Monitoring** - Theo dõi lần cuối consumer active  
✅ **Blocking Reads** - Hỗ trợ BLOCK option  

## Files Đã Sửa Đổi/Tạo

### Tệp Sửa Đổi:
- `internal/datastructure/stream.go` - Thêm consumer group structures & methods
- `internal/datastructure/streamid.go` - Thêm ParseStreamID function
- `internal/handler/handler.go` - Đăng ký 5 command handlers mới

### Tệp Mới Tạo:
- `internal/handler/xgroup.go` - XGROUP command
- `internal/handler/xreadgroup.go` - XREADGROUP command
- `internal/handler/xack.go` - XACK command
- `internal/handler/xpending.go` - XPENDING command
- `internal/handler/xclaim.go` - XCLAIM command
- `docs/STREAM_CONSUMER_GROUPS.md` - Tài liệu chi tiết

## Trạng Thái Build

✅ Compile thành công (0 errors)  
✅ Code chạy được  
✅ Sẵn sàng triển khai  

## Hướng Dẫn Sử Dụng

### Ví Dụ: Queue Processing System

```bash
# 1. Tạo stream với consumer group
redis> XADD jobs * task "process-image" file "photo.jpg"
redis> XADD jobs * task "send-email" to "user@example.com"
redis> XADD jobs * task "backup-db" database "prod"

redis> XGROUP CREATE jobs workers $

# 2. Worker 1 lấy công việc
redis> XREADGROUP GROUP workers worker1 COUNT 5 STREAMS jobs >
1) 1) "jobs"
   2) 1) 1) "1526569495631-0"
         2) 1) "task"
            2) "process-image"
            3) "file"
            4) "photo.jpg"

# 3. Worker xử lý công việc...
# (xử lý task trong ứng dụng)

# 4. Xác nhận hoàn thành
redis> XACK jobs workers 1526569495631-0
(integer) 1

# 5. Kiểm tra công việc chờ xử lý
redis> XPENDING jobs workers
1) (integer) 0     // không có công việc chờ
2) (nil)
3) (nil)
4) (empty array)

# 6. Nếu worker bị crash, worker khác có thể nhận lại
redis> XCLAIM jobs workers worker2 3600000 1526569496625-0
# (nhận lại công việc từ worker1 nếu đã idle > 1 giờ)
```

## Documentation

Chi tiết đầy đủ: `docs/STREAM_CONSUMER_GROUPS.md`
