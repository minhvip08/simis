package server

import (
	"errors"
	"io"
	"strings"

	"github.com/minhvip08/simis/app/command"
	"github.com/minhvip08/simis/app/config"
	"github.com/minhvip08/simis/app/connection"
	"github.com/minhvip08/simis/app/constants"
	"github.com/minhvip08/simis/app/logger"
	"github.com/minhvip08/simis/app/utils"
)

func Handle(conn *connection.RedisConnection) {
	HandleInitialData(conn, []byte{})
}

func HandleInitialData(conn *connection.RedisConnection, initialData []byte) {
	defer conn.Close()

	buf := make([]byte, constants.DefaultBufferSize)

	// accumulate data efficiently with strings.Builder
	var builder strings.Builder
	if len(initialData) > 0 {
		builder.Write(initialData)
		leftover := HandleRequests(conn, builder.String())
		resetBuilderWithLeftover(&builder, leftover)
	}

	for {
		n, err := conn.Read(buf)
		if err != nil {
			if errors.Is(err, io.EOF) {
				logger.Info("Connection closed by client: %s", conn.GetAddress())
				break
			}
			logger.Error("Error reading from connection %s: %v", conn.GetAddress(), err)
			break
		}

		if n == 0 {
			continue
		}

		builder.Write(buf[:n])

		// Handle complete requests and get leftover data
		leftover := HandleRequests(conn, builder.String())

		// Reset builder and write leftover for next iteration
		resetBuilderWithLeftover(&builder, leftover)
	}

}

func resetBuilderWithLeftover(builder *strings.Builder, leftover string) {
	builder.Reset()
	builder.WriteString(leftover)
}

// HandleRequests processes the incoming data and returns any leftover data that couldn't be processed
func HandleRequests(conn *connection.RedisConnection, buffer string) string {
	commands, remaining, err := command.ParseMultipleRequests(buffer)
	if err != nil {
		logger.Error("Error parsing requests from %s: %v", conn.GetAddress(), err)

	}

	for _, cmd := range commands {
		logger.Debug("Processing command", "role", config.GetInstance().Role, "command", cmd.Name, "args", cmd.Args)
		Route(conn, cmd.Name, cmd.Args)

		cmdArray := []string{cmd.Name}
		cmdArray = append(cmdArray, cmd.Args...)
		respCommand := utils.ToArray(cmdArray)

		if conn.IsMaster() {
			cfg := config.GetInstance()
			newOffset := cfg.AddOffset(len(respCommand))
			logger.Debug("Received command from master",
				"bytes", len(respCommand), "command", cmd.Name, "args", cmd.Args, "offset", newOffset)
		}
	}

	return remaining // Temporarily return all as leftover
}
