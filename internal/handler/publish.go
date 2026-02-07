package handler

import (
	"github.com/minhvip08/simis/internal/connection"
	"github.com/minhvip08/simis/internal/pubsub"
	"github.com/minhvip08/simis/internal/utils"
)

type PublishHandler struct{}

type publishParams struct {
	channel string
	message string
}

func parsePublish(args []string) (*publishParams, error) {
	if len(args) < 2 {
		return nil, ErrInvalidArguments
	}
	return &publishParams{channel: args[0], message: args[1]}, nil
}

func executePublish(params *publishParams) *ExecutionResult {
	pubsubManager := pubsub.GetPubSubManager()
	subscribers := pubsubManager.Publish(params.channel, params.message)

	result := NewExecutionResult()
	result.Response = utils.ToRespInt(subscribers)
	return result
}

func (h *PublishHandler) Execute(cmd *connection.Command) *ExecutionResult {
	params, err := parsePublish(cmd.Args)
	if err != nil {
		return NewErrorResult(err)
	}
	return executePublish(params)
}
