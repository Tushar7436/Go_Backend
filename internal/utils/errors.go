package utils

import "errors"

var (
	ErrInvalidInput = errors.New("invalid input")
	ErrTimeInvalid  = errors.New("invalid time range")
)
