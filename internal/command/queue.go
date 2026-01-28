package command

import (
	"sync"

	"github.com/minhvip08/simis/internal/connection"
	"github.com/minhvip08/simis/internal/constants"
)

var (
	instance *Queue
	once     sync.Once
)

type Queue struct {
	cmdQueue     chan *connection.Command
	transactions chan *Transaction
}

type Transaction struct {
	Commands []connection.Command
	Response chan string
}

func GetQueueInstance() *Queue {
	once.Do(func() {
		instance = &Queue{
			cmdQueue:     make(chan *connection.Command, constants.DefaultCommandQueueSize),
			transactions: make(chan *Transaction, constants.DefaultTransactionQueueSize),
		}
	})
	return instance
}

// EnqueueCommand adds a command to the command queue
func (q *Queue) EnqueueCommand(cmd *connection.Command) {
	q.cmdQueue <- cmd
}

// EnqueueTransaction adds a transaction to the transaction queue
func (q *Queue) EnqueueTransaction(trans *Transaction) {
	q.transactions <- trans
}

// CommandQueue returns the command channel
func (q *Queue) CommandQueue() <-chan *connection.Command {
	return q.cmdQueue
}

// TransactionQueue returns the transaction channel
func (q *Queue) TransactionQueue() <-chan *Transaction {
	return q.transactions
}

func CreateCommand(command string, args []string) connection.Command {
	return connection.CreateCommand(command, args...)
}
