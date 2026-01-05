package executor

import (
	"github.com/minhvip08/simis/app/command"
	"github.com/minhvip08/simis/app/connection"
	"github.com/minhvip08/simis/app/handler"
	"github.com/minhvip08/simis/app/logger"
	"github.com/minhvip08/simis/app/utils"
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
			// TODO: handle transactions
			// case trans := <-queue.TransactionQueue():
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

	if result.Error != nil {
		cmd.Response <- utils.ToError(result.Error.Error())
		return
	}

	cmd.Response <- result.Response
	// TODO: Handle other commands
}
