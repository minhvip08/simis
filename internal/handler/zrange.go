package handler

import (
	"strconv"

	"github.com/minhvip08/simis/internal/connection"
	"github.com/minhvip08/simis/internal/store"
	"github.com/minhvip08/simis/internal/utils"
)

type ZRangeHandler struct{}

type zRangeParams struct {
	key   string
	start int
	end   int
}

func parseZRange(args []string) (*zRangeParams, error) {
	if len(args) < 3 {
		return nil, ErrInvalidArguments
	}
	start, err := strconv.Atoi(args[1])
	if err != nil {
		return nil, ErrInvalidArguments
	}
	end, err := strconv.Atoi(args[2])
	if err != nil {
		return nil, ErrInvalidArguments
	}
	return &zRangeParams{
		key:   args[0],
		start: int(start),
		end:   int(end),
	}, nil
}

func executeZRange(params *zRangeParams) *ExecutionResult {
	sortedSet, ok := store.GetInstance().LoadOrStoreSortedSet(params.key)
	result := NewExecutionResult()
	if !ok {
		result.Response = utils.ToRespArray([]string{})
		return result
	}
	setRange := sortedSet.GetRange(params.start, params.end)
	members := make([]string, len(setRange))
	for i := 0; i < len(setRange); i++ {
		members[i] = setRange[i].Key
	}
	result.Response = utils.ToRespArray(members)
	return result
}

func (h *ZRangeHandler) Execute(cmd *connection.Command) *ExecutionResult {
	params, err := parseZRange(cmd.Args)
	if err != nil {
		return NewErrorResult(err)
	}
	return executeZRange(params)
}
