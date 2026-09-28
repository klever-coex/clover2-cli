package cliapp

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"sort"
	"strings"

	"github.com/spf13/cobra"
)

var Version = "dev"

var RootOverride string

func SetLogLevel(verbose int) {
	level := slog.LevelError

	switch {
	case verbose >= 3:
		level = slog.LevelDebug
	case verbose == 2:
		level = slog.LevelInfo
	case verbose == 1:
		level = slog.LevelWarn
	}

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})))
}

var out io.Writer = os.Stdout

type Result interface {
	Text() string
	Data() any
}

type Payload map[string]any

func (p Payload) Text() string {
	keys := make([]string, 0, len(p))
	for k := range p {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	s := ""
	for _, k := range keys {
		s += fmt.Sprintf("%s: %v\n", k, p[k])
	}

	return strings.TrimSuffix(s, "\n")
}

func (p Payload) Data() any { return p }

func PayloadFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("json", false, "emit machine-readable JSON")
	cmd.Flags().String("field", "", "print a single payload field")
}

func Emit(cmd *cobra.Command, res Result) error {
	asJSON, _ := cmd.Flags().GetBool("json")
	field, _ := cmd.Flags().GetString("field")

	if asJSON && field != "" {
		return ExitErrorf(ExitBadArgument, "--json and --field are mutually exclusive")
	}

	switch {
	case asJSON:
		return printJSON(out, res.Data())
	case field != "":
		raw, err := json.Marshal(res.Data())
		if err != nil {
			return err
		}
		var data map[string]any
		if err := json.Unmarshal(raw, &data); err != nil {
			return err
		}
		v, ok := data[field]
		if !ok {
			keys := make([]string, 0, len(data))
			for k := range data {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			return ExitErrorf(ExitBadArgument, "unknown field '%s'; available: %v", field, keys)
		}

		_, err = fmt.Fprintln(out, v)
		return err
	default:
		_, err := fmt.Fprintln(out, res.Text())
		return err
	}
}

func printJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)

	return enc.Encode(v)
}
