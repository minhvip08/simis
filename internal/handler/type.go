package handler

import (
	"github.com/minhvip08/simis/internal/connection"
	"github.com/minhvip08/simis/internal/datastructure"
	"github.com/minhvip08/simis/internal/store"
	"github.com/minhvip08/simis/internal/utils"
)

type TypeHandler struct {
}

type typeParams struct {
	key string
}

func parseType(args []string) (*typeParams, error) {
	if len(args) < 1 {
		return nil, ErrInvalidArguments
	}
	return &typeParams{
		key: args[0],
	}, nil
}

func executeType(params *typeParams) *ExecutionResult {
	db := store.GetInstance()
	val, ok := db.GetValue(params.key)
	result := NewExecutionResult()
	if !ok {
		result.Response = utils.ToSimpleString(DataTypeNone)
		return result
	}

	switch val.Get().Kind() {
	case datastructure.RString:
		result.Response = utils.ToSimpleString(DataTypeString)
	case datastructure.RList:
		result.Response = utils.ToSimpleString(DataTypeList)
	case datastructure.RStream:
		result.Response = utils.ToSimpleString(DataTypeStream)
	case datastructure.RSortedSet:
		result.Response = utils.ToSimpleString(DataTypeSet)
	default:
		result.Response = utils.ToSimpleString(DataTypeNone)
	}
	return result

}

func (h *TypeHandler) Execute(cmd *connection.Command) *ExecutionResult {
	params, err := parseType(cmd.Args)
	if err != nil {
		result := NewExecutionResult()
		result.Error = err
		return result
	}
	return executeType(params)
}
