package proxy

import (
	"fmt"

	"github.com/emersion/go-smtp"
)

func NewNotMemberError(email string) error {
	return &smtp.SMTPError{
		Code:         553,
		EnhancedCode: smtp.EnhancedCode{5, 3, 0},
		Message:      fmt.Sprintf("<%s>... User unknown, not local address.", email),
		ForceClose:   true,
	}
}

func NewAuthorizationError(email string) error {
	return &smtp.SMTPError{
		Code:         550,
		EnhancedCode: smtp.EnhancedCode{5, 5, 0},
		Message:      fmt.Sprintf("can't found mail server of <%s>.", email),
	}
}

func NewNotFoundError(email string) error {
	return &smtp.SMTPError{
		Code:         550,
		EnhancedCode: smtp.EnhancedCode{5, 5, 0},
		Message:      fmt.Sprintf("can't found mail server of <%s>.", email),
	}
}

func NewBadCommandError() error {
	return &smtp.SMTPError{
		Code:         503,
		EnhancedCode: smtp.EnhancedCode{5, 0, 3},
		Message:      "Bad sequence of commands",
	}
}

func NewError(err error) error {
	return &smtp.SMTPError{
		Code:         451,
		EnhancedCode: smtp.EnhancedCode{4, 5, 1},
		Message:      fmt.Sprintf("%s", err.Error()),
	}
}
