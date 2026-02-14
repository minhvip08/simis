package handler

import (
	"github.com/minhvip08/simis/internal/connection"
	"github.com/minhvip08/simis/internal/store"
	"github.com/minhvip08/simis/internal/utils"
)

type ZRankHandler struct{}

type zRankParams struct {
	key    string
	member string
}

func parseZRank(args []string) (*zRankParams, error) {
	if len(args) < 2 {
		return nil, ErrInvalidArguments
	}
	return &zRankParams{
		key:    args[0],
		member: args[1],
	}, nil
}

func executeZRank(params *zRankParams) *ExecutionResult {
	sortedSet, _ := store.GetInstance().LoadOrStoreSortedSet(params.key)
	rank := sortedSet.GetRank(params.member)
	result := NewExecutionResult()
	if rank < 0 {
		result.Response = utils.ToNullBulkString()
	} else {
		result.Response = utils.ToRespInt(rank)
	}
	return result
}

func (h *ZRankHandler) Execute(cmd *connection.Command) *ExecutionResult {
	params, err := parseZRank(cmd.Args)
	if err != nil {
		return NewErrorResult(err)
	}
	return executeZRank(params)

}
