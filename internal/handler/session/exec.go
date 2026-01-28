package session

import (
	"github.com/minhvip08/simis/internal/command"
	"github.com/minhvip08/simis/internal/connection"
	err "github.com/minhvip08/simis/internal/error"
)

type ExecHandler struct{}

func (h *ExecHandler) Execute(cmd *connection.Command, conn *connection.RedisConnection) error {
	if !conn.IsInTransaction() {
		return err.ErrExecWithoutMulti
	}

	conn.EndTransaction()
	transaction := command.Transaction{
		Commands: conn.QueuedCommands(),
		Response: make(chan string),
	}
	command.GetQueueInstance().EnqueueTransaction(&transaction)
	response := <-transaction.Response
	conn.SendResponse(response)
	return nil
}
