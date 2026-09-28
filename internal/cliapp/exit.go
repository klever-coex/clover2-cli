package cliapp

import "fmt"

type ExitCode int

const (
	ExitSuccess     ExitCode = 0
	ExitGeneric     ExitCode = 1
	ExitNoProject   ExitCode = 3 // no/invalid tooling/tooling.json
	ExitBadArgument ExitCode = 4 // invalid flag, field or command argument
	ExitNetwork     ExitCode = 5 // network/storage failure (artifact push/pull, robots unreachable)
)

type Error struct {
	Code ExitCode
	Err  error
}

func (e *Error) Error() string { return e.Err.Error() }
func (e *Error) Unwrap() error { return e.Err }

func ExitErrorf(code ExitCode, format string, args ...any) *Error {
	return &Error{Code: code, Err: fmt.Errorf(format, args...)}
}
