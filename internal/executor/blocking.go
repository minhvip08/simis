package executor

import (
	"time"

	"github.com/minhvip08/simis/internal/connection"
	"github.com/minhvip08/simis/internal/handler"
	"github.com/minhvip08/simis/internal/logger"
	"github.com/minhvip08/simis/internal/utils"
)

type WaitingCommand struct {
	Command     *connection.Command
	WaitingKeys []string
}

type BlockingCommandManager struct {
	blockingCommands map[string][]*WaitingCommand
}

func NewBlockingCommandManager() *BlockingCommandManager {
	return &BlockingCommandManager{
		blockingCommands: make(map[string][]*WaitingCommand),
	}
}

func (bcm *BlockingCommandManager) AddWaitingCommand(timeout int, cmd *connection.Command, waitingKeys []string) {
	wc := &WaitingCommand{
		Command:     cmd,
		WaitingKeys: waitingKeys,
	}

	for _, key := range waitingKeys {
		bcm.blockingCommands[key] = append(bcm.blockingCommands[key], wc)
	}

	// Remove the command if timeout occurs
	if timeout > 0 {
		go func(wc *WaitingCommand) {
			time.Sleep(time.Duration(timeout) * time.Millisecond)

			for _, key := range waitingKeys {
				bcm.blockingCommands[key] = bcm.removeWaitingCommand(bcm.blockingCommands[key], wc)
			}
			wc.Command.Response <- utils.ToRespArray(nil)
		}(wc)
	}
}

// UnblockCommandsWaitingOnKey unblocks all commands waiting on the given key
func (bcm *BlockingCommandManager) UnblockCommandsWaitingForKey(keys []string) {
	for _, key := range keys {
		blockedCommands := bcm.blockingCommands[key]
		logger.Debug("Unblocking commands", "key", key, "count", len(blockedCommands))
		lastExecutedCommand := -1
		for i, blockedCmd := range blockedCommands {
			command := blockedCmd.Command
			logger.Debug("Executing blocked command", "command", command.Command, "args", command.Args)
			cmdHandler, ok := handler.Handlers[command.Command]
			if !ok {
				logger.Warn("Unknown command in blocked queue", "command", command.Command)
				command.Response <- utils.ToError("unknown command: " + command.Command)
				continue
			}
			// Here we assume that the command won't unblock another command (causing a chain reaction)
			// and so we ignore the keys that were set in its execution
			executionResult := cmdHandler.Execute(command)
			logger.Debug("Blocked command executed", "command", command.Command, "timeout", executionResult.BlockingTimeout, "error", executionResult.Error)
			if executionResult.BlockingTimeout >= 0 {
				break
			}

			// remove the waiting command from the blocking commands map
			waitingKeys := blockedCmd.WaitingKeys
			for _, waitingKey := range waitingKeys {
				bcm.blockingCommands[waitingKey] = bcm.removeWaitingCommand(bcm.blockingCommands[waitingKey], blockedCmd)
			}

			lastExecutedCommand = i
			if executionResult.Error != nil {
				command.Response <- utils.ToError(executionResult.Error.Error())
				continue
			}
			command.Response <- executionResult.Response
		}
		bcm.blockingCommands[key] = blockedCommands[lastExecutedCommand+1:]
	}
}

// removeWaitingCommand removes a waiting command from the list
func (bcm *BlockingCommandManager) removeWaitingCommand(waitingCommands []*WaitingCommand, blockedCommand *WaitingCommand) []*WaitingCommand {
	for i, waitingCommand := range waitingCommands {
		if waitingCommand.Command == blockedCommand.Command {
			return append(waitingCommands[:i], waitingCommands[i+1:]...)
		}
	}
	return waitingCommands
}
