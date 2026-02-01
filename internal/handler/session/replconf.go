package session

import (
	"strconv"
	"strings"

	"github.com/minhvip08/simis/internal/config"
	"github.com/minhvip08/simis/internal/connection"
	err "github.com/minhvip08/simis/internal/error"
	"github.com/minhvip08/simis/internal/logger"
	"github.com/minhvip08/simis/internal/replication"
	"github.com/minhvip08/simis/internal/utils"
)

type ReplconfHandler struct{}

type replconfParams struct {
	subcommand    string
	listeningPort int
	capabilities  []string
	offset        int
}

func parseReplconf(args []string) (*replconfParams, error) {
	if len(args) < 2 {
		return nil, err.ErrInvalidArguments
	}

	params := &replconfParams{
		subcommand: strings.ToLower(args[0]),
	}

	switch params.subcommand {
	case "listening-port":
		port, parseErr := strconv.Atoi(args[1])
		if parseErr != nil {
			return nil, parseErr
		}
		params.listeningPort = port

	case "capa":
		remainingArgs := args
		for len(remainingArgs) > 1 {
			if remainingArgs[0] != "capa" {
				return nil, err.ErrInvalidArguments
			}
			params.capabilities = append(params.capabilities, remainingArgs[1])
			remainingArgs = remainingArgs[2:]
		}

	case "getack":
		// No additional parsing needed for getack

	case "ack":
		offset, parseErr := strconv.Atoi(args[1])
		if parseErr != nil {
			return nil, parseErr
		}
		params.offset = offset

	default:
		return nil, err.ErrInvalidArguments
	}

	return params, nil
}

func executeReplconf(params *replconfParams, conn *connection.RedisConnection) error {
	replicaInfo := replication.NewReplicaInfo(conn)
	if existingReplicaInfo := replication.GetManager().GetReplica(conn); existingReplicaInfo != nil {
		replicaInfo = existingReplicaInfo
	} else {
		replication.GetManager().AddReplica(replicaInfo)
	}

	switch params.subcommand {
	case "listening-port":
		replicaInfo.ListeningPort = params.listeningPort

	case "capa":
		for _, capability := range params.capabilities {
			replicaInfo.AddCapability(capability)
		}

	case "getack":
		offset := config.GetInstance().GetOffset()
		logger.Debug("Sending ACK to master", "offset", offset)
		conn.SetSuppressResponse(false)
		conn.SendResponse(utils.ToRespArray(
			[]string{"REPLCONF", "ACK", strconv.Itoa(offset)},
		))
		conn.SetSuppressResponse(true)
		return nil

	case "ack":
		replicaInfo.LastAckOffset = params.offset
		logger.Debug("Replica ACK received", "address", conn.GetAddress(), "offset", params.offset)
		// For ACK we don't send any response back
		return nil
	}

	conn.SendResponse(utils.ToSimpleString("OK"))
	return nil
}

func (h *ReplconfHandler) Execute(cmd *connection.Command, conn *connection.RedisConnection) error {
	params, parseErr := parseReplconf(cmd.Args)
	if parseErr != nil {
		return parseErr
	}

	return executeReplconf(params, conn)
}
