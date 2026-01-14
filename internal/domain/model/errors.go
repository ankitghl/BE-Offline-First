package domain

/*
MutationError is a semantic contract between backend and client.

It answers:
1. Can this error be retried automatically?
2. Is this a conflict that requires user intervention?
*/
type MutationError interface {
	error
	IsRetryable() bool
	IsConflict() bool
}

/*
========================

	Not Found

========================
*/

type notFoundError struct{}

func (e notFoundError) Error() string {
	return "not found"
}

func (e notFoundError) IsRetryable() bool {
	return false
}

func (e notFoundError) IsConflict() bool {
	return false
}

var ErrNotFound = notFoundError{}

/*
========================

	Already Exists

========================
*/
type alreadyExistError struct{}

func (e alreadyExistError) Error() string {
	return "item already exists"
}

func (e alreadyExistError) IsRetryable() bool {
	return false
}

func (e alreadyExistError) IsConflict() bool {
	return false
}

var ErrAlreadyExists = alreadyExistError{}

/*
========================

	Conflict Error

========================
*/
type ConflictError struct {
	ServerItem *Item
}

func NewConflictError(item *Item) *ConflictError {
	return &ConflictError{ServerItem: item}
}

func (e *ConflictError) Error() string {
	return "version conflict"
}

func (e *ConflictError) IsRetryable() bool {
	return false
}

func (e *ConflictError) IsConflict() bool {
	return true
}

/*
========================

	Retryable Error

========================

Used for temporary failures:
- DB timeouts
- network hiccups
- transaction serialization failures

These errors tell the client:
"You can retry this mutation safely."
*/
type retryableError struct {
	err error
}

func (e retryableError) Error() string {
	return e.err.Error()
}

func (e retryableError) IsRetryable() bool {
	return true
}

func (e retryableError) IsConflict() bool {
	return false
}

func Retryable(err error) error {
	if err == nil {
		return nil
	}
	return retryableError{err: err}
}

/*
========================

	Helper

========================
*/
func IsMutationError(err error) (MutationError, bool) {
	me, ok := err.(MutationError)
	return me, ok
}
