package handler

import (
	"strconv"
	"strings"

	"github.com/minhvip08/simis/internal/connection"
	"github.com/minhvip08/simis/internal/store"
)

type XClaimHandler struct {
}

func (h *XClaimHandler) Execute(cmd *connection.Command) *ExecutionResult {
	result := NewExecutionResult()

	// XCLAIM key group consumer min-idle-time id-1 id-2 ... [options]
	if len(cmd.Args) < 5 {
		result.Error = ErrInvalidArguments
		return result
	}

	streamKey := cmd.Args[0]
	groupName := cmd.Args[1]
	consumerName := cmd.Args[2]

	minIdleTime, err := strconv.ParseInt(cmd.Args[3], 10, 64)
	if err != nil {
		result.Error = ErrInvalidArguments
		return result
	}

	entryIDs := []string{}
	i := 4
	for i < len(cmd.Args) {
		arg := strings.ToUpper(cmd.Args[i])
		if arg == "IDLE" || arg == "TIME" || arg == "RETRYCOUNT" || arg == "FORCE" {
			// Skip option and its value
			i += 2
		} else if arg == "JUSTID" {
			i++
		} else {
			// This is an entry ID
			entryIDs = append(entryIDs, cmd.Args[i])
			i++
		}
	}

	db := store.GetInstance()
	stream, ok := db.GetStream(streamKey)
	if !ok {
		result.Error = ErrInvalidArguments
		return result
	}

	claimedEntries, err := stream.ClaimEntries(groupName, consumerName, minIdleTime, entryIDs)
	if err != nil {
		result.Error = err
		return result
	}

	result.Response = toStreamEntries(claimedEntries)
	result.ModifiedKeys = []string{streamKey}
	return result
}
