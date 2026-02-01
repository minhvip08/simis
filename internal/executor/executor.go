package executor

import (
	"github.com/minhvip08/simis/internal/command"
	"github.com/minhvip08/simis/internal/connection"
	"github.com/minhvip08/simis/internal/constants"
	"github.com/minhvip08/simis/internal/handler"
	"github.com/minhvip08/simis/internal/logger"
	"github.com/minhvip08/simis/internal/replication"
	"github.com/minhvip08/simis/internal/utils"
)

type Executor struct {
	blockingCommandManager *BlockingCommandManager
	transactionExecutor    *TransactionExecutor
}

func NewExecutor() *Executor {
	blockingMgr := NewBlockingCommandManager()
	transactionExecutor := NewTransactionExecutor(blockingMgr)
	return &Executor{
		blockingCommandManager: blockingMgr,
		transactionExecutor:    transactionExecutor,
	}
}

func (e *Executor) Start() {
	queue := command.GetQueueInstance()
	for {
		select {
		case cmd := <-queue.CommandQueue():
			e.executeCommand(cmd)
		case trans := <-queue.TransactionQueue():
			e.transactionExecutor.Execute(trans)
		}
	}
}

func (e *Executor) executeCommand(cmd *connection.Command) {
	cmdHandler, ok := handler.Handlers[cmd.Command]

	if !ok {
		logger.Warn("Unknown command", "command", cmd.Command)
		cmd.Response <- utils.ToError("unknown command: " + cmd.Command)
		return
	}

	result := cmdHandler.Execute(cmd)

	if result.BlockingTimeout >= 0 {
		e.blockingCommandManager.AddWaitingCommand(result.BlockingTimeout, cmd, result.WaitingKeys)
		return
	}

	e.blockingCommandManager.UnblockCommandsWaitingForKey(result.ModifiedKeys)

	if result.Error != nil {
		cmd.Response <- utils.ToError(result.Error.Error())
		return
	}

	if isWriteCommand(cmd.Command) {
		replication.GetManager().PropagateCommand(cmd.Command, cmd.Args)
	}

	cmd.Response <- result.Response
}

func isWriteCommand(cmdName string) bool {
	return constants.WriteCommands[cmdName]
}
