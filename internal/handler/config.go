package handler

import (
	"fmt"
	"strings"

	"github.com/minhvip08/simis/internal/config"
	"github.com/minhvip08/simis/internal/connection"
	"github.com/minhvip08/simis/internal/utils"
)

// CONFIG Handler
type ConfigHandler struct{}

type configParams struct {
	subcommand string
	key        string
	value      string
}

func parseConfig(args []string) (*configParams, error) {
	if len(args) < 2 {
		return nil, ErrInvalidArguments
	}
	p := &configParams{subcommand: strings.ToUpper(args[0]), key: strings.ToLower(args[1])}
	if len(args) >= 3 {
		p.value = args[2]
	}
	return p, nil
}

func executeConfig(params *configParams) *ExecutionResult {
	cfg := config.GetInstance()
	result := NewExecutionResult()

	switch params.subcommand {
	case "GET":
		switch params.key {
		case "dir":
			result.Response = utils.ToRespArray([]string{params.key, cfg.Dir})
		case "dbfilename":
			result.Response = utils.ToRespArray([]string{params.key, cfg.DBFileName})
		case "maxmemory":
			result.Response = utils.ToRespArray([]string{params.key, fmt.Sprintf("%d", cfg.MaxMemory)})
		case "maxmemory-policy":
			result.Response = utils.ToRespArray([]string{params.key, string(cfg.EvictionPolicy)})
		default:
			result.Error = ErrInvalidArguments
		}

	case "SET":
		if params.value == "" {
			result.Error = ErrInvalidArguments
			return result
		}
		switch params.key {
		case "maxmemory":
			size, err := utils.ParseMemorySize(params.value)
			if err != nil {
				result.Error = err
				return result
			}
			cfg.MaxMemory = size
			result.Response = utils.ToSimpleString("OK")
		case "maxmemory-policy":
			policy := config.EvictionPolicy(params.value)
			switch policy {
			case config.PolicyNoEviction,
				config.PolicyAllKeysLRU,
				config.PolicyVolatileLRU,
				config.PolicyAllKeysRandom,
				config.PolicyVolatileRandom,
				config.PolicyVolatileTTL:
				cfg.EvictionPolicy = policy
				result.Response = utils.ToSimpleString("OK")
			default:
				result.Error = fmt.Errorf("invalid maxmemory-policy: %q", params.value)
			}
		default:
			result.Error = ErrInvalidArguments
		}

	default:
		result.Error = ErrInvalidArguments
	}

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
