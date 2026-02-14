package handler

import (
	"github.com/minhvip08/simis/internal/connection"
	"github.com/minhvip08/simis/internal/store"
	"github.com/minhvip08/simis/internal/utils"
)

type ZRemHandler struct{}

type zRemParams struct {
	key     string
	members []string
}

func parseZRem(args []string) (*zRemParams, error) {
	if len(args) < 2 {
		return nil, ErrInvalidArguments
	}
	return &zRemParams{
		key:     args[0],
		members: args[1:],
	}, nil
}

func executeZRem(params *zRemParams) *ExecutionResult {
	sortedSet, ok := store.GetInstance().GetSortedSet(params.key)
	result := NewExecutionResult()
	if !ok {
		result.Response = utils.ToRespInt(0)
		return result
	}
	numRemovedValues := sortedSet.Remove(params.members...)
	result.Response = utils.ToRespInt(numRemovedValues)
	return result
}

func (h *ZRemHandler) Execute(cmd *connection.Command) *ExecutionResult {
	params, err := parseZRem(cmd.Args)
	if err != nil {
		return NewErrorResult(err)
	}
	return executeZRem(params)
}
