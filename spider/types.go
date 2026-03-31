package spider

import (
	"errors"
)

var (
	ErrEmptyBody      = errors.New("empty")
	ErrUnexpectedBody = errors.New("unexpected")
	ErrIdMismatch     = errors.New("id mismatch")
	ErrDirtyData      = errors.New("dirty data")
	ErrSign           = errors.New("failed to sign")
	ErrBadParams      = errors.New("bad params")
	ErrEmptyData      = errors.New("empty data")
	ErrRequest        = errors.New("request error")
)

type StatusCode int

const (
	DeleteField        = "__DELETE_FIELD"
	ModeDefault    int = 0
	ModeSubSpider  int = 1
	ModeRunForever int = 2
)

const (
	StatusCodeFailed StatusCode = iota
	StatusCodeOk
	StatusCodeSkip
	StatusCodeOnRetry
	StatusCodeUninit
)
