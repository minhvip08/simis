package handler

import (
	"strconv"

	"github.com/minhvip08/simis/internal/connection"
	"github.com/minhvip08/simis/internal/store"
	"github.com/minhvip08/simis/internal/utils"
)

type ZScoreHandler struct{}

type zScoreParams struct {
	key    string
	member string
}

func parseZScore(args []string) (*zScoreParams, error) {
	if len(args) < 2 {
		return nil, ErrInvalidArguments
	}
	return &zScoreParams{
		key:    args[0],
		member: args[1],
	}, nil
}

func executeZScore(params *zScoreParams) *ExecutionResult {
	sortedSet, ok := store.GetInstance().GetSortedSet(params.key)
	result := NewExecutionResult()
	if !ok {
		result.Response = utils.ToNullBulkString()
		return result
	}

	score, found := sortedSet.GetScore(params.member)
	if !found {
		result.Response = utils.ToNullBulkString()
	} else {
		result.Response = utils.ToBulkString(strconv.FormatFloat(score, 'f', -1, 64))
	}
	return result
}

func (h *ZScoreHandler) Execute(cmd *connection.Command) *ExecutionResult {
	params, err := parseZScore(cmd.Args)
	if err != nil {
		return NewErrorResult(err)
	}
	return executeZScore(params)
}
