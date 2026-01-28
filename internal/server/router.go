package server

import (
	"fmt"

	"github.com/minhvip08/simis/internal/command"
	"github.com/minhvip08/simis/internal/connection"
	"github.com/minhvip08/simis/internal/constants"
	"github.com/minhvip08/simis/internal/handler/session"
	"github.com/minhvip08/simis/internal/utils"
)

func Route(conn *connection.RedisConnection, cmdName string, args []string) {
	// TODO: Check Authentication

	if conn.IsInTransaction() && !isTransactionControlCommand(cmdName) {
		queueCommand(conn, cmdName, args)
		return
	}

	if conn.IsInPubSubMode() && !isPubSubCommand(cmdName) {
		conn.SendError(fmt.Sprintf("Can't execute '%s': only (P|S)SUBSCRIBE / (P|S)UNSUBSCRIBE / PING / QUIT / RESET are allowed in this context", cmdName))
		return
	}

	// Special case: PING in pubsub mode returns an array
	if cmdName == "PING" && conn.IsInPubSubMode() {
		conn.SendResponse(utils.ToRespArray([]string{"pong", ""}))
		return
	}

	// TODO Session Level
	if handler, exists := session.SessionHandlers[cmdName]; exists {
		// Create command object for the handler
		cmd := connection.CreateCommand(cmdName, args...)

		// Execute the session handler
		handlerErr := handler.Execute(&cmd, conn)
		if handlerErr != nil {
			conn.SendError(handlerErr.Error())
		}
		// Note: Session handlers send responses directly via conn.SendResponse()
		// so we don't need to handle the response return value
		return
	}

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

func isTransactionControlCommand(cmdName string) bool {
	return cmdName == "EXEC" || cmdName == "DISCARD"
}

func queueCommand(conn *connection.RedisConnection, cmdName string, args []string) {
	queued, queueErr := conn.EnqueueCommand(command.CreateCommand(cmdName, args))
	if queueErr != nil {
		conn.SendError(queueErr.Error())
		return
	}
	if queued {
		conn.SendResponse(utils.ToSimpleString("QUEUED"))
	}
}
