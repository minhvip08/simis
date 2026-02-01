package session

import (
	"strconv"

	"github.com/minhvip08/simis/internal/connection"
	errs "github.com/minhvip08/simis/internal/error"
	"github.com/minhvip08/simis/internal/replication"
	"github.com/minhvip08/simis/internal/utils"
)

type WaitHandler struct{}

type waitParams struct {
	numreplicas int
	timeout     int
}

func parseWait(args []string) (*waitParams, error) {
	if len(args) != 2 {
		return nil, errs.ErrInvalidArguments
	}

	numreplicas, err := strconv.Atoi(args[0])
	if err != nil || numreplicas < 0 {
		return nil, errs.ErrInvalidArguments
	}

	timeout, err := strconv.Atoi(args[1])
	if err != nil || timeout < 0 {
		return nil, errs.ErrInvalidArguments
	}

	return &waitParams{
		numreplicas: numreplicas,
		timeout:     timeout,
	}, nil
}

func executeWait(params *waitParams) (string, error) {
	updatedReplicas := replication.GetManager().WaitForReplicas(params.numreplicas, params.timeout)
	// return strconv.Itoa(updatedReplicas), nil
	return utils.ToRespInt(updatedReplicas), nil
}

func (h *WaitHandler) Execute(cmd *connection.Command, conn *connection.RedisConnection) error {
	params, parseErr := parseWait(cmd.Args)
	if parseErr != nil {
		return parseErr
	}

	result, execErr := executeWait(params)
	if execErr != nil {
		return execErr
	}

	conn.SendResponse(result)
	return nil
}
