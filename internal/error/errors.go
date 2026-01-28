package error

import "errors"

var (
	ErrNotInTransaction        = errors.New("not in transaction")
	ErrFailedToLoadOrStoreList = errors.New("failed to load or store list")
	ErrAlreadyInTransaction    = errors.New("already in transaction")
	ErrDiscardWithoutMulti     = errors.New("DISCARD without MULTI")
	ErrExecWithoutMulti        = errors.New("EXEC without MULTI")
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
