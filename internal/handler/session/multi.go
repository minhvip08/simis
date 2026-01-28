package session

import "github.com/minhvip08/simis/internal/connection"

type MultiHandler struct{}

func (h *MultiHandler) Execute(cmd *connection.Command, conn *connection.RedisConnection) error {
	resp, multiErr := conn.StartTransaction()
	if multiErr != nil {
		return multiErr
	}
	conn.SendResponse(resp)
	return nil
}
