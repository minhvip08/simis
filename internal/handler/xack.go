package handler

import (
	"github.com/minhvip08/simis/internal/connection"
	"github.com/minhvip08/simis/internal/store"
	"github.com/minhvip08/simis/internal/utils"
)

type XAckHandler struct {
}

func (h *XAckHandler) Execute(cmd *connection.Command) *ExecutionResult {
	result := NewExecutionResult()

	// XACK key group ID-1 ID-2 ... ID-N
	if len(cmd.Args) < 3 {
		result.Error = ErrInvalidArguments
		return result
	}

	streamKey := cmd.Args[0]
	groupName := cmd.Args[1]
	entryIDs := cmd.Args[2:]

	db := store.GetInstance()
	stream, ok := db.GetStream(streamKey)
	if !ok {
		result.Error = ErrInvalidArguments
		return result
	}

	ackedCount, err := stream.AckEntries(groupName, entryIDs)
	if err != nil {
		result.Error = err
		return result
	}

	result.Response = utils.ToRespInt(ackedCount)
	result.ModifiedKeys = []string{streamKey}
	return result
}
