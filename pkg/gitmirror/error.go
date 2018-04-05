package gitmirror

import "fmt"

type errCode int

const (
	errAlreadyCloned errCode = iota
)

type gitError struct {
	s    string
	code errCode
}

func newErrorf(code errCode, msg string, args ...interface{}) error {
	return gitError{s: fmt.Sprintf(msg, args...), code: code}
}

func (e gitError) Error() string {
	return e.s
}
