package main

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gdamore/tcell/v3"
	"github.com/rivo/uniseg"
)

var errQuit = errors.New("quit")

type diffLayout uint8

const (
	layoutSplit diffLayout = iota
	layoutUnified
)

func (layout diffLayout) String() string {
	if layout == layoutUnified {
		return "unified"
	}
	return "split"
}

type app struct {
	screen          tcell.Screen
	rows            []diffRow
	sourceRows      []diffRow
	reload          func() ([]diffRow, error)
	vertical        int
	horizontal      int
	status          string
	statusIsError   bool
	refreshEvery    time.Duration
	searching       bool
	searchQuery     string
	searchRegex     *regexp.Regexp
	searchMatches   []int
	searchCurrent   int
	searchError     string
	searchStart     int
	searchBefore    string
	showSidebar     bool
	currentFile     int
	distractionFree bool
	shortcutsOpen   bool
	viewedFiles     map[string][]diffRow
	singleFilePath  string
	layout          diffLayout
}

func newApp(screen tcell.Screen, rows []diffRow, reload func() ([]diffRow, error), refreshEvery time.Duration) *app {
	return &app{
		screen:        screen,
		rows:          rows,
		sourceRows:    rows,
		reload:        reload,
		refreshEvery:  refreshEvery,
		viewedFiles:   make(map[string][]diffRow),
		searchCurrent: -1,
		layout:        layoutSplit,
	}
}

func (a *app) run() error {
	var ticker *time.Ticker
	var refresh <-chan time.Time
	if a.refreshEvery > 0 {
		ticker = time.NewTicker(a.refreshEvery)
		refresh = ticker.C
		defer ticker.Stop()
	}

	a.draw()
	for {
		select {
		case event, ok := <-a.screen.EventQ():
			if !ok {
				return nil
			}
			switch typedEvent := event.(type) {
			case *tcell.EventResize:
				a.screen.Sync()
				a.clampOffsets()
			case *tcell.EventKey:
				if err := a.handleKey(typedEvent); err != nil {
					return err
				}
			}
			a.draw()
		case <-refresh:
			if a.refresh(false) {
				a.draw()
			}
		}
	}
}

func (a *app) handleKey(event *tcell.EventKey) error {
	if a.searching {
		return a.handleSearchKey(event)
	}
	if a.shortcutsOpen {
		return a.handleShortcutsKey(event)
	}

	_, height := a.screen.Size()
	page := max(1, height-3)
	if event.Str() != "r" {
		a.status = ""
		a.statusIsError = false
	}

	switch {
	case event.Key() == tcell.KeyCtrlC || event.Str() == "q":
		return errQuit
	case event.Key() == tcell.KeyEscape:
		if a.singleFilePath != "" {
			a.exitSingleFileMode()
			break
		}
		return errQuit
	case event.Key() == tcell.KeyEnter:
		a.toggleSingleFileMode()
	case event.Key() == tcell.KeyDown || event.Str() == "j":
		a.moveTo(a.vertical + 1)
	case event.Key() == tcell.KeyUp || event.Str() == "k":
		a.moveTo(a.vertical - 1)
	case event.Key() == tcell.KeyPgDn || event.Str() == " ":
		a.moveTo(a.vertical + page)
	case event.Key() == tcell.KeyPgUp:
		a.moveTo(a.vertical - page)
	case event.Key() == tcell.KeyHome || event.Str() == "g":
		a.moveTo(0)
	case event.Key() == tcell.KeyEnd || event.Str() == "G":
		a.moveTo(len(a.rows))
	case event.Key() == tcell.KeyRight || event.Str() == "l":
		a.horizontal += 4
	case event.Key() == tcell.KeyLeft || event.Str() == "h":
		a.horizontal -= 4
	case event.Str() == "n":
		if a.searchRegex != nil {
			a.jumpToSearch(true)
		} else {
			a.jumpTo(rowHunk, true)
		}
	case event.Str() == "N":
		if a.searchRegex != nil {
			a.jumpToSearch(false)
		} else {
			a.jumpTo(rowHunk, false)
		}
	case event.Str() == "]":
		a.jumpToFile(true)
	case event.Str() == "[":
		a.jumpToFile(false)
	case event.Str() == "r":
		a.refresh(true)
	case event.Str() == "/":
		a.startSearch()
	case event.Str() == "f":
		a.toggleSidebar()
	case event.Str() == "z":
		a.toggleDistractionFree()
	case event.Str() == "s":
		a.toggleLayout()
	case event.Str() == "v":
		a.toggleCurrentFileViewed()
	case event.Str() == "?":
		a.shortcutsOpen = true
	}

	a.clampOffsets()
	return nil
}

func (a *app) handleShortcutsKey(event *tcell.EventKey) error {
	if event.Key() == tcell.KeyCtrlC {
		return errQuit
	}
	if event.Key() == tcell.KeyEscape || event.Str() == "q" || event.Str() == "?" {
		a.shortcutsOpen = false
	}
	return nil
}

func (a *app) moveTo(row int) {
	a.vertical = row
	files := changedFiles(a.rows)
	if current := currentFileIndex(files, row); current >= 0 {
		a.currentFile = current
	}
}

func (a *app) jumpToFile(forward bool) {
	if a.singleFilePath != "" {
		a.jumpToSingleFile(forward)
		return
	}
	files := changedFiles(a.rows)
	if len(files) == 0 {
		return
	}
	order := sidebarFileOrder(files)
	target := filePosition(order, a.currentFile)
	if target < 0 {
		return
	}
	if forward {
		target++
	} else {
		target--
	}
	if target < 0 || target >= len(files) {
		return
	}
	a.currentFile = order[target]
	a.vertical = files[a.currentFile].row
}

func (a *app) fullRows() []diffRow {
	if a.sourceRows != nil {
		return a.sourceRows
	}
	return a.rows
}

func (a *app) rebuildVisibleRows() {
	rows := a.fullRows()
	if a.singleFilePath != "" {
		files := changedFiles(rows)
		fileIndex := changedFileIndex(files, a.singleFilePath)
		rows = diffRowsForFile(rows, files, fileIndex)
	}
	if a.layout == layoutUnified {
		rows = unifiedRows(rows)
	}
	a.rows = rows
}

func unifiedRows(rows []diffRow) []diffRow {
	result := make([]diffRow, 0, len(rows))
	for _, row := range rows {
		if row.kind != rowChange {
			result = append(result, row)
			continue
		}
		if row.left.kind != cellEmpty {
			result = append(result, diffRow{kind: rowChange, left: row.left})
		}
		if row.right.kind != cellEmpty {
			result = append(result, diffRow{kind: rowChange, right: row.right})
		}
	}
	return result
}

func (a *app) currentFilePath() string {
	if a.singleFilePath != "" {
		return a.singleFilePath
	}
	files := changedFiles(a.rows)
	if a.currentFile < 0 || a.currentFile >= len(files) {
		return ""
	}
	return files[a.currentFile].path
}

func (a *app) toggleLayout() {
	path := a.currentFilePath()
	if a.sourceRows == nil {
		a.sourceRows = slices.Clone(a.rows)
	}
	if a.layout == layoutSplit {
		a.layout = layoutUnified
	} else {
		a.layout = layoutSplit
	}
	a.rebuildVisibleRows()
	files := changedFiles(a.rows)
	fileIndex := changedFileIndex(files, path)
	if fileIndex >= 0 {
		a.currentFile = fileIndex
		a.vertical = files[fileIndex].row
	} else {
		a.currentFile = min(a.currentFile, max(0, len(files)-1))
		a.vertical = 0
	}
	if a.searchRegex != nil {
		a.rebuildSearchMatches()
	}
	a.status = a.layout.String() + " view"
}

func (a *app) toggleSingleFileMode() {
	if a.singleFilePath != "" {
		a.exitSingleFileMode()
		return
	}
	files := changedFiles(a.fullRows())
	if a.currentFile < 0 || a.currentFile >= len(files) {
		return
	}
	a.enterSingleFileMode(files[a.currentFile].path)
}

func (a *app) enterSingleFileMode(path string) bool {
	if a.sourceRows == nil {
		a.sourceRows = a.rows
	}
	files := changedFiles(a.fullRows())
	fileIndex := changedFileIndex(files, path)
	if fileIndex < 0 {
		return false
	}
	a.singleFilePath = path
	a.rebuildVisibleRows()
	a.currentFile = 0
	a.vertical = 0
	if a.searchRegex != nil {
		a.rebuildSearchMatches()
	}
	a.status = "single file: " + files[fileIndex].name
	return true
}

func (a *app) exitSingleFileMode() {
	path := a.singleFilePath
	a.singleFilePath = ""
	a.rebuildVisibleRows()
	files := changedFiles(a.rows)
	fileIndex := changedFileIndex(files, path)
	if fileIndex >= 0 {
		a.currentFile = fileIndex
		a.vertical = files[fileIndex].row
	} else {
		a.currentFile = min(a.currentFile, max(0, len(files)-1))
		a.vertical = 0
	}
	if a.searchRegex != nil {
		a.rebuildSearchMatches()
	}
	a.status = "all files"
}

func (a *app) jumpToSingleFile(forward bool) {
	files := changedFiles(a.fullRows())
	fileIndex := changedFileIndex(files, a.singleFilePath)
	if fileIndex < 0 {
		return
	}
	order := sidebarFileOrder(files)
	target := filePosition(order, fileIndex)
	if forward {
		target++
	} else {
		target--
	}
	if target < 0 || target >= len(order) {
		return
	}
	a.enterSingleFileMode(files[order[target]].path)
}

func changedFileIndex(files []changedFile, path string) int {
	for index, file := range files {
		if file.path == path {
			return index
		}
	}
	return -1
}

func (a *app) toggleSidebar() {
	if a.distractionFree {
		a.distractionFree = false
		a.showSidebar = false
	}
	if a.showSidebar {
		a.showSidebar = false
		a.status = "file list hidden"
		return
	}
	width, _ := a.screen.Size()
	if width < minimumSidebarTerminalWidth {
		a.status = "terminal too narrow for file list"
		a.statusIsError = true
		return
	}
	a.showSidebar = true
	a.status = "file list shown"
}

func (a *app) toggleDistractionFree() {
	a.distractionFree = !a.distractionFree
	if a.distractionFree {
		a.status = ""
		return
	}
	a.status = "distraction-free mode off"
}

func (a *app) toggleCurrentFileViewed() {
	files := changedFiles(a.rows)
	if a.currentFile < 0 || a.currentFile >= len(files) {
		return
	}
	file := files[a.currentFile]
	if _, viewed := a.viewedFiles[file.path]; viewed {
		delete(a.viewedFiles, file.path)
		a.status = file.name + " marked unviewed"
		return
	}
	if a.viewedFiles == nil {
		a.viewedFiles = make(map[string][]diffRow)
	}
	sourceFiles := changedFiles(a.fullRows())
	sourceIndex := changedFileIndex(sourceFiles, file.path)
	a.viewedFiles[file.path] = slices.Clone(diffRowsForFile(a.fullRows(), sourceFiles, sourceIndex))
	a.status = file.name + " marked viewed"
}

func (a *app) fileIsViewed(path string) bool {
	_, viewed := a.viewedFiles[path]
	return viewed
}

func (a *app) reconcileViewedFiles(rows []diffRow) {
	if len(a.viewedFiles) == 0 {
		return
	}
	files := changedFiles(rows)
	current := make(map[string][]diffRow, len(files))
	for index, file := range files {
		current[file.path] = diffRowsForFile(rows, files, index)
	}
	for path, snapshot := range a.viewedFiles {
		if rows, exists := current[path]; !exists || !slices.Equal(snapshot, rows) {
			delete(a.viewedFiles, path)
		}
	}
}

func (a *app) startSearch() {
	a.searching = true
	a.searchStart = a.vertical
	a.searchBefore = a.searchQuery
	a.setSearchQuery("")
}

func (a *app) handleSearchKey(event *tcell.EventKey) error {
	switch event.Key() {
	case tcell.KeyEscape:
		a.searching = false
		a.moveTo(a.searchStart)
		a.setSearchQuery(a.searchBefore)
	case tcell.KeyCtrlC:
		return errQuit
	case tcell.KeyEnter:
		if a.searchError != "" {
			return nil
		}
		a.searching = false
		if a.searchQuery == "" {
			a.status = "search cleared"
		} else {
			a.status = fmt.Sprintf("%d matching rows", len(a.searchMatches))
		}
		a.statusIsError = false
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		a.setSearchQuery(trimLastRune(a.searchQuery))
	case tcell.KeyCtrlU:
		a.setSearchQuery("")
	case tcell.KeyRune:
		if event.Modifiers()&(tcell.ModCtrl|tcell.ModAlt|tcell.ModMeta) == 0 {
			a.setSearchQuery(a.searchQuery + event.Str())
		}
	}
	a.clampOffsets()
	return nil
}

func trimLastRune(value string) string {
	runes := []rune(value)
	if len(runes) == 0 {
		return value
	}
	return string(runes[:len(runes)-1])
}

func (a *app) setSearchQuery(query string) {
	a.searchQuery = query
	a.searchRegex = nil
	a.searchMatches = nil
	a.searchCurrent = -1
	a.searchError = ""
	a.status = ""
	a.statusIsError = false
	if query == "" {
		return
	}

	expression, err := regexp.Compile(query)
	if err != nil {
		a.searchError = err.Error()
		return
	}
	a.searchRegex = expression
	a.rebuildSearchMatches()
	if a.searchCurrent >= 0 {
		a.moveTo(a.searchMatches[a.searchCurrent])
	}
}

func (a *app) rebuildSearchMatches() {
	a.searchMatches = nil
	a.searchCurrent = -1
	if a.searchRegex == nil {
		return
	}
	for index, row := range a.rows {
		if rowMatches(row, a.searchRegex) {
			a.searchMatches = append(a.searchMatches, index)
		}
	}
	for index, rowIndex := range a.searchMatches {
		if rowIndex >= a.vertical {
			a.searchCurrent = index
			return
		}
	}
	if len(a.searchMatches) > 0 {
		a.searchCurrent = 0
	}
}

func rowMatches(row diffRow, expression *regexp.Regexp) bool {
	for _, value := range rowSearchValues(row) {
		if expression.MatchString(value) {
			return true
		}
	}
	return false
}

func rowSearchValues(row diffRow) []string {
	if row.kind == rowMeta {
		return []string{row.text}
	}
	return []string{row.left.text, row.right.text}
}

func (a *app) jumpToSearch(forward bool) {
	if len(a.searchMatches) == 0 {
		return
	}
	if forward {
		a.searchCurrent = (a.searchCurrent + 1) % len(a.searchMatches)
	} else {
		a.searchCurrent = (a.searchCurrent - 1 + len(a.searchMatches)) % len(a.searchMatches)
	}
	a.moveTo(a.searchMatches[a.searchCurrent])
}

func (a *app) jumpTo(kind rowKind, forward bool) {
	if forward {
		for index := a.vertical + 1; index < len(a.rows); index++ {
			if a.rows[index].kind == kind {
				a.moveTo(index)
				return
			}
		}
		return
	}

	for index := min(a.vertical-1, len(a.rows)-1); index >= 0; index-- {
		if a.rows[index].kind == kind {
			a.moveTo(index)
			return
		}
	}
}

func (a *app) refresh(manual bool) bool {
	selectedPath := a.currentFilePath()
	rows, err := a.reload()
	if err != nil {
		changed := !a.statusIsError || a.status != err.Error()
		a.status = err.Error()
		a.statusIsError = true
		return changed
	}
	changed := !slices.Equal(a.fullRows(), rows)
	a.reconcileViewedFiles(rows)
	a.sourceRows = rows
	if a.singleFilePath != "" {
		files := changedFiles(rows)
		fileIndex := changedFileIndex(files, a.singleFilePath)
		if fileIndex >= 0 {
			a.rebuildVisibleRows()
			a.currentFile = 0
		} else {
			a.singleFilePath = ""
			a.rebuildVisibleRows()
			visibleFiles := changedFiles(a.rows)
			a.currentFile = min(a.currentFile, max(0, len(visibleFiles)-1))
			a.vertical = 0
			a.status = "focused file no longer has changes"
		}
	} else {
		a.rebuildVisibleRows()
		files := changedFiles(a.rows)
		fileIndex := changedFileIndex(files, selectedPath)
		if fileIndex >= 0 {
			a.currentFile = fileIndex
		} else {
			a.currentFile = min(a.currentFile, max(0, len(files)-1))
		}
	}
	if a.searchRegex != nil {
		a.rebuildSearchMatches()
	}
	if manual {
		a.status = "refreshed"
		changed = true
	} else if a.statusIsError {
		a.status = ""
		changed = true
	}
	a.statusIsError = false
	a.clampOffsets()
	return changed
}

func (a *app) clampOffsets() {
	_, height := a.screen.Size()
	viewportHeight := a.diffViewportHeight(height)
	maxVertical := max(0, len(a.rows)-viewportHeight)
	a.vertical = min(max(a.vertical, 0), maxVertical)
	a.horizontal = max(a.horizontal, 0)
}

func (a *app) contentHeight(terminalHeight int) int {
	if a.distractionFree && !a.searching {
		return terminalHeight
	}
	return terminalHeight - 1
}

func (a *app) diffViewportHeight(terminalHeight int) int {
	return max(1, a.contentHeight(terminalHeight)-1)
}

func (a *app) draw() {
	a.screen.Clear()
	a.screen.HideCursor()
	width, height := a.screen.Size()
	if width <= 0 || height <= 0 {
		return
	}

	if width < 24 || height < 3 {
		drawText(a.screen, 0, 0, width, 0, "Terminal too small", tcell.StyleDefault)
		a.screen.Show()
		return
	}

	contentHeight := a.contentHeight(height)
	sidebarWidth := 0
	if a.showSidebar && !a.distractionFree {
		sidebarWidth = fileSidebarWidth(width)
	}
	contentX := 0
	contentWidth := width
	if sidebarWidth > 0 {
		contentX = sidebarWidth + 1
		contentWidth = width - contentX
		a.drawSidebar(sidebarWidth, contentHeight)
	}
	lineNumberWidth := a.lineNumberWidth()
	leftWidth := contentWidth
	dividerX := contentX + contentWidth
	rightStart := dividerX
	rightWidth := 0
	if a.layout == layoutSplit {
		leftWidth = contentWidth / 2
		dividerX = contentX + leftWidth
		rightStart = dividerX + 1
		rightWidth = width - rightStart
	}
	viewportHeight := max(0, contentHeight-1)
	a.drawPaneLabels(contentX, contentWidth, leftWidth, rightStart, rightWidth)

	if len(a.rows) == 0 {
		message := "No changes"
		drawText(a.screen, contentX+max(0, (contentWidth-len(message))/2), 1+viewportHeight/2, contentWidth, 0, message, styleMuted)
	} else {
		for screenY := 0; screenY < viewportHeight; screenY++ {
			rowIndex := a.vertical + screenY
			if rowIndex >= len(a.rows) {
				break
			}
			a.drawRow(a.rows[rowIndex], screenY+1, contentX, contentWidth, leftWidth, rightStart, rightWidth, lineNumberWidth)
		}
	}

	for y := 0; y < contentHeight; y++ {
		if a.layout == layoutSplit {
			showPaneDivider := true
			if y > 0 {
				rowIndex := a.vertical + y - 1
				showPaneDivider = rowIndex >= len(a.rows) || a.rows[rowIndex].kind != rowFile
			}
			if showPaneDivider {
				a.screen.SetContent(dividerX, y, '│', nil, styleDivider)
			}
		}
		if sidebarWidth > 0 {
			a.screen.SetContent(sidebarWidth, y, '│', nil, styleDivider)
		}
	}
	if !a.distractionFree || a.searching {
		a.drawStatus(width, height-1)
	}
	if a.shortcutsOpen {
		a.drawShortcuts(width, height)
	}
	a.screen.Show()
}

func (a *app) drawPaneLabels(contentX, contentWidth, leftWidth, rightStart, rightWidth int) {
	if a.layout == layoutUnified {
		fill(a.screen, contentX, 0, contentWidth, stylePaneHeader)
		drawText(a.screen, contentX+2, 0, max(0, contentWidth-2), 0, "±  UNIFIED", styleUnifiedLabel)
		a.drawSingleFileIndicator(contentX, contentWidth)
		return
	}
	fill(a.screen, contentX, 0, leftWidth, stylePaneHeader)
	fill(a.screen, rightStart, 0, rightWidth, stylePaneHeader)
	drawText(a.screen, contentX+2, 0, max(0, leftWidth-2), 0, "−  BEFORE", styleBeforeLabel)
	drawText(a.screen, rightStart+2, 0, max(0, rightWidth-2), 0, "+  AFTER", styleAfterLabel)
	a.drawSingleFileIndicator(contentX, contentWidth)
}

func (a *app) drawSingleFileIndicator(contentX, contentWidth int) {
	position, total := a.singleFilePosition()
	if position == 0 {
		return
	}
	mode := fmt.Sprintf("SINGLE FILE · %d/%d", position, total)
	modeWidth := uniseg.StringWidth(mode)
	modeX := contentX + contentWidth - modeWidth - 1
	minimumX := contentX + 12
	if modeX < minimumX {
		mode = fmt.Sprintf("FILE %d/%d", position, total)
		modeWidth = uniseg.StringWidth(mode)
		modeX = contentX + contentWidth - modeWidth - 1
	}
	if modeX >= minimumX {
		drawText(a.screen, modeX, 0, modeWidth, 0, mode, styleSingleFile)
	}
}

func (a *app) singleFilePosition() (int, int) {
	if a.singleFilePath == "" {
		return 0, 0
	}
	files := changedFiles(a.fullRows())
	fileIndex := changedFileIndex(files, a.singleFilePath)
	if fileIndex < 0 {
		return 0, len(files)
	}
	return filePosition(sidebarFileOrder(files), fileIndex) + 1, len(files)
}

func (a *app) drawRow(row diffRow, y, contentX, contentWidth, leftWidth, rightStart, rightWidth, lineNumberWidth int) {
	if a.layout == layoutUnified {
		a.drawUnifiedRow(row, y, contentX, contentWidth, lineNumberWidth)
		return
	}
	switch row.kind {
	case rowFile:
		a.drawFileBanner(row, y, contentX, contentWidth)
	case rowHunk:
		fill(a.screen, contentX, y, leftWidth, styleHunk)
		fill(a.screen, rightStart, y, rightWidth, styleHunk)
		drawSearchText(a.screen, contentX+2, y, max(0, leftWidth-2), 0, row.left.text, styleHunk, a.searchRegex)
		drawSearchText(a.screen, rightStart+2, y, max(0, rightWidth-2), 0, row.right.text, styleHunk, a.searchRegex)
	case rowMeta:
		fill(a.screen, contentX, y, contentWidth, styleMeta)
		drawSearchText(a.screen, contentX+2, y, contentWidth-2, 0, row.text, styleMeta, a.searchRegex)
	default:
		a.drawCell(row.left, contentX, y, leftWidth, lineNumberWidth)
		a.drawCell(row.right, rightStart, y, rightWidth, lineNumberWidth)
	}
}

func (a *app) drawUnifiedRow(row diffRow, y, x, width, lineNumberWidth int) {
	switch row.kind {
	case rowFile:
		a.drawFileBanner(row, y, x, width)
	case rowHunk:
		fill(a.screen, x, y, width, styleHunk)
		drawSearchText(a.screen, x+2, y, max(0, width-2), 0, row.text, styleHunk, a.searchRegex)
	case rowMeta:
		fill(a.screen, x, y, width, styleMeta)
		drawSearchText(a.screen, x+2, y, max(0, width-2), 0, row.text, styleMeta, a.searchRegex)
	default:
		a.drawUnifiedCell(row, x, y, width, lineNumberWidth)
	}
}

func (a *app) drawUnifiedCell(row diffRow, x, y, width, lineNumberWidth int) {
	cell := row.left
	if cell.kind == cellEmpty {
		cell = row.right
	}
	style := styleForCell(cell.kind)
	fill(a.screen, x, y, width, style)

	oldLine := row.left.line
	newLine := row.right.line
	marker := " "
	if cell.kind == cellDeletion {
		marker = "−"
	} else if cell.kind == cellAddition {
		marker = "+"
	}
	secondNumberX := x + lineNumberWidth + 1
	markerX := secondNumberX + lineNumberWidth + 1
	gutterStyle := style.Background(styleLineNumberBackground).Foreground(styleLineNumberForeground)
	fill(a.screen, x, y, min(markerX-x, width), gutterStyle)
	if oldLine > 0 {
		value := strconv.Itoa(oldLine)
		drawText(a.screen, x+lineNumberWidth-len(value), y, lineNumberWidth, 0, value, gutterStyle)
	}
	if newLine > 0 {
		value := strconv.Itoa(newLine)
		drawText(a.screen, secondNumberX+lineNumberWidth-len(value), y, lineNumberWidth, 0, value, gutterStyle)
	}
	if width > markerX-x {
		drawText(a.screen, markerX, y, 1, 0, marker, style.Bold(cell.kind == cellAddition || cell.kind == cellDeletion))
	}
	contentStart := markerX - x + 2
	if width > contentStart {
		drawSearchText(a.screen, x+contentStart, y, width-contentStart, a.horizontal, expandTabs(cell.text, 4), style, a.searchRegex)
	}
}

func (a *app) drawFileBanner(row diffRow, y, x, width int) {
	path, status := changedFilePathAndStatus(row)
	viewed := a.fileIsViewed(path)
	style, background := fileBannerStyle(path == a.currentFilePath())
	fill(a.screen, x, y, width, style)

	drawText(a.screen, x+1, y, 4, 0, "FILE", styleFileEyebrow.Background(background))
	drawText(a.screen, x+7, y, 3, 0, "["+status+"]", fileStatusBadgeStyle(status))

	review := "○"
	if viewed {
		review = "✓"
	}
	if width >= 32 {
		if viewed {
			review += " Viewed"
		} else {
			review += " Unviewed"
		}
	}
	reviewWidth := uniseg.StringWidth(review)
	reviewX := x + width - reviewWidth - 1
	drawText(a.screen, reviewX, y, reviewWidth, 0, review, reviewStyle(viewed, background))

	change := fileStatusLabel(status)
	changeWidth := uniseg.StringWidth(change)
	changeX := reviewX - changeWidth - 3
	pathEnd := reviewX - 2
	if width >= 72 && changeX > x+8 {
		drawText(a.screen, changeX, y, changeWidth, 0, change, styleFileDetail.Background(background))
		drawText(a.screen, reviewX-2, y, 1, 0, "·", styleFileDetail.Background(background))
		pathEnd = changeX - 2
	}
	pathText := fileBannerPath(row, status)
	drawSearchText(a.screen, x+12, y, max(0, pathEnd-x-12), 0, pathText, style, a.searchRegex)
}

func fileBannerPath(row diffRow, status string) string {
	if status == "R" {
		return row.left.text + " → " + row.right.text
	}
	path, _ := changedFilePathAndStatus(row)
	return path
}

func fileStatusLabel(status string) string {
	switch status {
	case "A":
		return "Added"
	case "D":
		return "Deleted"
	case "R":
		return "Renamed"
	default:
		return "Modified"
	}
}

func (a *app) drawSidebar(width, height int) {
	fillArea(a.screen, 0, 0, width, height, styleSidebar)
	files := changedFiles(a.fullRows())
	fill(a.screen, 0, 0, width, styleSidebarHeader)
	viewedCount := 0
	for _, file := range files {
		if a.fileIsViewed(file.path) {
			viewedCount++
		}
	}
	drawText(a.screen, 1, 0, max(0, width-1), 0, fmt.Sprintf("FILES %d · %d VIEWED", len(files), viewedCount), styleSidebarHeader)
	if len(files) == 0 || height <= 1 {
		return
	}

	current := min(a.currentFile, len(files)-1)
	if a.singleFilePath != "" {
		current = changedFileIndex(files, a.singleFilePath)
	}
	lines := makeSidebarLines(files, current)
	currentLine := 0
	for index, line := range lines {
		if line.fileIndex == current {
			currentLine = index
			break
		}
	}
	available := height - 1
	start := min(max(currentLine-available/2, 0), max(0, len(lines)-available))
	for offset := 0; offset < available && start+offset < len(lines); offset++ {
		line := lines[start+offset]
		y := offset + 1
		if line.directory {
			icon := closedFolderIcon
			if line.expanded {
				icon = openFolderIcon
			}
			viewed, total := a.directoryViewedProgress(files, strings.TrimSuffix(line.text, "/"))
			progress := fmt.Sprintf("%d/%d", viewed, total)
			progressWidth := uniseg.StringWidth(progress)
			drawText(a.screen, 1, y, max(0, width-progressWidth-2), 0, icon+" "+line.text, styleSidebarDirectory)
			drawText(a.screen, max(1, width-progressWidth-1), y, progressWidth, 0, progress, sidebarViewedStyle(viewed == total, false))
			continue
		}

		active := line.fileIndex == current
		style := styleSidebar
		if active {
			style = styleSidebarActive
			fill(a.screen, 0, y, width, style)
		}
		file := files[line.fileIndex]
		statusX := 1
		nameX := 3
		if file.directory != "" {
			statusX = 3
			nameX = 5
		}
		drawText(a.screen, statusX, y, 1, 0, file.status, sidebarStatusStyle(file.status, active))
		drawText(a.screen, nameX, y, max(0, width-nameX-3), 0, line.text, style)
		viewed := a.fileIsViewed(file.path)
		marker := "○"
		if viewed {
			marker = "✓"
		}
		drawText(a.screen, max(0, width-2), y, 1, 0, marker, sidebarViewedStyle(viewed, active))
	}
}

func (a *app) directoryViewedProgress(files []changedFile, directory string) (int, int) {
	viewed := 0
	total := 0
	for _, file := range files {
		if file.directory != directory {
			continue
		}
		total++
		if a.fileIsViewed(file.path) {
			viewed++
		}
	}
	return viewed, total
}

func (a *app) drawCell(cell diffCell, x, y, width, lineNumberWidth int) {
	style := styleForCell(cell.kind)
	fill(a.screen, x, y, width, style)

	gutterWidth := lineNumberWidth + 1
	gutterStyle := style.Background(styleLineNumberBackground).Foreground(styleLineNumberForeground)
	fill(a.screen, x, y, min(gutterWidth, width), gutterStyle)

	if cell.line > 0 {
		lineNumber := strconv.Itoa(cell.line)
		drawText(a.screen, x+lineNumberWidth-len(lineNumber), y, lineNumberWidth, 0, lineNumber, gutterStyle)
	}
	marker := " "
	switch cell.kind {
	case cellDeletion:
		marker = "−"
	case cellAddition:
		marker = "+"
	}
	if width > gutterWidth {
		drawText(a.screen, x+gutterWidth, y, 1, 0, marker, style.Bold(cell.kind == cellAddition || cell.kind == cellDeletion))
	}
	contentStart := gutterWidth + 2
	if width > contentStart {
		drawSearchText(a.screen, x+contentStart, y, width-contentStart, a.horizontal, expandTabs(cell.text, 4), style, a.searchRegex)
	}
}

func (a *app) drawStatus(width, y int) {
	fill(a.screen, 0, y, width, styleStatus)
	if a.searching {
		a.drawSearchStatus(width, y)
		return
	}

	navigation := "hunk"
	if a.searchRegex != nil {
		navigation = "match"
	}
	fileMode := "Enter focus"
	if a.singleFilePath != "" {
		fileMode = "Enter all"
	}
	status := fmt.Sprintf("j/k scroll  [/] file  %s  s layout  v viewed  / search  ? shortcuts", fileMode)
	if width >= 120 {
		status = fmt.Sprintf("j/k scroll  [/] file  n/N %s  %s  s layout  v viewed  / search  z zen  ? shortcuts", navigation, fileMode)
	}
	style := styleStatus
	if a.status != "" {
		status = "✓ " + a.status + "   " + status
	}
	if a.statusIsError {
		status = "! " + a.status
		style = styleStatusError
	}

	position := a.positionSummary()
	positionWidth := uniseg.StringWidth(position)
	drawText(a.screen, 1, y, max(0, width-positionWidth-2), 0, status, style)
	drawText(a.screen, max(0, width-positionWidth), y, positionWidth, 0, position, styleStatus)
}

func (a *app) drawSearchStatus(width, y int) {
	prompt := "/" + a.searchQuery
	context := "  Enter apply · Esc cancel"
	style := styleStatus
	if a.searchError != "" {
		context = "  invalid regex"
		style = styleStatusError
	} else if a.searchQuery != "" {
		context = fmt.Sprintf("  %d matching rows", len(a.searchMatches))
	}
	drawText(a.screen, 1, y, max(0, width-1), 0, prompt+context, style)
	cursorX := min(width-1, 1+uniseg.StringWidth(prompt))
	a.screen.ShowCursor(cursorX, y)
}

func (a *app) positionSummary() string {
	files := changedFiles(a.fullRows())
	fileCount := len(files)
	match := ""
	if a.searchRegex != nil {
		match = fmt.Sprintf(" match %d/%d ·", a.searchCurrent+1, len(a.searchMatches))
	}
	if fileCount == 0 {
		return match + " 0 files "
	}
	fileIndex := a.currentFile
	if a.singleFilePath != "" {
		fileIndex = changedFileIndex(files, a.singleFilePath)
	}
	viewed := "○ unviewed"
	if fileIndex >= 0 && fileIndex < fileCount && a.fileIsViewed(files[fileIndex].path) {
		viewed = "✓ viewed"
	}
	current := filePosition(sidebarFileOrder(files), fileIndex) + 1
	current = min(max(current, 1), fileCount)
	mode := "file"
	if a.singleFilePath != "" {
		mode = "single file"
	}
	return fmt.Sprintf("%s %s · %s %d/%d · row %d/%d ", match, viewed, mode, current, fileCount, min(a.vertical+1, len(a.rows)), len(a.rows))
}

func (a *app) drawShortcuts(width, height int) {
	items := []struct {
		key    string
		action string
	}{
		{"Enter", "single file / all files"},
		{"v", "toggle file viewed"},
		{"[ / ]", "previous / next file"},
		{"j / k", "scroll"},
		{"h / l", "pan"},
		{"n / N", "match or hunk"},
		{"/", "search"},
		{"f", "toggle files"},
		{"s", "split / unified layout"},
		{"z", "distraction-free"},
		{"r", "refresh"},
		{"q", "quit"},
		{"? / Esc", "close shortcuts"},
	}
	boxWidth := min(44, width-2)
	boxHeight := min(len(items)+3, height-2)
	if boxWidth < 18 || boxHeight < 3 {
		return
	}
	x := (width - boxWidth) / 2
	y := (height - boxHeight) / 2
	fillArea(a.screen, x, y, boxWidth, boxHeight, styleMenu)
	for column := 1; column < boxWidth-1; column++ {
		a.screen.SetContent(x+column, y, '─', nil, styleMenuBorder)
		a.screen.SetContent(x+column, y+boxHeight-1, '─', nil, styleMenuBorder)
	}
	for row := 1; row < boxHeight-1; row++ {
		a.screen.SetContent(x, y+row, '│', nil, styleMenuBorder)
		a.screen.SetContent(x+boxWidth-1, y+row, '│', nil, styleMenuBorder)
	}
	a.screen.SetContent(x, y, '┌', nil, styleMenuBorder)
	a.screen.SetContent(x+boxWidth-1, y, '┐', nil, styleMenuBorder)
	a.screen.SetContent(x, y+boxHeight-1, '└', nil, styleMenuBorder)
	a.screen.SetContent(x+boxWidth-1, y+boxHeight-1, '┘', nil, styleMenuBorder)
	drawText(a.screen, x+2, y+1, boxWidth-4, 0, "SHORTCUTS", styleMenuTitle)
	for index, item := range items {
		row := y + index + 2
		if row >= y+boxHeight-1 {
			break
		}
		drawText(a.screen, x+2, row, 9, 0, item.key, styleMenuKey)
		drawText(a.screen, x+12, row, max(0, boxWidth-14), 0, item.action, styleMenu)
	}
}

func (a *app) lineNumberWidth() int {
	maximum := 1
	for _, row := range a.rows {
		maximum = max(maximum, row.left.line, row.right.line)
	}
	return max(3, len(strconv.Itoa(maximum)))
}

func fill(screen tcell.Screen, x, y, width int, style tcell.Style) {
	for offset := 0; offset < width; offset++ {
		screen.SetContent(x+offset, y, ' ', nil, style)
	}
}

func fillArea(screen tcell.Screen, x, y, width, height int, style tcell.Style) {
	for row := 0; row < height; row++ {
		fill(screen, x, y+row, width, style)
	}
}

func drawText(screen tcell.Screen, x, y, width, horizontalOffset int, value string, style tcell.Style) {
	drawSearchText(screen, x, y, width, horizontalOffset, value, style, nil)
}

func drawSearchText(screen tcell.Screen, x, y, width, horizontalOffset int, value string, style tcell.Style, expression *regexp.Regexp) {
	if width <= 0 {
		return
	}

	matches := [][2]int(nil)
	if expression != nil {
		for _, match := range expression.FindAllStringIndex(value, -1) {
			matches = append(matches, [2]int{match[0], match[1]})
		}
	}
	graphemes := uniseg.NewGraphemes(value)
	consumed := 0
	drawn := 0
	for graphemes.Next() {
		start, end := graphemes.Positions()
		clusterWidth := graphemes.Width()
		if consumed+clusterWidth <= horizontalOffset {
			consumed += clusterWidth
			continue
		}
		if consumed < horizontalOffset || drawn+clusterWidth > width {
			consumed += clusterWidth
			continue
		}

		runes := []rune(graphemes.Str())
		if len(runes) == 0 {
			continue
		}
		characterStyle := style
		for _, match := range matches {
			if start < match[1] && end > match[0] {
				characterStyle = styleSearchMatch
				break
			}
		}
		screen.SetContent(x+drawn, y, runes[0], runes[1:], characterStyle)
		drawn += clusterWidth
		consumed += clusterWidth
	}
}

func expandTabs(value string, tabWidth int) string {
	var result strings.Builder
	column := 0
	for _, character := range value {
		if character != '\t' {
			result.WriteRune(character)
			column++
			continue
		}
		spaces := tabWidth - column%tabWidth
		result.WriteString(strings.Repeat(" ", spaces))
		column += spaces
	}
	return result.String()
}
