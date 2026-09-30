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
	"go.yaml.in/yaml/v3"
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
	cmd.Flags().Bool("json", false, "emit machine-readable JSON (alias of --format json)")
	cmd.Flags().String("format", "", "output format: text, json or yaml")
	_ = cmd.RegisterFlagCompletionFunc("format", cobra.FixedCompletions([]string{"text", "json", "yaml"}, cobra.ShellCompDirectiveNoFileComp))
	cmd.Flags().String("field", "", "print a single payload field (text output only)")
}

func Emit(cmd *cobra.Command, res Result) error {
	format, _ := cmd.Flags().GetString("format")
	asJSON, _ := cmd.Flags().GetBool("json")
	field, _ := cmd.Flags().GetString("field")

	if asJSON && format != "" && format != "json" {
		return ExitErrorf(ExitBadArgument, "--json is an alias of --format json; pass only one")
	}

	if asJSON {
		format = "json"
	}

	switch format {
	case "", "text":
		// text mode, --field applies
	case "json", "yaml":
		if field != "" {
			return ExitErrorf(ExitBadArgument, "--field only works with text output")
		}
	default:
		return ExitErrorf(ExitBadArgument, "invalid --format %q (want text, json or yaml)", format)
	}

	switch format {
	case "json":
		return printJSON(out, res.Data())
	case "yaml":
		data, err := yaml.Marshal(res.Data())
		if err != nil {
			return err
		}

		_, err = out.Write(data)
		return err
	default:
		return emitFieldOrText(res, field)
	}
}

// emitFieldOrText renders text output, extracting one field when --field is set.
func emitFieldOrText(res Result, field string) error {
	if field == "" {
		_, err := fmt.Fprintln(out, res.Text())
		return err
	}

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
}

func printJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)

	return enc.Encode(v)
}
