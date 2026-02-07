package handler

import (
	"github.com/minhvip08/simis/internal/config"
	"github.com/minhvip08/simis/internal/connection"
	"github.com/minhvip08/simis/internal/utils"
)

// CONFIG Handler
type ConfigHandler struct{}
type configParams struct {
	subcommand string
	key        string
}

func parseConfig(args []string) (*configParams, error) {
	if len(args) < 2 {
		return nil, ErrInvalidArguments
	}
	return &configParams{subcommand: args[0], key: args[1]}, nil
}

func executeConfig(params *configParams) *ExecutionResult {
	cfg := config.GetInstance()
	result := NewExecutionResult()
	switch params.key {
	case "dir":
		result.Response = utils.ToRespArray([]string{params.key, cfg.Dir})
		return result
	case "dbfilename":
		result.Response = utils.ToRespArray([]string{params.key, cfg.DBFileName})
		return result
	}
	result.Error = ErrInvalidArguments
	return result
}

func (h *ConfigHandler) Execute(cmd *connection.Command) *ExecutionResult {
	params, err := parseConfig(cmd.Args)
	if err != nil {
		result := NewExecutionResult()
		result.Error = err
		return result
	}
	return executeConfig(params)
}
