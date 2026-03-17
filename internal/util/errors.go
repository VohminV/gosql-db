package util

import "fmt"

// ErrorType определяет категорию ошибки.
type ErrorType string

const (
	ErrIO          ErrorType = "IO_ERROR"
	ErrTransaction ErrorType = "TRANSACTION_ERROR"
	ErrSyntax      ErrorType = "SYNTAX_ERROR"
	ErrInternal    ErrorType = "INTERNAL_ERROR"
)

// DBError представляет структурированную ошибку базы данных.
type DBError struct {
	Type    ErrorType
	Message string
	Cause   error
}

func (e *DBError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

func NewError(typ ErrorType, msg string, cause error) *DBError {
	return &DBError{Type: typ, Message: msg, Cause: cause}
}