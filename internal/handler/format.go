package handler

import "errors"

var (
	ErrInvalidArguments = errors.New("Invalid arguments")
	ErrTimeout          = errors.New("Timeout")
	ErrUserNotFound     = errors.New("User not found")
)

func ToEmptyArray() string {
	return "*0\r\n"
}