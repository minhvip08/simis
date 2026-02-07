package session

import (
	"github.com/minhvip08/simis/internal/connection"
	err "github.com/minhvip08/simis/internal/error"
	"github.com/minhvip08/simis/internal/pubsub"
	"github.com/minhvip08/simis/internal/utils"
)

type UnsubscribeHandler struct{}

func (h *UnsubscribeHandler) Execute(cmd *connection.Command, conn *connection.RedisConnection) error {
	if len(cmd.Args) < 1 {
		return err.ErrInvalidArguments
	}
	channel := cmd.Args[0]
	pubsubManager := pubsub.GetPubSubManager()
	channels := pubsubManager.Unsubscribe(conn, channel)
	conn.SendResponse(
		utils.ToSimpleRespArray([]string{
			utils.ToBulkString("unsubscribe"),
			utils.ToBulkString(channel),
			utils.ToRespInt(channels),
		}),
	)
	return nil
}
