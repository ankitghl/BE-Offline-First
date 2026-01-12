package domain

import "errors"

var ErrNotFound = errors.New("not found")

var ErrAlreadyExists = errors.New("item already exists")

type ConflictError struct {
	ServerItem *Item
}

func NewConflictError(item *Item) *ConflictError {
	return &ConflictError{ServerItem: item}
}

func (e *ConflictError) Error() string {
	return "version conflict"
}
