package handler

import (
	"github.com/minhvip08/simis/internal/connection"
	"github.com/minhvip08/simis/internal/datastructure"
	err "github.com/minhvip08/simis/internal/error"
	"github.com/minhvip08/simis/internal/store"
)

type XRangeHandler struct {
}

type xRagneParams struct {
	key     string
	startID string
	endID   string
}

func parseXRangeParams(args []string) (*xRagneParams, error) {
	if len(args) < 3 {
		return nil, ErrInvalidArguments
	}
	return &xRagneParams{
		key:     args[0],
		startID: args[1],
		endID:   args[2],
	}, nil
}

func executeXRange(params *xRagneParams) *ExecutionResult {
	db := store.GetInstance()
	stream, _ := db.LoadOrStoreStream(params.key)
	if stream == nil {
		result := NewExecutionResult()
		result.Error = err.ErrFailedToLoadOrStoreStream
		return result
	}

	lastID := stream.GetLastStreamID()
	startID, err := datastructure.ParseStartStreamID(params.startID, lastID)
	if err != nil {
		result := NewExecutionResult()
		result.Error = err
		return result
	}

	endID, err := datastructure.ParseEndStreamID(params.endID)

	if err != nil {
		result := NewExecutionResult()
		result.Error = err
		return result
	}

	entries, err := stream.RangeScan(startID, endID)
	if err != nil {
		result := NewExecutionResult()
		result.Error = err
		return result
	}

	response := NewExecutionResult()
	response.Response = toStreamEntries(entries)
	return response
}

func (h *XRangeHandler) Execute(cmd *connection.Command) *ExecutionResult {
	params, err := parseXRangeParams(cmd.Args)
	if err != nil {
		result := NewExecutionResult()
		result.Error = err
		return result
	}
	return executeXRange(params)
}
