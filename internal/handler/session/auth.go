package session

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"slices"

	"github.com/minhvip08/simis/internal/acl"
	"github.com/minhvip08/simis/internal/connection"
	err "github.com/minhvip08/simis/internal/error"
	"github.com/minhvip08/simis/internal/utils"
)

type AuthHandler struct{}

func (h *AuthHandler) Execute(cmd *connection.Command, conn *connection.RedisConnection) error {
	if len(cmd.Args) < 1 {
		return err.ErrInvalidArguments
	}
	username := cmd.Args[0]
	password := cmd.Args[1]
	user, exists := acl.GetInstance().GetUser(username)
	if !exists {
		conn.SendResponse(fmt.Sprintf("-%s\r\n", err.ErrInvalidUserPassword.Error()))
		return nil
	}
	hash := sha256.Sum256([]byte(password))
	if !slices.Contains(user.Passwords, hex.EncodeToString(hash[:])) {
		conn.SendResponse(fmt.Sprintf("-%s\r\n", err.ErrInvalidUserPassword.Error()))
		return nil
	}

	conn.SetAuthenticated(true)
	conn.SetUsername(username)
	conn.SendResponse(utils.ToSimpleString("OK"))
	return nil
}
