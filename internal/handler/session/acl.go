package session

import (
	"crypto/sha256"
	"encoding/hex"

	"github.com/minhvip08/simis/internal/acl"
	"github.com/minhvip08/simis/internal/connection"
	err "github.com/minhvip08/simis/internal/error"
	"github.com/minhvip08/simis/internal/utils"
)

type ACLHandler struct{}

func getUserInfo(username string) (string, error) {
	manager := acl.GetInstance()

	// Get user info for the specified username (or current user if not specified)
	user, exists := manager.GetUser(username)
	if !exists {
		return "", err.ErrUserNotFound
	}

	// Build response array with user info
	// Format: ["flags", ["on", "allkeys"], "passwords", [...], ...]
	response := []string{
		utils.ToBulkString("flags"),
		utils.ToRespArray(user.Flags),
		utils.ToBulkString("passwords"),
		utils.ToRespArray(user.Passwords),
	}

	if len(user.Keys) > 0 {
		response = append(response, utils.ToBulkString("keys"))
		response = append(response, utils.ToRespArray(user.Keys))
	}

	if len(user.Commands) > 0 {
		response = append(response, utils.ToBulkString("commands"))
		response = append(response, utils.ToRespArray(user.Commands))
	}

	if len(user.Channels) > 0 {
		response = append(response, utils.ToBulkString("channels"))
		response = append(response, utils.ToRespArray(user.Channels))
	}

	return utils.ToSimpleRespArray(response), nil
}

func setUserPassword(username string, password string) (string, error) {
	manager := acl.GetInstance()
	user, exists := manager.GetUser(username)
	if !exists {
		return "", err.ErrUserNotFound
	}

	hash := sha256.Sum256([]byte(password))
	user.Passwords = append(user.Passwords, hex.EncodeToString(hash[:]))
	// Remove "nopass" flag if it exists
	newFlags := []string{}
	for _, flag := range user.Flags {
		if flag != "nopass" {
			newFlags = append(newFlags, flag)
		}
	}
	user.Flags = newFlags
	manager.SetUser(user)
	return utils.ToSimpleString("OK"), nil
}

func (h *ACLHandler) Execute(cmd *connection.Command, conn *connection.RedisConnection) error {
	if len(cmd.Args) < 1 {
		return err.ErrInvalidArguments
	}

	subcommand := cmd.Args[0]
	username := conn.GetUsername()

	switch subcommand {
	case "WHOAMI":
		// Return the username of the connection that issued this command
		conn.SendResponse(utils.ToBulkString(username))
		return nil

	case "GETUSER":
		info, err := getUserInfo(username)
		if err != nil {
			return err
		}
		conn.SendResponse(info)
		return nil

	case "SETUSER":
		username = cmd.Args[1]
		password := cmd.Args[2][1:]
		info, err := setUserPassword(username, password)
		if err != nil {
			return err
		}
		conn.SendResponse(info)
		return nil
	}

	conn.SendError("Unknown ACL subcommand: " + subcommand)
	return nil
}
