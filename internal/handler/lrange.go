package handler

import (
	"strconv"

	"github.com/minhvip08/simis/internal/connection"
	"github.com/minhvip08/simis/internal/store"
	"github.com/minhvip08/simis/internal/utils"
)

type LRangeHandler struct{}

type lRangeParams struct {
	key   string
	left  int
	right int
}

func parseLRange(args []string) (*lRangeParams, error) {
	if len(args) < 3 {
		return nil, ErrInvalidArguments
	}

	left, err := strconv.Atoi(args[1])
	if err != nil {
		return nil, ErrInvalidArguments
	}

	right, err := strconv.Atoi(args[2])
	if err != nil {
		return nil, ErrInvalidArguments
	}

	return &lRangeParams{
		key:   args[0],
		left:  left,
		right: right,
	}, nil
}

func executeLRange(params *lRangeParams) *ExecutionResult {
	db := store.GetInstance()

	list, ok := db.GetList(params.key)
	if !ok {
		result := NewExecutionResult()
		result.Response = ToEmptyArray()
		return result
	}

	values := list.LRange(params.left, params.right)

	result := NewExecutionResult()
	result.Response = utils.ToRespArray(values)
	return result
}

func (h *LRangeHandler) Execute(cmd *connection.Command) *ExecutionResult {
	params, err := parseLRange(cmd.Args)
	if err != nil {
		result := NewExecutionResult()
		result.Error = err
		return result
	}
	return executeLRange(params)
}
