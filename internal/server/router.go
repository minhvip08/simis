package server

import (
	"fmt"

	"github.com/minhvip08/simis/internal/command"
	"github.com/minhvip08/simis/internal/connection"
	"github.com/minhvip08/simis/internal/constants"
	"github.com/minhvip08/simis/internal/utils"
)

func Route(conn *connection.RedisConnection, cmdName string, args []string) {
	// TODO: Check Authentication

	// TODO: Transaction Handling

	if conn.IsInPubSubMode() && !isPubSubCommand(cmdName) {
		conn.SendError(fmt.Sprintf("Can't execute '%s': only (P|S)SUBSCRIBE / (P|S)UNSUBSCRIBE / PING / QUIT / RESET are allowed in this context", cmdName))
		return
	}

	// Special case: PING in pubsub mode returns an array
	if cmdName == "PING" && conn.IsInPubSubMode() {
		conn.SendResponse(utils.ToArray([]string{"pong", ""}))
		return
	}

	// TODO Session Level

	// Execute Regular data command
	executeDataCommand(conn, cmdName, args)
}

// isPubSubCommand checks if a command is a pubsub command
func isPubSubCommand(cmdName string) bool {
	return constants.PubSubCommands[cmdName]
}

// execute Regular data command
func executeDataCommand(conn *connection.RedisConnection, cmdName string, args []string) {
	commandObj := connection.CreateCommand(cmdName, args...)
	command.GetQueueInstance().EnqueueCommand(&commandObj)
	response := <-commandObj.Response
	conn.SendResponse(response)
}
