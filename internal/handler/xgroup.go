package handler

import (
	"strings"

	"github.com/minhvip08/simis/internal/connection"
	"github.com/minhvip08/simis/internal/store"
	"github.com/minhvip08/simis/internal/utils"
)

type XGroupHandler struct {
}

func (h *XGroupHandler) Execute(cmd *connection.Command) *ExecutionResult {
	result := NewExecutionResult()

	if len(cmd.Args) < 2 {
		result.Error = ErrInvalidArguments
		return result
	}

	subcommand := strings.ToUpper(cmd.Args[0])

	switch subcommand {
	case "CREATE":
		return handleXGroupCreate(cmd)
	case "DESTROY":
		return handleXGroupDestroy(cmd)
	case "DELCONSUMER":
		return handleXGroupDelConsumer(cmd)
	case "SETID":
		return handleXGroupSetID(cmd)
	default:
		result.Error = ErrInvalidArguments
		return result
	}
}

func handleXGroupCreate(cmd *connection.Command) *ExecutionResult {
	result := NewExecutionResult()

	// XGROUP CREATE key group id [MKSTREAM]
	if len(cmd.Args) < 4 {
		result.Error = ErrInvalidArguments
		return result
	}

	streamKey := cmd.Args[1]
	groupName := cmd.Args[2]
	startID := cmd.Args[3]

	// Check for optional MKSTREAM flag
	mkstream := false
	if len(cmd.Args) > 4 && strings.ToUpper(cmd.Args[4]) == "MKSTREAM" {
		mkstream = true
	}

	db := store.GetInstance()
	stream, loaded, err := db.LoadOrStoreStream(streamKey)
	if err != nil {
		result.Error = err
		return result
	}

	// If stream doesn't exist and MKSTREAM not specified, return error
	if !loaded && !mkstream {
		result.Error = ErrInvalidArguments
		return result
	}

	// Create the consumer group
	err = stream.CreateGroup(groupName, startID)
	if err != nil {
		result.Error = err
		return result
	}

	result.Response = utils.ToSimpleString("OK")
	result.ModifiedKeys = []string{streamKey}
	return result
}

func handleXGroupDestroy(cmd *connection.Command) *ExecutionResult {
	result := NewExecutionResult()

	// XGROUP DESTROY key group
	if len(cmd.Args) < 3 {
		result.Error = ErrInvalidArguments
		return result
	}

	streamKey := cmd.Args[1]
	groupName := cmd.Args[2]

	db := store.GetInstance()
	stream, ok := db.GetStream(streamKey)
	if !ok {
		result.Error = ErrInvalidArguments
		return result
	}

	err := stream.DestroyGroup(groupName)
	if err != nil {
		result.Error = err
		return result
	}

	result.Response = utils.ToRespInt(1)
	result.ModifiedKeys = []string{streamKey}
	return result
}

func handleXGroupDelConsumer(cmd *connection.Command) *ExecutionResult {
	result := NewExecutionResult()

	// XGROUP DELCONSUMER key group consumer
	if len(cmd.Args) < 4 {
		result.Error = ErrInvalidArguments
		return result
	}

	streamKey := cmd.Args[1]
	groupName := cmd.Args[2]
	consumerName := cmd.Args[3]

	db := store.GetInstance()
	stream, ok := db.GetStream(streamKey)
	if !ok {
		result.Error = ErrInvalidArguments
		return result
	}

	pendingCount, err := stream.DeleteConsumer(groupName, consumerName)
	if err != nil {
		result.Error = err
		return result
	}

	result.Response = utils.ToRespInt(pendingCount)
	result.ModifiedKeys = []string{streamKey}
	return result
}

func handleXGroupSetID(cmd *connection.Command) *ExecutionResult {
	result := NewExecutionResult()

	// XGROUP SETID key group id
	if len(cmd.Args) < 4 {
		result.Error = ErrInvalidArguments
		return result
	}

	streamKey := cmd.Args[1]
	groupName := cmd.Args[2]
	newID := cmd.Args[3]

	db := store.GetInstance()
	stream, ok := db.GetStream(streamKey)
	if !ok {
		result.Error = ErrInvalidArguments
		return result
	}

	err := stream.SetGroupID(groupName, newID)
	if err != nil {
		result.Error = err
		return result
	}

	result.Response = utils.ToSimpleString("OK")
	result.ModifiedKeys = []string{streamKey}
	return result
}
