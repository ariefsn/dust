package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/ariefsn/dust/internal/cleaner"
	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		return m, nil

	case tickMsg:
		// Just trigger a re-render; the views read time.Since(...) directly.
		return m, tick()

	case scanAllMsg:
		m.categories = msg.categories
		m.scanning = false
		m.firstScan = false
		// Land on the first category with visible items under the current filter.
		for i, c := range m.categories {
			if categoryVisible(c, m.showEmpty) {
				m.catIdx = i
				m.itemIdx = m.firstVisibleItemIdx(c)
				break
			}
		}
		return m, nil

	case projectsLoadedMsg:
		m.projectsScanning = false
		m.projectsLoaded = true
		switch {
		case msg.scanErr != nil:
			m.statusMsg = fmt.Sprintf("Project scan failed: %v", msg.scanErr)
		case len(msg.items) > 0:
			sortItemsBySize(msg.items)
			m.categories = append(m.categories, &category{
				name:  "Projects",
				items: msg.items,
			})
			m.statusMsg = fmt.Sprintf("Projects: %d with reclaimable artifacts (out of %d found)",
				len(msg.items), msg.totalSeen)
		case msg.totalSeen == 0:
			rootsStr := strings.Join(msg.roots, ", ")
			if rootsStr == "" {
				rootsStr = "(no auto-detected project roots)"
			}
			m.statusMsg = "No projects found in: " + rootsStr
		default:
			m.statusMsg = fmt.Sprintf("Found %d projects, but none have reclaimable artifacts", msg.totalSeen)
		}
		return m, nil

	case runStartMsg:
		m.runCurrent = msg.name
		m.runDone = msg.idx - 1
		m.runTotal = msg.total
		if m.verbose {
			m.runLog = append(m.runLog, fmt.Sprintf("[%d/%d] starting %s", msg.idx, msg.total, msg.name))
		}
		return m, nil

	case runOneDoneMsg:
		m.runDone = msg.idx
		m.runTotal = msg.total
		m.runResults = append(m.runResults, runResult{itemID: msg.itemID, freed: msg.freed, err: msg.err})
		var line string
		switch {
		case msg.err != nil:
			line = fmt.Sprintf("[%d/%d] ✗ %s — %v", msg.idx, msg.total, msg.name, msg.err)
		case m.dryRun:
			line = fmt.Sprintf("[%d/%d] ~ %s — would free %s", msg.idx, msg.total, msg.name, cleaner.HumanBytes(msg.freed))
		default:
			line = fmt.Sprintf("[%d/%d] ✓ %s — freed %s", msg.idx, msg.total, msg.name, cleaner.HumanBytes(msg.freed))
		}
		// Always log errors so the user sees what failed; otherwise only in verbose.
		if m.verbose || msg.err != nil {
			m.runLog = append(m.runLog, line)
		}
		// Trim log to last 50 lines so we don't accumulate forever.
		if len(m.runLog) > 50 {
			m.runLog = m.runLog[len(m.runLog)-50:]
		}
		return m, nil

	case runAllDoneMsg:
		m.screen = screenDone
		// Skip the rescan after a dry-run — nothing was deleted, so sizes
		// haven't changed. Rescanning would just make the user wait for no
		// reason.
		if m.dryRun {
			return m, nil
		}
		// Real clean — refresh sizes so the list reflects post-clean state.
		m.scanning = true
		m.scanStart = time.Now()
		return m, scanAllCmd()

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.scanning {
		// Only quit + help while initial scan is in flight.
		if key.Matches(msg, m.keys.Quit) {
			return m, tea.Quit
		}
		return m, nil
	}

	if m.helpOpen {
		m.helpOpen = false
		return m, nil
	}

	switch m.screen {
	case screenConfirm:
		return m.handleKeyConfirm(msg)
	case screenDone:
		// Any key dismisses the result screen back to the list.
		m.screen = screenList
		m.runResults = nil
		return m, nil
	}

	switch {
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	case key.Matches(msg, m.keys.Help):
		m.helpOpen = true
		return m, nil
	case key.Matches(msg, m.keys.Refresh):
		m.scanning = true
		m.scanStart = time.Now()
		return m, scanAllCmd()
	case key.Matches(msg, m.keys.DryRun):
		m.dryRun = !m.dryRun
		if m.dryRun {
			m.statusMsg = "Dry-run mode: enabled"
		} else {
			m.statusMsg = "Dry-run mode: disabled"
		}
		return m, nil
	case key.Matches(msg, m.keys.Verbose):
		m.verbose = !m.verbose
		if m.verbose {
			m.statusMsg = "Verbose mode: enabled"
		} else {
			m.statusMsg = "Verbose mode: disabled"
		}
		return m, nil
	case key.Matches(msg, m.keys.ShowEmpty):
		m.showEmpty = !m.showEmpty
		if m.showEmpty {
			m.statusMsg = "Showing empty + unavailable items"
		} else {
			m.statusMsg = "Hiding empty + unavailable items"
		}
		// Clamp cursors back into bounds — visible list size just changed.
		m.clampCursors()
		return m, nil
	case key.Matches(msg, m.keys.Projects):
		if m.projectsScanning {
			m.statusMsg = "Project scan already in progress..."
			return m, nil
		}
		if m.projectsLoaded {
			m.statusMsg = "Projects already scanned — press r to rescan everything"
			return m, nil
		}
		m.projectsScanning = true
		m.statusMsg = "Scanning project dirs (this takes a few seconds)..."
		return m, scanProjectsCmd(m.projectsConfig, m.preferTool)
	case key.Matches(msg, m.keys.Up):
		m.moveUp()
		return m, nil
	case key.Matches(msg, m.keys.Down):
		m.moveDown()
		return m, nil
	case key.Matches(msg, m.keys.Left):
		m.focus = focusCategories
		return m, nil
	case key.Matches(msg, m.keys.Right):
		m.focus = focusItems
		m.itemIdx = m.firstVisibleItemIdx(m.currentCategory())
		return m, nil
	case key.Matches(msg, m.keys.Toggle):
		m.toggleCurrent()
		return m, nil
	case key.Matches(msg, m.keys.All):
		m.selectAllInCategory(true)
		return m, nil
	case key.Matches(msg, m.keys.None):
		m.selectAllInCategory(false)
		return m, nil
	case key.Matches(msg, m.keys.Clean):
		if m.anySelected() {
			m.screen = screenConfirm
		} else {
			m.statusMsg = "Nothing selected — press space on items first"
		}
		return m, nil
	}
	return m, nil
}

func (m Model) handleKeyConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case key.Matches(msg, m.keys.Cancel):
		m.screen = screenList
		return m, nil
	case key.Matches(msg, m.keys.Confirm):
		var items []*item
		for _, c := range m.categories {
			for _, it := range c.items {
				if it.selected {
					items = append(items, it)
				}
			}
		}
		m.screen = screenRunning
		m.runStart = time.Now()
		m.runDone = 0
		m.runTotal = len(items)
		m.runCurrent = ""
		m.runLog = nil
		m.runResults = nil
		return m, runSelectedCmds(items, m.dryRun)
	case key.Matches(msg, m.keys.Quit):
		return m, tea.Quit
	}
	return m, nil
}

func (m *Model) moveUp() {
	switch m.focus {
	case focusCategories:
		visCats := m.visibleCategoryIndexes()
		pos := indexOf(visCats, m.catIdx)
		if pos > 0 {
			m.catIdx = visCats[pos-1]
			m.itemIdx = m.firstVisibleItemIdx(m.currentCategory())
		}
	case focusItems:
		c := m.currentCategory()
		if c == nil {
			return
		}
		visItems := m.visibleItemIndexes(c)
		pos := indexOf(visItems, m.itemIdx)
		if pos > 0 {
			m.itemIdx = visItems[pos-1]
		}
	}
}

func (m *Model) moveDown() {
	switch m.focus {
	case focusCategories:
		visCats := m.visibleCategoryIndexes()
		pos := indexOf(visCats, m.catIdx)
		if pos >= 0 && pos < len(visCats)-1 {
			m.catIdx = visCats[pos+1]
			m.itemIdx = m.firstVisibleItemIdx(m.currentCategory())
		}
	case focusItems:
		c := m.currentCategory()
		if c == nil {
			return
		}
		visItems := m.visibleItemIndexes(c)
		pos := indexOf(visItems, m.itemIdx)
		if pos >= 0 && pos < len(visItems)-1 {
			m.itemIdx = visItems[pos+1]
		}
	}
}

// firstVisibleItemIdx returns the first visible item index in `c`, or 0 if none.
func (m *Model) firstVisibleItemIdx(c *category) int {
	if c == nil {
		return 0
	}
	if vis := m.visibleItemIndexes(c); len(vis) > 0 {
		return vis[0]
	}
	return 0
}

func indexOf(slice []int, v int) int {
	for i, x := range slice {
		if x == v {
			return i
		}
	}
	return -1
}

func (m *Model) currentCategory() *category {
	if m.catIdx < 0 || m.catIdx >= len(m.categories) {
		return nil
	}
	return m.categories[m.catIdx]
}

func (m *Model) currentItem() *item {
	c := m.currentCategory()
	if c == nil || m.itemIdx < 0 || m.itemIdx >= len(c.items) {
		return nil
	}
	return c.items[m.itemIdx]
}

func (m *Model) toggleCurrent() {
	if m.focus == focusCategories {
		// Toggle every available item in the current category.
		c := m.currentCategory()
		if c == nil {
			return
		}
		// Pick a target state based on whether everything is currently selected.
		allSel := true
		for _, it := range c.items {
			if it.scanned && !it.selected {
				allSel = false
				break
			}
		}
		for _, it := range c.items {
			if it.scanned {
				it.selected = !allSel
			}
		}
		return
	}
	it := m.currentItem()
	if it != nil && it.scanned {
		it.selected = !it.selected
	}
}

func (m *Model) selectAllInCategory(selected bool) {
	c := m.currentCategory()
	if c == nil {
		return
	}
	for _, it := range c.items {
		if it.scanned {
			it.selected = selected
		}
	}
}

func (m *Model) anySelected() bool {
	for _, c := range m.categories {
		for _, it := range c.items {
			if it.selected {
				return true
			}
		}
	}
	return false
}

// sortItemsBySize sorts a list of items biggest-first.
func sortItemsBySize(items []*item) {
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && items[j].res.Bytes > items[j-1].res.Bytes; j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
}

func hasAvailable(c *category) bool {
	for _, it := range c.items {
		if it.scanned {
			return true
		}
	}
	return false
}

// itemVisible reports whether an item should be rendered in the items pane.
// When showEmpty is false, hide both unavailable items (scan failed / dir
// doesn't exist) and items with 0 bytes — they're not actionable.
func itemVisible(it *item, showEmpty bool) bool {
	if showEmpty {
		return true
	}
	if !it.scanned {
		return false
	}
	return it.res.Bytes > 0
}

// categoryVisible reports whether a category should appear in the categories
// pane — true if any of its items are visible under the current filter.
func categoryVisible(c *category, showEmpty bool) bool {
	if showEmpty {
		return true
	}
	for _, it := range c.items {
		if itemVisible(it, showEmpty) {
			return true
		}
	}
	return false
}

// visibleCategories returns the indexes (into m.categories) that survive the
// current filter, in their original order.
func (m Model) visibleCategoryIndexes() []int {
	var out []int
	for i, c := range m.categories {
		if categoryVisible(c, m.showEmpty) {
			out = append(out, i)
		}
	}
	return out
}

// visibleItemIndexes returns the indexes (into c.items) of items that survive
// the current filter, in their original order.
func (m Model) visibleItemIndexes(c *category) []int {
	var out []int
	for i, it := range c.items {
		if itemVisible(it, m.showEmpty) {
			out = append(out, i)
		}
	}
	return out
}

// clampCursors keeps catIdx/itemIdx pointing at a visible row after the
// filter changes. Falls back to 0 (or stays put when there's nothing visible).
func (m *Model) clampCursors() {
	visCats := m.visibleCategoryIndexes()
	if len(visCats) == 0 {
		return
	}
	// Snap to the nearest visible category, preferring the current one if it's
	// still visible.
	if !categoryVisible(m.categories[m.catIdx], m.showEmpty) {
		m.catIdx = visCats[0]
		m.itemIdx = 0
		return
	}
	c := m.currentCategory()
	if c == nil {
		return
	}
	visItems := m.visibleItemIndexes(c)
	if len(visItems) == 0 {
		m.itemIdx = 0
		return
	}
	// If the current item is no longer visible, snap to the first visible one.
	if !itemVisible(c.items[m.itemIdx], m.showEmpty) {
		m.itemIdx = visItems[0]
	}
}
