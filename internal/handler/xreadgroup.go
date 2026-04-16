package handler

import (
	"strconv"
	"strings"

	"github.com/minhvip08/simis/internal/connection"
	"github.com/minhvip08/simis/internal/datastructure"
	"github.com/minhvip08/simis/internal/store"
	"github.com/minhvip08/simis/internal/utils"
)

type XReadGroupHandler struct {
}

type xReadGroupParams struct {
	groupName  string
	consumer   string
	timeout    int
	count      int
	streamKeys []string
	startIDs   []string
}

func parseXReadGroupArgs(args []string) (*xReadGroupParams, error) {
	// XREADGROUP GROUP group consumer [COUNT count] [BLOCK milliseconds] [NOACK] STREAMS key [key ...] ID [ID ...]
	if len(args) < 5 {
		return nil, ErrInvalidArguments
	}

	params := &xReadGroupParams{
		timeout: -1,
		count:   0,
	}

	i := 0

	// Parse GROUP keyword
	if strings.ToUpper(args[i]) != "GROUP" {
		return nil, ErrInvalidArguments
	}
	i++

	// Parse group name
	params.groupName = args[i]
	i++

	// Parse consumer name
	params.consumer = args[i]
	i++

	// Parse optional parameters
	for i < len(args) {
		arg := strings.ToUpper(args[i])
		if arg == "COUNT" {
			i++
			if i >= len(args) {
				return nil, ErrInvalidArguments
			}
			count, err := strconv.Atoi(args[i])
			if err != nil {
				return nil, ErrInvalidArguments
			}
			params.count = count
			i++
		} else if arg == "BLOCK" {
			i++
			if i >= len(args) {
				return nil, ErrInvalidArguments
			}
			timeout, err := strconv.Atoi(args[i])
			if err != nil {
				return nil, ErrInvalidArguments
			}
			params.timeout = timeout
			i++
		} else if arg == "NOACK" {
			// For now, we'll process this but might not implement it fully
			i++
		} else if arg == "STREAMS" {
			i++
			break
		} else {
			return nil, ErrInvalidArguments
		}
	}

	// Parse STREAMS keyword
	if i >= len(args) {
		return nil, ErrInvalidArguments
	}

	// Get the remaining args as stream keys and IDs
	remainingArgs := args[i:]
	if len(remainingArgs)%2 != 0 {
		return nil, ErrInvalidArguments
	}

	argCount := len(remainingArgs)
	params.streamKeys = remainingArgs[0 : argCount/2]
	params.startIDs = remainingArgs[argCount/2:]

	return params, nil
}

func (h *XReadGroupHandler) Execute(cmd *connection.Command) *ExecutionResult {
	params, err := parseXReadGroupArgs(cmd.Args)
	if err != nil {
		result := NewExecutionResult()
		result.Error = err
		return result
	}

	return executeXReadGroup(params, cmd)
}

func executeXReadGroup(params *xReadGroupParams, cmd *connection.Command) *ExecutionResult {
	result := NewExecutionResult()
	db := store.GetInstance()

	// Load all streams
	streams := make(map[string]interface{})
	for _, key := range params.streamKeys {
		stream, ok := db.GetStream(key)
		if !ok {
			result.Error = ErrInvalidArguments
			return result
		}
		streams[key] = stream
	}

	// Read entries from each stream for the group
	streamEntries := make([]StreamKeyEntries, 0, len(params.streamKeys))

	for i, streamKey := range params.streamKeys {
		stream := streams[streamKey].(*datastructure.Stream)
		startID := params.startIDs[i]

		// For XREADGROUP, if ID is ">", it means get new messages
		// Otherwise, get pending messages for the consumer
		var entries []*datastructure.Entry
		var err error

		if startID == ">" {
			// Read new entries (not yet delivered to this group)
			entries, err = stream.ReadGroupNewEntries(params.groupName, params.consumer, params.count)
		} else {
			// Read pending entries for this consumer
			// This is handled by reading from the pending list
			pendingEntries, err := stream.GetPendingEntries(params.groupName, params.consumer)
			if err != nil {
				result.Error = err
				return result
			}

			// Convert pending entries to regular entries
			entries = make([]*datastructure.Entry, 0, len(pendingEntries))
			for _, pending := range pendingEntries {
				// Get the actual entry
				allEntries, _ := stream.RangeScan(pending.ID, pending.ID)
				if len(allEntries) > 0 {
					entries = append(entries, allEntries[0])
				}
			}
		}

		if err != nil {
			result.Error = err
			return result
		}

		if len(entries) > 0 {
			streamEntries = append(streamEntries, StreamKeyEntries{
				Key:     streamKey,
				Entries: entries,
			})
		}
	}

	if len(streamEntries) == 0 && params.timeout >= 0 {
		// Block and wait for new entries
		cmd.Args = []string{
			"GROUP", params.groupName,
			"CONSUMER", params.consumer,
			"BLOCK", strconv.Itoa(params.timeout),
		}
		if params.count > 0 {
			cmd.Args = append(cmd.Args, "COUNT", strconv.Itoa(params.count))
		}
		cmd.Args = append(cmd.Args, "STREAMS")
		cmd.Args = append(cmd.Args, params.streamKeys...)
		cmd.Args = append(cmd.Args, params.startIDs...)

		result.BlockingTimeout = params.timeout
		result.WaitingKeys = params.streamKeys
		result.Response = utils.ToRespArray(nil)
		return result
	}

	result.Response = toStreamEntriesByKey(streamEntries)
	return result
}
