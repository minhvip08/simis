package session

import (
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/minhvip08/simis/internal/config"
	"github.com/minhvip08/simis/internal/connection"
	err "github.com/minhvip08/simis/internal/error"
	"github.com/minhvip08/simis/internal/logger"
	"github.com/minhvip08/simis/internal/replication"
	"github.com/minhvip08/simis/internal/utils"
)

type PSyncHandler struct{}

type psyncParams struct {
	replicationID string
	offset        int
}

func parsePsync(args []string) (*psyncParams, error) {
	if len(args) < 2 {
		return nil, err.ErrInvalidArguments
	}

	offset, parseErr := strconv.Atoi(args[1])
	if parseErr != nil {
		return nil, parseErr
	}

	return &psyncParams{
		replicationID: args[0],
		offset:        offset,
	}, nil
}

func executePsync(params *psyncParams, conn *connection.RedisConnection) error {
	cfg := config.GetInstance()
	if params.replicationID != cfg.ReplicationID {
		conn.SendResponse(utils.ToSimpleString("FULLRESYNC " + cfg.ReplicationID + " 0"))

		// Get RDB file path relative to working directory
		rdbPath := filepath.Join("data", "dump.rdb")
		hexData, readErr := os.ReadFile(rdbPath)
		if readErr != nil {
			logger.Error("Failed to read RDB file", "path", rdbPath, "error", readErr)
			return readErr
		}
		// Decode hex string to bytes
		file, decodeErr := hex.DecodeString(string(hexData))
		if decodeErr != nil {
			return decodeErr
		}
		conn.SendResponse(fmt.Sprintf("$%d\r\n%s", len(file), file))

		// Mark the handshake as complete for this replica
		manager := replication.GetManager()
		replicas := manager.GetAllReplicas()
		// Find the replica with this connection and mark handshake as done
		for _, replica := range replicas {
			if replica.Connection == conn {
				replica.SetHandshakeDone()
				break
			}
		}

		return nil
	}

	conn.SendResponse(utils.ToSimpleString("OK"))
	return nil
}

func (h *PSyncHandler) Execute(cmd *connection.Command, conn *connection.RedisConnection) error {
	params, parseErr := parsePsync(cmd.Args)
	if parseErr != nil {
		return parseErr
	}

	return executePsync(params, conn)
}
