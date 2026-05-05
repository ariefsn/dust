package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/briandowns/spinner"
	"github.com/mattn/go-isatty"
)

// startSpinner shows a spinner with `prefix` while the caller does work, with
// an elapsed-time counter (e.g. "Scanning... (2.3s)"). Returns a stop function
// the caller defers; stop() prints a final "✓ prefix (Xs)" line so the user
// sees how long each phase took.
//
// If `out` isn't a terminal, the spinner is a no-op (keeps `--json` clean and
// CI logs readable) but stop() still prints the timing line in plain text.
func startSpinner(out io.Writer, prefix string) (stop func()) {
	start := time.Now()

	if !isTerminal(out) {
		return func() {
			fmt.Fprintf(out, "✓ %s (%s)\n", prefix, formatElapsed(time.Since(start)))
		}
	}

	s := spinner.New(spinner.CharSets[14], 80*time.Millisecond, spinner.WithWriter(out))
	s.Suffix = " " + prefix
	s.Start()

	tickerDone := make(chan struct{})
	go func() {
		t := time.NewTicker(100 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-tickerDone:
				return
			case <-t.C:
				s.Suffix = fmt.Sprintf(" %s (%s)", prefix, formatElapsed(time.Since(start)))
			}
		}
	}()

	return func() {
		close(tickerDone)
		s.Stop()
		fmt.Fprintf(out, "✓ %s (%s)\n", prefix, formatElapsed(time.Since(start)))
	}
}

// progressBar renders a textual progress bar.
//
//	progressBar(3, 10, 20) -> "[▰▰▰▰▰▰▱▱▱▱▱▱▱▱▱▱▱▱▱▱]"
//
// Width is the bar's character width (filled + unfilled).
func progressBar(done, total, width int) string {
	if width < 4 {
		width = 4
	}
	if total <= 0 {
		return "[" + strings.Repeat("▱", width) + "]"
	}
	if done > total {
		done = total
	}
	filled := int(float64(width) * float64(done) / float64(total))
	if filled > width {
		filled = width
	}
	return "[" + strings.Repeat("▰", filled) + strings.Repeat("▱", width-filled) + "]"
}

// startProgressSpinner is a spinner with a live progress bar prefix. Update
// done/total via the returned setter; stop() prints a final ✓ line with the
// total elapsed time.
//
// In non-TTY environments it falls back to plain text — each setProgress call
// prints a one-line status update so CI logs stay readable.
func startProgressSpinner(out io.Writer, total int, label string) (setProgress func(done int, currentLine string), stop func()) {
	start := time.Now()
	const barWidth = 20

	if !isTerminal(out) {
		setProgress = func(done int, currentLine string) {
			fmt.Fprintf(out, "  %s [%d/%d] %s\n", label, done, total, currentLine)
		}
		stop = func() {
			fmt.Fprintf(out, "✓ %s (%s)\n", label, formatElapsed(time.Since(start)))
		}
		return
	}

	s := spinner.New(spinner.CharSets[14], 80*time.Millisecond, spinner.WithWriter(out))
	s.Prefix = " "
	s.Suffix = " " + label
	s.Start()

	doneCount := 0
	currentLine := ""

	tickerDone := make(chan struct{})
	go func() {
		t := time.NewTicker(100 * time.Millisecond)
		defer t.Stop()
		for {
			select {
			case <-tickerDone:
				return
			case <-t.C:
				bar := progressBar(doneCount, total, barWidth)
				elapsed := formatElapsed(time.Since(start))
				suffix := fmt.Sprintf(" %s [%d/%d] (%s)", bar, doneCount, total, elapsed)
				if currentLine != "" {
					suffix += " " + currentLine
				}
				s.Suffix = suffix
			}
		}
	}()

	setProgress = func(done int, line string) {
		doneCount = done
		currentLine = line
	}
	stop = func() {
		close(tickerDone)
		s.Stop()
		fmt.Fprintf(out, "✓ %s (%s)\n", label, formatElapsed(time.Since(start)))
	}
	return
}

func formatElapsed(d time.Duration) string {
	switch {
	case d < time.Second:
		return fmt.Sprintf("%dms", d.Milliseconds())
	case d < time.Minute:
		return fmt.Sprintf("%.1fs", d.Seconds())
	default:
		m := int(d.Minutes())
		s := int(d.Seconds()) - m*60
		return fmt.Sprintf("%dm%02ds", m, s)
	}
}

func isTerminal(w io.Writer) bool {
	f, ok := w.(*os.File)
	if !ok {
		return false
	}
	return isatty.IsTerminal(f.Fd())
}
