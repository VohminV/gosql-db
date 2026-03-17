package client

import "fmt"

// ErrorType определяет категорию ошибки на стороне клиента.
type ErrorType string

const (
	ErrConnection ErrorType = "CONNECTION_ERROR"
	ErrProtocol   ErrorType = "PROTOCOL_ERROR"
	ErrServer     ErrorType = "SERVER_ERROR"
	ErrTimeout    ErrorType = "TIMEOUT_ERROR"
)

// ClientError представляет структурированную ошибку клиента.
type ClientError struct {
	Type    ErrorType
	Message string
	Cause   error
}

func (e *ClientError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %v", e.Type, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s", e.Type, e.Message)
}

// NewConnectionError создает ошибку соединения.
func NewConnectionError(msg string, cause error) *ClientError {
	return &ClientError{Type: ErrConnection, Message: msg, Cause: cause}
}

// NewServerError создает ошибку, полученную от сервера.
func NewServerError(msg string) *ClientError {
	return &ClientError{Type: ErrServer, Message: msg}
}

// NewProtocolError создает ошибку протокола.
func NewProtocolError(msg string, cause error) *ClientError {
	return &ClientError{Type: ErrProtocol, Message: msg, Cause: cause}
}