package error

import "errors"

var (
	ErrNotInTransaction        = errors.New("not in transaction")
	ErrFailedToLoadOrStoreList = errors.New("failed to load or store list")
	ErrAlreadyInTransaction    = errors.New("already in transaction")
	ErrDiscardWithoutMulti     = errors.New("DISCARD without MULTI")
	ErrExecWithoutMulti        = errors.New("EXEC without MULTI")
	ErrInvalidUserPassword = errors.New("WRONGPASS invalid username-password pair or user is disabled.")
	ErrAuthenticationRequired = errors.New("NOAUTH Authentication required.")
)

var (
	ErrInvalidArguments = errors.New("Invalid arguments")
	ErrTimeout          = errors.New("Timeout")
	ErrUserNotFound     = errors.New("User not found")
)

var (
	ErrBadLengthListPack      = errors.New("corrupt listpack: bad length varint during rebuild")
	ErrTruncatedEntryListPack = errors.New("corrupt listpack: truncated entry during rebuild")
	ErrCountMismatchListPack  = errors.New("corrupt listpack: count mismatch during rebuild")
)

var (
	ErrInvalidFieldCount   = errors.New("invalid field count")
	ErrInvalidLengthVarint = errors.New("invalid length varint")
	ErrTruncatedData       = errors.New("truncated data")
)

var (
	ErrFailedToLoadOrStoreStream = errors.New("failed to load or store stream")
)

var (
	ErrInvalidRDBHeader = errors.New("invalid RDB header")
	ErrUnexpectedEOF    = errors.New("unexpected EOF while parsing RDB")
	ErrUnsupportedType  = errors.New("unsupported RDB value type or encoding")
)