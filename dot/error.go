package dot

import "fmt"

type Error struct {
	Base error
	Msg  string
	Code string
}

func (e *Error) Error() string {
	if e.Code == "-333" {
		return e.Base.Error() + "/" + e.Msg // 用 / 仅仅是对阿里云日志搜索友好
	}
	return fmt.Sprintf("%s/%s/%s", e.Base.Error(), e.Code, e.Msg)
}

func (e *Error) Is(target error) bool {
	return e.Base == target
}

func (e *Error) GetMsg() string {
	return e.Msg
}

func (e *Error) GetCode() string {
	return e.Code
}

func (e *Error) GetBase() error {
	return e.Base
}

func NewError(base error, msg string) error {
	return &Error{base, msg, "-333"}
}

func NewErrorCode[T any](base error, msg string, code T) error {
	return &Error{base, msg, fmt.Sprintf("%v", code)}
}
