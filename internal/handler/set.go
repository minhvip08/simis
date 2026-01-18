package handler

import (
	"strconv"
	"time"

	"github.com/minhvip08/simis/internal/connection"
	"github.com/minhvip08/simis/internal/constants"
	ds "github.com/minhvip08/simis/internal/datastructure"
	"github.com/minhvip08/simis/internal/store"
	"github.com/minhvip08/simis/internal/utils"
)

type SetHandler struct {
}

type setParams struct {
	key       string
	value     string
	expType   string
	expTime   int
	hasExpiry bool
}

func parseSet(args []string) (*setParams, error) {
	if len(args) < 2 {
		return nil, ErrInvalidArguments
	}
	params := &setParams{
		key:       args[0],
		value:     args[1],
		hasExpiry: false,
	}

	if len(args) >= 4 {
		expTime, err := strconv.Atoi(args[3])
		if err != nil {
			return nil, err
		}

		params.expType = args[2]
		params.expTime = expTime
		params.hasExpiry = true
	}

	return params, nil
}

func executeSet(params *setParams) *ExecutionResult {
	kvObj := ds.NewStringObject(params.value)

	db := store.GetInstance()
	db.Store(params.key, &kvObj)

	if params.hasExpiry {
		switch params.expType {
		case "EX":
			kvObj.SetExpiry(time.Duration(params.expTime) * time.Second)
		case "PX":
			kvObj.SetExpiry(time.Duration(params.expTime) * time.Millisecond)
		default:
			result := NewExecutionResult()
			result.Error = ErrInvalidArguments
			return result
		}
		db.Store(params.key, &kvObj)
	}

	result := NewExecutionResult()
	result.Response = utils.ToSimpleString(constants.OKResponse)
	result.ModifiedKeys = []string{params.key}
	return result
}

func (h *SetHandler) Execute(cmd *connection.Command) *ExecutionResult {
	params, err := parseSet(cmd.Args)
	if err != nil {
		result := NewExecutionResult()
		result.Error = err
		return result
	}
	return executeSet(params)
}
