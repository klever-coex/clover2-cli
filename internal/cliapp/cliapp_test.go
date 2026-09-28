package cliapp

import (
	"bytes"
	"errors"
	"testing"

	"github.com/spf13/cobra"
)

// ExitCodeOf maps an error to the exit code main() would use.
func ExitCodeOf(t *testing.T, err error) ExitCode {
	t.Helper()
	var coded *Error
	if errors.As(err, &coded) {
		return coded.Code
	}
	if err != nil {
		return ExitGeneric
	}
	return ExitSuccess
}

// tableResult is a Result with a non-flat text rendering, to exercise the
// interface path (Payload covers the default).
type tableResult struct{}

func (tableResult) Text() string { return "NAME  VER\na     1.0.0" }
func (tableResult) Data() any    { return map[string]any{"version": "1.0.0", "in_sync": true} }

func runEmit(t *testing.T, res Result, args ...string) (string, error) {
	t.Helper()
	var buf bytes.Buffer
	old := out
	out = &buf
	t.Cleanup(func() { out = old })

	cmd := &cobra.Command{
		Use:  "t",
		RunE: func(c *cobra.Command, _ []string) error { return Emit(c, res) },
	}
	PayloadFlags(cmd)
	cmd.SilenceUsage = true
	cmd.SetArgs(args)
	err := cmd.Execute()
	return buf.String(), err
}

func TestEmitPayload(t *testing.T) {
	p := Payload{"tag": "v1.2.3", "base": "1.2.3"}

	// default: sorted key: value lines
	got, err := runEmit(t, p)
	if err != nil {
		t.Fatal(err)
	}
	if got != "base: 1.2.3\ntag: v1.2.3\n" {
		t.Errorf("text = %q", got)
	}

	// --json: one compact line
	got, err = runEmit(t, p, "--json")
	if err != nil {
		t.Fatal(err)
	}
	if got != `{"base":"1.2.3","tag":"v1.2.3"}`+"\n" {
		t.Errorf("json = %q", got)
	}

	// --field
	got, err = runEmit(t, p, "--field", "tag")
	if err != nil {
		t.Fatal(err)
	}
	if got != "v1.2.3\n" {
		t.Errorf("field = %q", got)
	}

	// unknown --field: coded error
	_, err = runEmit(t, p, "--field", "nope")
	if got := ExitCodeOf(t, err); got != ExitBadArgument {
		t.Errorf("unknown field code = %v", got)
	}

	// --json + --field: coded error
	_, err = runEmit(t, p, "--json", "--field", "tag")
	if got := ExitCodeOf(t, err); got != ExitBadArgument {
		t.Errorf("mutually exclusive code = %v", got)
	}
}

func TestEmitTypedResult(t *testing.T) {
	got, err := runEmit(t, tableResult{})
	if err != nil || got != "NAME  VER\na     1.0.0\n" {
		t.Errorf("text = %q, err = %v", got, err)
	}
	got, err = runEmit(t, tableResult{}, "--field", "in_sync")
	if err != nil || got != "true\n" {
		t.Errorf("field on typed Data = %q, err = %v", got, err)
	}
}
