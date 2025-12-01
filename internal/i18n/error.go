package i18n

import (
	"fmt"
)

var (
	ErrInvalidEmojiFormat   = newError("invalid emoji format: %s")
	ErrUnknownComponentType = newError("unknown component type: %s")
	ErrDuplicateLocale      = newError("duplicate locale: %s")
	ErrUnknownLocale        = newError("unknown locale: %s")
	ErrInvalidButtonStyle   = newError("invalid button style: %s")
)

type Error struct {
	Message string `json:"message"`
	Args    []any  `json:"args,omitempty"`
}

var _ error = (*Error)(nil)

func (e *Error) Error() string {
	return fmt.Sprintf(e.Message, e.Args...)
}

func (e *Error) Is(target error) bool {
	if targetErr, ok := target.(*Error); ok {
		return e.Message == targetErr.Message
	}
	return false
}

func (e *Error) Format(args ...any) error {
	if len(args) == 0 {
		return e
	}
	return &Error{Message: e.Message, Args: append(e.Args, args...)}
}

func newError(message string, args ...any) *Error {
	if len(args) == 0 {
		return &Error{Message: message}
	}
	return &Error{Message: message, Args: args}
}
