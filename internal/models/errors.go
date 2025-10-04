package models

import "errors"

var (
	ErrDuplicateUser      = errors.New("username already exists")
	ErrInvalidCredentials = errors.New("invalid username or password")
)
