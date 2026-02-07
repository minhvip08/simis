package handler

import (
	"github.com/minhvip08/simis/internal/connection"
	"github.com/minhvip08/simis/internal/store"
	"github.com/minhvip08/simis/internal/utils"
)

type KeysHandler struct{}

type keysParams struct {
	pattern string
}

func parseKeysParams(args []string) (*keysParams, error) {
	if len(args) < 1 {
		return nil, ErrInvalidArguments
	}

	// Currently, only the wildcard pattern "*" is supported
	if args[0] != "*" {
		return nil, ErrInvalidArguments
	}
	return &keysParams{pattern: args[0]}, nil
}

func executeKeys(params *keysParams) *ExecutionResult {
	db := store.GetInstance()
	keys := db.GetAllKeys()
	result := NewExecutionResult()
	result.Response = utils.ToRespArray(keys)
	return result
}

func (h *KeysHandler) Execute(cmd *connection.Command) *ExecutionResult {
	params, err := parseKeysParams(cmd.Args)
	if err != nil {
		result := NewExecutionResult()
		result.Error = err
		return result
	}
	return executeKeys(params)
}
