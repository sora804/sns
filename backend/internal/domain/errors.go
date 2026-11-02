package domain

import "errors"

// 業務エラー。handler で HTTP ステータスに変換する。
var (
	ErrNotFound     = errors.New("not found")
	ErrForbidden    = errors.New("forbidden")
	ErrUnauthorized = errors.New("unauthorized")
	ErrConflict     = errors.New("conflict")
	ErrValidation   = errors.New("validation error")
)

// ValidationError は入力値が不正なときのエラー。errors.Is(err, ErrValidation) で判定できる。
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string { return e.Field + ": " + e.Message }

func (e *ValidationError) Is(target error) bool { return target == ErrValidation }

func invalid(field, message string) error {
	return &ValidationError{Field: field, Message: message}
}
