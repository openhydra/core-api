package error

import (
	"fmt"
)

type NotFound struct {
	Code    int
	Message string
}

func (ce *NotFound) Error() string {
	return fmt.Sprintf("Error code: %d, message: %s", ce.Code, ce.Message)
}

func NewNotFound(code int, message string) *NotFound {
	return &NotFound{
		Code:    code,
		Message: message,
	}
}

type Unauthorized struct {
	Code    int
	Message string
}

func (ce *Unauthorized) Error() string {
	return fmt.Sprintf("Error code: %d, message: %s", ce.Code, ce.Message)
}

func NewUnauthorized(code int, message string) *Unauthorized {
	return &Unauthorized{
		Code:    code,
		Message: message,
	}
}
