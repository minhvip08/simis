package handler

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/minhvip08/simis/internal/config"
	"github.com/minhvip08/simis/internal/connection"
	"github.com/minhvip08/simis/internal/logger"
	"github.com/minhvip08/simis/internal/rdb"
	"github.com/minhvip08/simis/internal/store"
	"github.com/minhvip08/simis/internal/utils"
)

type BGSaveHandler struct{}

type bgsaveParams struct{}

func parseBGSave(args []string) (*bgsaveParams, error) {
	if len(args) > 0 {
		return nil, ErrInvalidArguments
	}
	return &bgsaveParams{}, nil
}

func executeBGSave(params *bgsaveParams) *ExecutionResult {
	result := NewExecutionResult()

	cfg := config.GetInstance()
	if cfg.Dir == "" || cfg.DBFileName == "" {
		result.Error = fmt.Errorf("ERR BGSAVE failed: dir or dbfilename not configured")
		return result
	}

	go func() {
		kvStore := store.GetInstance()
		rdbPath := filepath.Join(cfg.Dir, cfg.DBFileName)

		logger.Info("Background save started", "path", rdbPath)

		rdbData, err := rdb.GenerateRDB(kvStore)
		if err != nil {
			logger.Error("Background save failed during RDB generation", "error", err)
			return
		}

		if err := os.WriteFile(rdbPath, rdbData, 0644); err != nil {
			logger.Error("Background save failed during file write", "path", rdbPath, "error", err)
			return
		}

		logger.Info("Background save completed successfully", "path", rdbPath, "size", len(rdbData))
	}()

	result.Response = utils.ToSimpleString("Background saving started")
	return result
}

func (h *BGSaveHandler) Execute(cmd *connection.Command) *ExecutionResult {
	params, err := parseBGSave(cmd.Args)
	if err != nil {
		result := NewExecutionResult()
		result.Error = err
		return result
	}

	return executeBGSave(params)
}
