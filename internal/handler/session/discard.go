package session

import "github.com/minhvip08/simis/internal/connection"

type DiscardHandler struct{}

func (h *DiscardHandler) Execute(cmd *connection.Command, conn *connection.RedisConnection) error {
	response, discardErr := conn.DiscardTransaction()
	if discardErr != nil {
		return discardErr
	}
	conn.SendResponse(response)
	return nil
}
