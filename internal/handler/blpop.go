package handler

import (
	"strconv"

	"github.com/minhvip08/simis/internal/connection"
	err "github.com/minhvip08/simis/internal/error"
	"github.com/minhvip08/simis/internal/store"
	"github.com/minhvip08/simis/internal/utils"
)

type BLPopHandler struct{}

type bLPopParams struct {
	keys    []string
	timeout float64
}

func parseBLPop(args []string) (*bLPopParams, error) {
	if len(args) < 2 {
		return nil, ErrInvalidArguments
	}

	timeout, err := strconv.ParseFloat(args[len(args)-1], 64)
	if err != nil {
		return nil, ErrInvalidArguments
	}

	return &bLPopParams{
		keys:    args[:len(args)-1],
		timeout: timeout,
	}, nil
}

func executeBLPop(params *bLPopParams) *ExecutionResult {
	db := store.GetInstance()
	result := NewExecutionResult()

	for _, key := range params.keys {
		dp, _ := db.LoadOrStoreList(key)
		if dp == nil {
			result.Error = err.ErrFailedToLoadOrStoreList
			return result
		}

		value, ok := dp.PopFront()
		if ok {
			result.Response = utils.ToRespArray([]string{key, value})
			result.ModifiedKeys = []string{key}
			return result
		}

	}
	if params.timeout >= 0 {
		result.BlockingTimeout = int(params.timeout * 1000) // convert to milliseconds
		result.WaitingKeys = append(result.WaitingKeys, params.keys...)
		return result
	}
	result.Response = utils.ToRespArray(nil)
	return result
}

func (h *BLPopHandler) Execute(cmd *connection.Command) *ExecutionResult {
	params, err := parseBLPop(cmd.Args)
	if err != nil {
		result := NewExecutionResult()
		result.Error = err
		return result
	}

	return executeBLPop(params)
}
