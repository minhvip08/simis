package handler

import (
	"github.com/minhvip08/simis/internal/connection"
	"github.com/minhvip08/simis/internal/store"
	"github.com/minhvip08/simis/internal/utils"
)

type ZCardHandler struct{}

type zCardParams struct {
	key string
}

func parseZCard(args []string) (*zCardParams, error) {
	if len(args) < 1 {
		return nil, ErrInvalidArguments
	}
	return &zCardParams{key: args[0]}, nil
}

func executeZCard(params *zCardParams) *ExecutionResult {
	sortedSet, ok := store.GetInstance().LoadOrStoreSortedSet(params.key)
	result := NewExecutionResult()
	if !ok {
		result.Response = utils.ToRespInt(0)
		return result
	}
	cardinality := sortedSet.GetCardinality()
	result.Response = utils.ToRespInt(cardinality)
	return result
}

func (h *ZCardHandler) Execute(cmd *connection.Command) *ExecutionResult {
	params, err := parseZCard(cmd.Args)
	if err != nil {
		return NewErrorResult(err)
	}
	return executeZCard(params)
}
