package handler

import (
	"errors"
	"strconv"

	"github.com/minhvip08/simis/internal/connection"
	ds "github.com/minhvip08/simis/internal/datastructure"
	"github.com/minhvip08/simis/internal/store"
	"github.com/minhvip08/simis/internal/utils"
)

type ZAddHandler struct{}

type zAddParams struct {
	key    string
	kvList []ds.KeyValue
}

func parseZadd(args []string) (*zAddParams, error) {
	if len(args) < 3 {
		return nil, ErrInvalidArguments
	}
	key := args[0]
	scoreMemberArgs := args[1:]
	if len(scoreMemberArgs)%2 != 0 {
		return nil, ErrInvalidArguments
	}

	kvList := make([]ds.KeyValue, len(scoreMemberArgs)/2)
	for i := 0; i < len(scoreMemberArgs); i += 2 {
		score, err := strconv.ParseFloat(scoreMemberArgs[i], 64)
		if err != nil {
			return nil, errors.New(err.Error())
		}
		kvList[i/2] = ds.KeyValue{Key: scoreMemberArgs[i+1], Value: score}
	}

	return &zAddParams{key: key, kvList: kvList}, nil
}

func executeZAdd(params *zAddParams) *ExecutionResult {
	db := store.GetInstance()
	sortedSet, _ := db.LoadOrStoreSortedSet(params.key)
	numAddedValues := sortedSet.Add(params.kvList)
	result := NewExecutionResult()
	result.Response = utils.ToRespInt(numAddedValues)
	return result
}

func (h *ZAddHandler) Execute(cmd *connection.Command) *ExecutionResult {
	params, err := parseZadd(cmd.Args)
	if err != nil {
		return NewErrorResult(err)
	}
	return executeZAdd(params)
}
