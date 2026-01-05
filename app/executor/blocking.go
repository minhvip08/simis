package executor

import "github.com/minhvip08/simis/app/connection"

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
