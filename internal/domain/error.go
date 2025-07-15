package domain

type ErrCode int

const (
	ErrCodeArchiveNotFound ErrCode = iota
)

type Error struct {
	code ErrCode
	msg  string
}

func (e *Error) Error() string {
	return e.msg
}

func (e *Error) Code() ErrCode {
	return e.code
}

func NewError(code ErrCode, msg string) *Error {
	return &Error{code, msg}
}
