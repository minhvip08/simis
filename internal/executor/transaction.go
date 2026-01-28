package executor

import (
	"github.com/minhvip08/simis/internal/command"
	"github.com/minhvip08/simis/internal/handler"
	"github.com/minhvip08/simis/internal/logger"
	"github.com/minhvip08/simis/internal/utils"
)

type TransactionExecutor struct {
	blockingMgr *BlockingCommandManager
}

func NewTransactionExecutor(blockingMgr *BlockingCommandManager) *TransactionExecutor {
	return &TransactionExecutor{
		blockingMgr: blockingMgr,
	}
}

// Execute executes a transaction and returns the response
func (te *TransactionExecutor) Execute(trans *command.Transaction) {
	result := make([]string, len(trans.Commands))
	unblockedKeys := make([]string, 0)

	for i, cmd := range trans.Commands {
		cmdHandler, ok := handler.Handlers[cmd.Command]
		if !ok {
			logger.Warn("Unknown command in transaction", "command", cmd.Command)
			result[i] = utils.ToError("unknown command: " + cmd.Command)
			continue
		}

		exec := cmdHandler.Execute(&cmd)
		if exec.BlockingTimeout >= 0 { // we assume that the transaction has no command that blocks
			panic("transaction blocked")
		}
		unblockedKeys = append(unblockedKeys, exec.ModifiedKeys...)
		if exec.Error != nil {
			result[i] = utils.ToError(exec.Error.Error())
			continue
		}

		result[i] = exec.Response
	}

	te.blockingMgr.UnblockCommandsWaitingForKey(unblockedKeys)
	trans.Response <- utils.ToSimpleRespArray(result)
}
