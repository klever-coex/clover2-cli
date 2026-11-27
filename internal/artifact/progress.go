package artifact

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/schollz/progressbar/v3"
	"golang.org/x/term"
)

func newProgressBar(maxBytes int64) *progressbar.ProgressBar {
	if term.IsTerminal(int(os.Stderr.Fd())) {
		return progressbar.DefaultBytes(maxBytes)
	}

	return progressbar.NewOptions64(
		maxBytes,
		progressbar.OptionSetWriter(io.Writer(progressLogWriter{os.Stderr})),
		progressbar.OptionThrottle(10*time.Second),
		progressbar.OptionShowCount(),
		progressbar.OptionShowBytes(true),
		progressbar.OptionShowTotalBytes(true),
		progressbar.OptionSetTheme(progressbar.Theme{}),
		progressbar.OptionSetWidth(0),
		progressbar.OptionSetRenderBlankState(true),
	)

}

type progressLogWriter struct{ io.Writer }

func (w progressLogWriter) Write(b []byte) (int, error) {
	// The renderer writes carriage returns and may clear the line with spaces.
	if line := strings.TrimSpace(string(b)); line != "" {
		if _, err := fmt.Fprintln(w.Writer, line); err != nil {
			return 0, err
		}
	}
	return len(b), nil
}
