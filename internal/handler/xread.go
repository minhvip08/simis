package handler

import (
	"strconv"

	"github.com/minhvip08/simis/internal/connection"
	"github.com/minhvip08/simis/internal/datastructure"
	"github.com/minhvip08/simis/internal/utils"
)

type XReadHandler struct {
}

type xReadParams struct {
	timeout    int
	streamKeys []string
	startIDs   []string
}

func parseXReadParams(args []string) (*xReadParams, error) {
	timeout, streamKeys, startIDs, err := parseXReadArgs(args)
	if err != nil {
		return nil, err
	}
	return &xReadParams{
		timeout:    timeout,
		streamKeys: streamKeys,
		startIDs:   startIDs,
	}, nil
}

func executeXread(params *xReadParams, cmd *connection.Command) *ExecutionResult {
	streams, err := loadStreams(params.streamKeys)
	result := NewExecutionResult()

	if err != nil {
		result.Error = err
		return result
	}

	streamEntries, err := readEntriesFromStreams(streams, params.streamKeys, params.startIDs)
	if err != nil {
		result.Response = utils.ToRespArray(nil)
		result.Error = err
		return result
	}

	if len(streamEntries) == 0 && params.timeout >= 0 {
		lastIDs := make([]string, len(streams))

		for i, stream := range streams {
			lastID := stream.GetLastStreamID()
			if lastID == nil {
				lastID = &datastructure.StreamID{Ms: 0, Seq: 0}
			}
			lastIDs[i] = lastID.String()
		}
		newArgs := []string{
			"BLOCK", strconv.Itoa(params.timeout),
			"STREAMS",
		}
		newArgs = append(newArgs, params.streamKeys...)
		newArgs = append(newArgs, lastIDs...)

		cmd.Args = newArgs
		result.BlockingTimeout = params.timeout
		result.WaitingKeys = append(result.WaitingKeys, params.streamKeys...)
		result.Response = utils.ToRespArray(nil)
		return result

	}

	result.Response = toStreamEntriesByKey(streamEntries)
	return result
}

func (h *XReadHandler) Execute(cmd *connection.Command) *ExecutionResult {
	params, err := parseXReadParams(cmd.Args)
	if err != nil {
		result := NewExecutionResult()
		result.Error = err
		return result
	}
	return executeXread(params, cmd)
}
