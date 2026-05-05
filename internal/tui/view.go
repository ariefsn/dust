package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/ariefsn/dust/internal/cleaner"
	"github.com/charmbracelet/lipgloss"
)

// formatElapsed mirrors cli.formatElapsed for consistent timing display.
func formatElapsed(d time.Duration) string {
	switch {
	case d < time.Second:
		return fmt.Sprintf("%dms", d.Milliseconds())
	case d < time.Minute:
		return fmt.Sprintf("%.1fs", d.Seconds())
	default:
		mins := int(d.Minutes())
		secs := int(d.Seconds()) - mins*60
		return fmt.Sprintf("%dm%02ds", mins, secs)
	}
}

// spinnerFrames cycles a braille spinner; pick the frame from current time.
var spinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func spinnerFrame() string {
	// 80ms per frame — same cadence as the CLI spinner.
	idx := int(time.Now().UnixMilli()/80) % len(spinnerFrames)
	return spinnerFrames[idx]
}

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#FAFAFA")).
			Background(lipgloss.Color("#7D56F4")).
			Padding(0, 1)

	paneTitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4"))

	paneBorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#5C5C5C")).
			Padding(0, 1)

	focusedBorderStyle = paneBorderStyle.BorderForeground(lipgloss.Color("#7D56F4"))

	cursorStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4")).Bold(true)
	checkedStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#46DCB7"))
	dimStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#6C6C6C"))
	sizeStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFCB6B"))
	dryRunStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF9800")).Bold(true)
	helpKeyStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("#7D56F4"))
	helpDescStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#A0A0A0"))
	errStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF6B6B"))
)

func (m Model) View() string {
	if m.scanning {
		return m.renderScanning()
	}
	switch m.screen {
	case screenConfirm:
		return m.renderConfirm()
	case screenRunning:
		return m.renderRunning()
	case screenDone:
		return m.renderDone()
	}
	if m.helpOpen {
		return m.renderHelp()
	}
	return m.renderList()
}

func (m Model) renderScanning() string {
	elapsed := formatElapsed(time.Since(m.scanStart))
	verb := "Scanning all caches in parallel..."
	hint := "initial scan can take 10–30s depending on cache sizes"
	if !m.firstScan {
		verb = "Refreshing sizes..."
		hint = "rescanning every cleaner — usually a few seconds"
	}
	return fmt.Sprintf("\n  %s\n  %s %s %s\n  %s\n",
		titleStyle.Render(" dust "),
		cursorStyle.Render(spinnerFrame()),
		verb,
		dimStyle.Render("("+elapsed+")"),
		dimStyle.Render(hint),
	)
}

func (m Model) renderList() string {
	w, h := m.width, m.height
	if w == 0 {
		w = 100
	}
	if h == 0 {
		h = 30
	}

	leftWidth := 30
	rightWidth := w - leftWidth - 6 // borders + gap
	if rightWidth < 30 {
		rightWidth = 30
	}
	bodyHeight := h - 6
	if bodyHeight < 10 {
		bodyHeight = 10
	}

	left := m.renderCategories(leftWidth, bodyHeight)
	right := m.renderItems(rightWidth, bodyHeight)
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	header := m.renderHeader(w)
	footer := m.renderFooter(w)
	return lipgloss.JoinVertical(lipgloss.Left, header, body, footer)
}

func (m Model) renderHeader(w int) string {
	title := titleStyle.Render(" dust ")
	mode := ""
	if m.dryRun {
		mode = " " + dryRunStyle.Render("DRY-RUN")
	}
	totalSel, countSel := m.selectionStats()
	stats := dimStyle.Render(fmt.Sprintf("  %d selected · %s reclaimable",
		countSel, cleaner.HumanBytes(totalSel)))
	scanHint := ""
	if m.projectsScanning {
		scanHint = "  " + cursorStyle.Render(spinnerFrame()) + dimStyle.Render(" scanning projects...")
	}
	return title + mode + stats + scanHint
}

func (m Model) selectionStats() (int64, int) {
	var total int64
	var n int
	seen := map[string]bool{}
	for _, c := range m.categories {
		for _, it := range c.items {
			if !it.selected {
				continue
			}
			n++
			if it.res.Path != "" && seen[it.res.Path] {
				continue
			}
			if it.res.Path != "" {
				seen[it.res.Path] = true
			}
			total += it.res.Bytes
		}
	}
	return total, n
}

func (m Model) renderCategories(width, height int) string {
	var lines []string
	lines = append(lines, paneTitleStyle.Render("Categories"))
	for i, c := range m.categories {
		if !categoryVisible(c, m.showEmpty) {
			continue
		}
		cursor := " "
		if i == m.catIdx {
			cursor = cursorStyle.Render("▸")
		}
		size := cleaner.HumanBytes(c.totalBytes())
		nameLen := width - 14
		if nameLen < 8 {
			nameLen = 8
		}
		name := truncate(c.name, nameLen)
		line := fmt.Sprintf("%s %s %s", cursor, padRight(name, nameLen), sizeStyle.Render(rightAlign(size, 9)))
		if i == m.catIdx && m.focus == focusCategories {
			line = lipgloss.NewStyle().Bold(true).Render(line)
		}
		lines = append(lines, line)
	}
	if !m.showEmpty {
		// Tell the user how to see what's hidden — easy to miss otherwise.
		hidden := len(m.categories) - len(m.visibleCategoryIndexes())
		if hidden > 0 {
			lines = append(lines, "", dimStyle.Render(fmt.Sprintf("(%d empty hidden — press s)", hidden)))
		}
	}
	style := paneBorderStyle.Width(width).Height(height)
	if m.focus == focusCategories {
		style = focusedBorderStyle.Width(width).Height(height)
	}
	return style.Render(strings.Join(lines, "\n"))
}

func (m Model) renderItems(width, height int) string {
	c := m.currentCategory()
	if c == nil {
		return paneBorderStyle.Width(width).Height(height).Render("(no category)")
	}
	var lines []string
	lines = append(lines, paneTitleStyle.Render(c.name))

	visIdxs := m.visibleItemIndexes(c)

	if len(visIdxs) == 0 {
		// Empty category that survived the filter only because we're
		// showing-all elsewhere — give the user a useful message.
		lines = append(lines, "", dimStyle.Render("(no items — try pressing s)"))
		style := paneBorderStyle.Width(width).Height(height)
		if m.focus == focusItems {
			style = focusedBorderStyle.Width(width).Height(height)
		}
		return style.Render(strings.Join(lines, "\n"))
	}

	maxLines := height - 4
	if maxLines < 5 {
		maxLines = 5
	}

	// Find the cursor's position within the visible list, then page based on
	// that — so scrolling stays intuitive when invisible items are skipped.
	cursorVisPos := 0
	for i, idx := range visIdxs {
		if idx == m.itemIdx {
			cursorVisPos = i
			break
		}
	}
	start := 0
	if m.focus == focusItems && cursorVisPos >= maxLines {
		start = cursorVisPos - maxLines + 1
	}
	end := start + maxLines
	if end > len(visIdxs) {
		end = len(visIdxs)
	}

	for i := start; i < end; i++ {
		origIdx := visIdxs[i]
		lines = append(lines, m.renderItemLine(c.items[origIdx], origIdx, width))
	}

	if !m.showEmpty {
		hidden := len(c.items) - len(visIdxs)
		if hidden > 0 {
			lines = append(lines, dimStyle.Render(fmt.Sprintf("\n(%d empty hidden — press s)", hidden)))
		}
	}

	style := paneBorderStyle.Width(width).Height(height)
	if m.focus == focusItems {
		style = focusedBorderStyle.Width(width).Height(height)
	}
	return style.Render(strings.Join(lines, "\n"))
}

func (m Model) renderItemLine(it *item, idx, width int) string {
	cursor := " "
	if m.focus == focusItems && idx == m.itemIdx {
		cursor = cursorStyle.Render("▸")
	}
	check := "[ ]"
	switch {
	case !it.scanned:
		check = dimStyle.Render("[ ]")
	case it.selected:
		check = checkedStyle.Render("[x]")
	}

	name := it.c.Name()
	size := ""
	switch {
	case !it.scanned:
		size = dimStyle.Render("not installed")
	case it.scanErr != nil:
		size = errStyle.Render("error")
	default:
		size = sizeStyle.Render(cleaner.HumanBytes(it.res.Bytes))
	}

	nameLen := width - 24
	if nameLen < 12 {
		nameLen = 12
	}
	line := fmt.Sprintf("%s %s %s %s", cursor, check, padRight(truncate(name, nameLen), nameLen), rightAlign(size, 14))
	if !it.scanned {
		line = dimStyle.Render(line)
	}
	return line
}

func (m Model) renderFooter(w int) string {
	keys := []string{
		fmt.Sprintf("%s %s", helpKeyStyle.Render("space"), helpDescStyle.Render("toggle")),
		fmt.Sprintf("%s %s", helpKeyStyle.Render("a/n"), helpDescStyle.Render("all/none")),
		fmt.Sprintf("%s %s", helpKeyStyle.Render("d"), helpDescStyle.Render("dry-run")),
		fmt.Sprintf("%s %s", helpKeyStyle.Render("s"), helpDescStyle.Render("show-empty")),
		fmt.Sprintf("%s %s", helpKeyStyle.Render("p"), helpDescStyle.Render("projects")),
		fmt.Sprintf("%s %s", helpKeyStyle.Render("r"), helpDescStyle.Render("rescan")),
		fmt.Sprintf("%s %s", helpKeyStyle.Render("enter"), helpDescStyle.Render("clean")),
		fmt.Sprintf("%s %s", helpKeyStyle.Render("?"), helpDescStyle.Render("help")),
		fmt.Sprintf("%s %s", helpKeyStyle.Render("q"), helpDescStyle.Render("quit")),
	}
	bar := strings.Join(keys, "  ·  ")
	if m.statusMsg != "" {
		bar = bar + "\n" + dimStyle.Render(m.statusMsg)
	}
	return bar
}

func (m Model) renderConfirm() string {
	totalSel, countSel := m.selectionStats()
	verb := "Clean"
	if m.dryRun {
		verb = "[dry-run] Preview cleaning of"
	}
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("#FFCB6B")).
		Padding(1, 2).
		Width(60)
	body := fmt.Sprintf("%s %d cleaner(s)?\n\nReclaimable: %s\n\n%s   %s",
		verb,
		countSel,
		sizeStyle.Render(cleaner.HumanBytes(totalSel)),
		helpKeyStyle.Render("y"),
		helpDescStyle.Render("confirm"),
	)
	body = body + "    " + helpKeyStyle.Render("n/esc") + " " + helpDescStyle.Render("cancel")
	return "\n" + box.Render(body)
}

func (m Model) renderRunning() string {
	elapsed := formatElapsed(time.Since(m.runStart))
	header := fmt.Sprintf("  %s  %s Cleaning [%d/%d] %s",
		titleStyle.Render(" dust "),
		cursorStyle.Render(spinnerFrame()),
		m.runDone,
		m.runTotal,
		dimStyle.Render("("+elapsed+")"),
	)

	// Progress bar — width adapts to terminal so it doesn't wrap.
	barWidth := m.width - 4
	if barWidth < 20 {
		barWidth = 20
	}
	bar := m.runProgress
	bar.Width = barWidth
	var ratio float64
	if m.runTotal > 0 {
		ratio = float64(m.runDone) / float64(m.runTotal)
	}
	progressLine := "\n  " + bar.ViewAs(ratio)

	cur := ""
	if m.runCurrent != "" {
		cur = "\n  " + dimStyle.Render("→ ") + m.runCurrent
	}
	verboseHint := ""
	if !m.verbose {
		verboseHint = "\n  " + dimStyle.Render("press v to enable verbose log")
	}

	logLines := ""
	if m.verbose && len(m.runLog) > 0 {
		// Show the last N lines that fit; for v1, hard-cap at 20.
		max := 20
		start := 0
		if len(m.runLog) > max {
			start = len(m.runLog) - max
		}
		var b strings.Builder
		b.WriteString("\n\n  ")
		b.WriteString(paneTitleStyle.Render("Log"))
		b.WriteString("\n")
		for _, ln := range m.runLog[start:] {
			b.WriteString("  ")
			switch {
			case strings.Contains(ln, "✗"):
				b.WriteString(errStyle.Render(ln))
			case strings.Contains(ln, "✓"):
				b.WriteString(checkedStyle.Render(ln))
			default:
				b.WriteString(dimStyle.Render(ln))
			}
			b.WriteString("\n")
		}
		logLines = b.String()
	}

	return "\n" + header + progressLine + cur + logLines + verboseHint + "\n"
}

func (m Model) renderDone() string {
	var totalFreed int64
	var failures int
	for _, r := range m.runResults {
		if r.err != nil {
			failures++
			continue
		}
		totalFreed += r.freed
	}
	verb := "Freed"
	if m.dryRun {
		verb = "[dry-run] Would have freed"
	}
	header := fmt.Sprintf("%s %s across %d cleaner(s).",
		verb,
		sizeStyle.Render(cleaner.HumanBytes(totalFreed)),
		len(m.runResults)-failures,
	)
	if failures > 0 {
		header += "\n  " + errStyle.Render(fmt.Sprintf("%d cleaner(s) failed.", failures))
	}

	// Per-cleaner breakdown — useful especially in dry-run, where the rescan
	// is skipped so this is the user's only feedback on what would happen.
	var rows []string
	for _, r := range m.runResults {
		name := m.lookupName(r.itemID)
		var line string
		switch {
		case r.err != nil:
			line = fmt.Sprintf("  %s %s — %v", errStyle.Render("✗"), name, r.err)
		case m.dryRun:
			line = fmt.Sprintf("  %s %s — would free %s",
				dimStyle.Render("~"), name, sizeStyle.Render(cleaner.HumanBytes(r.freed)))
		default:
			line = fmt.Sprintf("  %s %s — freed %s",
				checkedStyle.Render("✓"), name, sizeStyle.Render(cleaner.HumanBytes(r.freed)))
		}
		rows = append(rows, line)
	}

	body := "\n  " + titleStyle.Render(" dust ") + "\n\n  " + header + "\n\n" + strings.Join(rows, "\n")
	body += "\n\n  " + dimStyle.Render("Press any key to continue.")
	return body
}

// lookupName returns the cleaner Name() for an itemID, or the ID itself if
// unknown (covers the case where the rescan dropped the cleaner).
func (m Model) lookupName(id string) string {
	for _, c := range m.categories {
		for _, it := range c.items {
			if it.c.ID() == id {
				return it.c.Name()
			}
		}
	}
	return id
}

func (m Model) renderHelp() string {
	rows := [][2]string{
		{"↑/k, ↓/j", "navigate within the focused pane"},
		{"←/h, →/l", "switch panes"},
		{"space", "toggle the highlighted item (or whole category if focus is on the left)"},
		{"a / n", "select all / select none in current category"},
		{"d", "toggle dry-run mode"},
		{"v", "toggle verbose log during clean"},
		{"s", "toggle show-empty (reveal hidden 0 B / not-installed)"},
		{"p", "scan project dirs (~/Projects, ~/Work, ...)"},
		{"r", "rescan everything"},
		{"enter", "clean every selected item (asks for confirmation)"},
		{"?", "toggle help"},
		{"q, ctrl+c", "quit"},
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render(" dust — keys "))
	b.WriteString("\n\n")
	for _, r := range rows {
		b.WriteString("  ")
		b.WriteString(helpKeyStyle.Render(padRight(r[0], 14)))
		b.WriteString(" ")
		b.WriteString(helpDescStyle.Render(r[1]))
		b.WriteString("\n")
	}
	b.WriteString("\n  ")
	b.WriteString(dimStyle.Render("press any key to close"))
	return b.String()
}

// helpers

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return "…"
	}
	return s[:n-1] + "…"
}

func padRight(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}

func rightAlign(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return strings.Repeat(" ", n-len(s)) + s
}
