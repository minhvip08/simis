package handler

import (
	"github.com/minhvip08/simis/internal/connection"
	err "github.com/minhvip08/simis/internal/error"
	"github.com/minhvip08/simis/internal/store"
	"github.com/minhvip08/simis/internal/utils"
)

type XAddHandler struct {
}

type xAddParams struct {
	key     string
	entryID string
	fields  []string
}

func parseXAddParams(args []string) (*xAddParams, error) {
	if len(args) < 4 {
		return nil, ErrInvalidArguments
	}
	return &xAddParams{
		key:     args[0],
		entryID: args[1],
		fields:  args[2:],
	}, nil
}

func executeXAdd(params *xAddParams) *ExecutionResult {
	db := store.GetInstance()
	stream, _ := db.LoadOrStoreStream(params.key)
	result := NewExecutionResult()
	if stream == nil {

		result.Error = err.ErrFailedToLoadOrStoreStream
		return result
	}

	entry, err := stream.CreateEntry(params.entryID, params.fields...)
	if err != nil {
		result.Error = err
		return result
	}

	err = stream.AppendEntry(entry)
	if err != nil {
		result.Error = err
		return result
	}

	result.Response = utils.ToBulkString(entry.ID.String())
	result.ModifiedKeys = []string{params.key}
	return result
}

func (h *XAddHandler) Execute(cmd *connection.Command) *ExecutionResult {
	params, err := parseXAddParams(cmd.Args)
	if err != nil {
		result := NewExecutionResult()
		result.Error = err
		return result
	}
	return executeXAdd(params)
}
