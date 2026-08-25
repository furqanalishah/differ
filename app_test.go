package main

import (
	"errors"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/vt"
)

func TestDrawShowsOrientedSideBySideDiff(t *testing.T) {
	screen := newTestScreen(t, 80, 8)
	defer screen.Fini()

	rows := parseUnifiedDiff(`diff --git a/file.txt b/file.txt
@@ -1 +1 @@
-before
+after`)
	application := newApp(screen, rows, func() ([]diffRow, error) { return rows, nil }, 0)
	application.draw()

	width, height := screen.Size()
	lines := screenLines(screen, width, height)
	if !strings.Contains(lines[0], "BEFORE") || !strings.Contains(lines[0], "AFTER") {
		t.Fatalf("expected pane orientation labels, got %q", lines[0])
	}
	if strings.Count(lines[1], "file.txt") != 1 || !strings.Contains(lines[1], "FILE  [M]") || !strings.Contains(lines[1], "Modified") || !strings.Contains(lines[1], "Unviewed") {
		t.Fatalf("expected one clear file banner, got %q", lines[1])
	}
	if strings.ContainsRune(lines[1], '│') {
		t.Fatalf("file banner should span both panes, got %q", lines[1])
	}
	if !strings.Contains(lines[2], "-1") || !strings.Contains(lines[2], "+1") {
		t.Fatalf("expected split hunk ranges, got %q", lines[2])
	}
	if !strings.Contains(lines[3], "before") || !strings.Contains(lines[3], "after") {
		t.Fatalf("expected both sides on one row, got %q", lines[3])
	}
	if !strings.Contains(lines[3], "−") || !strings.Contains(lines[3], "+") {
		t.Fatalf("expected accessible change markers, got %q", lines[3])
	}
	if lines[3][width/2] != '|' && !strings.Contains(lines[3], "│") {
		t.Fatalf("expected center divider, got %q", lines[3])
	}
	if !strings.Contains(lines[height-1], "file 1/1") {
		t.Fatalf("expected file progress in status, got %q", lines[height-1])
	}
}

func TestUnifiedLayoutUsesOneFullWidthDiffColumn(t *testing.T) {
	screen := newTestScreen(t, 100, 9)
	defer screen.Fini()

	rows := parseUnifiedDiff(`diff --git a/file.txt b/file.txt
@@ -1 +1 @@
-before
+after`)
	application := newApp(screen, rows, func() ([]diffRow, error) { return rows, nil }, 0)

	if err := application.handleKey(tcell.NewEventKey(tcell.KeyRune, "s", tcell.ModNone)); err != nil {
		t.Fatal(err)
	}
	application.draw()
	lines := screenLines(screen, 100, 9)
	if application.layout != layoutUnified || !strings.Contains(lines[0], "UNIFIED") {
		t.Fatalf("expected a clear unified layout guide, got %q", lines[0])
	}
	if strings.Contains(lines[0], "BEFORE") || strings.Contains(lines[0], "AFTER") {
		t.Fatalf("unified layout should not retain split pane labels, got %q", lines[0])
	}
	if !strings.Contains(lines[2], "@@ -1 +1 @@") {
		t.Fatalf("expected the complete unified hunk range, got %q", lines[2])
	}
	if !strings.Contains(lines[3], "before") || strings.Contains(lines[3], "after") || !strings.Contains(lines[3], "−") {
		t.Fatalf("expected a standalone deletion row, got %q", lines[3])
	}
	if !strings.Contains(lines[4], "after") || strings.Contains(lines[4], "before") || !strings.Contains(lines[4], "+") {
		t.Fatalf("expected a standalone addition row, got %q", lines[4])
	}
	if strings.ContainsRune(lines[3], '│') || strings.ContainsRune(lines[4], '│') {
		t.Fatal("unified rows should use the full width without a pane divider")
	}

	if err := application.handleKey(tcell.NewEventKey(tcell.KeyRune, "s", tcell.ModNone)); err != nil {
		t.Fatal(err)
	}
	application.draw()
	lines = screenLines(screen, 100, 9)
	if application.layout != layoutSplit || !strings.Contains(lines[0], "BEFORE") || !strings.Contains(lines[0], "AFTER") {
		t.Fatalf("expected s to restore split layout, got %q", lines[0])
	}
}

func TestUnifiedLayoutCombinesWithSingleFileAndViewedState(t *testing.T) {
	screen := newTestScreen(t, 120, 9)
	defer screen.Fini()

	rows := parseUnifiedDiff(`diff --git a/one.txt b/one.txt
@@ -1 +1 @@
-old one
+new one
diff --git a/two.txt b/two.txt
@@ -1 +1 @@
-old two
+new two`)
	application := newApp(screen, rows, func() ([]diffRow, error) { return rows, nil }, 0)
	application.jumpToFile(true)
	application.toggleSingleFileMode()
	application.toggleLayout()
	application.toggleCurrentFileViewed()
	application.refresh(false)
	application.draw()

	lines := screenLines(screen, 120, 9)
	if application.layout != layoutUnified || application.singleFilePath != "two.txt" || len(changedFiles(application.rows)) != 1 {
		t.Fatal("unified layout should preserve single-file focus")
	}
	if !application.fileIsViewed("two.txt") {
		t.Fatal("layout changes and refresh should preserve an unchanged viewed mark")
	}
	if !strings.Contains(lines[0], "UNIFIED") || !strings.Contains(lines[0], "SINGLE FILE · 2/2") {
		t.Fatalf("expected both layout and file-scope indicators, got %q", lines[0])
	}
}

func TestFileBannersExplainEveryGitStatus(t *testing.T) {
	screen := newTestScreen(t, 120, 8)
	defer screen.Fini()

	rows := []diffRow{
		{kind: rowFile, left: diffCell{text: "README.md"}, right: diffCell{text: "README.md"}},
		{kind: rowFile, left: diffCell{text: "∅"}, right: diffCell{text: "new.go"}},
		{kind: rowFile, left: diffCell{text: "old.txt"}, right: diffCell{text: "∅"}},
		{kind: rowFile, left: diffCell{text: "old-name.md"}, right: diffCell{text: "new-name.md"}},
	}
	application := newApp(screen, rows, func() ([]diffRow, error) { return rows, nil }, 0)
	application.draw()
	lines := screenLines(screen, 120, 8)

	wants := []struct {
		line   int
		status string
		path   string
		label  string
	}{
		{1, "M", "README.md", "Modified"},
		{2, "A", "new.go", "Added"},
		{3, "D", "old.txt", "Deleted"},
		{4, "R", "old-name.md → new-name.md", "Renamed"},
	}
	for _, want := range wants {
		line := lines[want.line]
		if !strings.Contains(line, "FILE  ["+want.status+"]") || !strings.Contains(line, want.path) || !strings.Contains(line, want.label) || !strings.Contains(line, "Unviewed") {
			t.Fatalf("expected an explicit %s file banner, got %q", want.label, line)
		}
		if strings.ContainsRune(line, '│') {
			t.Fatalf("file banner should not be split into repeated panes, got %q", line)
		}
	}
	badgeStyles := make(map[tcell.Style]bool)
	for line := 1; line <= 4; line++ {
		_, style, _ := screen.Get(8, line)
		badgeStyles[style] = true
	}
	if len(badgeStyles) != 4 {
		t.Fatal("each Git status badge should have a distinct visual treatment")
	}
	_, firstBannerStyle, _ := screen.Get(20, 1)
	_, bannerStyle, _ := screen.Get(20, 2)
	if firstBannerStyle.GetBackground() != colorFileBannerActive || bannerStyle.GetBackground() != colorFileBanner {
		t.Fatal("the focused file should use the active banner background")
	}
	activeRed, activeGreen, activeBlue := colorFileBannerActive.RGB()
	inactiveRed, inactiveGreen, inactiveBlue := colorFileBanner.RGB()
	if activeRed+activeGreen+activeBlue <= inactiveRed+inactiveGreen+inactiveBlue {
		t.Fatal("the active banner background should be brighter than inactive banners")
	}
	if bannerStyle.GetBackground() == styleHunk.GetBackground() {
		t.Fatal("file banners should remain visually separate from hunk rows")
	}

	if err := application.handleKey(tcell.NewEventKey(tcell.KeyRune, "]", tcell.ModNone)); err != nil {
		t.Fatal(err)
	}
	application.draw()
	_, firstBannerStyle, _ = screen.Get(20, 1)
	_, bannerStyle, _ = screen.Get(20, 2)
	if firstBannerStyle.GetBackground() != colorFileBanner || bannerStyle.GetBackground() != colorFileBannerActive {
		t.Fatal("the brighter banner should follow file focus")
	}
}

func TestFileSidebarTogglesWithoutTakingFocus(t *testing.T) {
	screen := newTestScreen(t, 120, 5)
	defer screen.Fini()

	rows := parseUnifiedDiff(`diff --git a/README.md b/README.md
@@ -1 +1 @@
-before
+after
diff --git a/src/app.go b/src/app.go
@@ -1 +1 @@
-old
+new`)
	application := newApp(screen, rows, func() ([]diffRow, error) { return rows, nil }, 0)

	if err := application.handleKey(tcell.NewEventKey(tcell.KeyRune, "f", tcell.ModNone)); err != nil {
		t.Fatal(err)
	}
	application.draw()
	lines := screenLines(screen, 120, 5)
	if !strings.Contains(lines[0], "FILES 2") || !strings.Contains(lines[0], "BEFORE") || !strings.Contains(lines[0], "AFTER") {
		t.Fatalf("expected file list beside the diff, got %q", lines[0])
	}
	if !strings.Contains(strings.Join(lines, "\n"), "src/") {
		t.Fatal("expected directory grouping in the file list")
	}
	if !strings.Contains(lines[2], closedFolderIcon) {
		t.Fatal("expected an unfocused directory to be closed")
	}
	_, activeStyle, _ := screen.Get(3, 1)
	if activeStyle != styleSidebarActive {
		t.Fatal("expected the current file to be highlighted")
	}

	if err := application.handleKey(tcell.NewEventKey(tcell.KeyRune, "]", tcell.ModNone)); err != nil {
		t.Fatal(err)
	}
	if application.vertical == 0 {
		t.Fatal("file navigation should continue working while the sidebar is visible")
	}
	application.draw()
	lines = screenLines(screen, 120, 5)
	if !strings.Contains(lines[2], openFolderIcon) {
		t.Fatal("expected the focused directory to open")
	}
	if !strings.HasPrefix(lines[3], "   M app.go") {
		t.Fatalf("expected the focused directory file to be indented, got %q", lines[3])
	}
	if err := application.handleKey(tcell.NewEventKey(tcell.KeyRune, "f", tcell.ModNone)); err != nil {
		t.Fatal(err)
	}
	if application.showSidebar {
		t.Fatal("second f press should hide the file list")
	}
}

func TestFileSidebarDoesNotOpenInNarrowTerminal(t *testing.T) {
	screen := newTestScreen(t, 60, 8)
	defer screen.Fini()

	application := newApp(screen, nil, func() ([]diffRow, error) { return nil, nil }, 0)
	if err := application.handleKey(tcell.NewEventKey(tcell.KeyRune, "f", tcell.ModNone)); err != nil {
		t.Fatal(err)
	}
	if application.showSidebar || !application.statusIsError {
		t.Fatal("narrow terminals should keep the file list hidden and show feedback")
	}
}

func TestFileNavigationTracksSelectionWhenTheWholeDiffFits(t *testing.T) {
	screen := newTestScreen(t, 120, 10)
	defer screen.Fini()

	rows := parseUnifiedDiff(`diff --git a/README.md b/README.md
@@ -1 +1 @@
-before
+after
diff --git a/src/app.go b/src/app.go
@@ -1 +1 @@
-old
+new`)
	application := newApp(screen, rows, func() ([]diffRow, error) { return rows, nil }, 0)
	application.showSidebar = true

	if err := application.handleKey(tcell.NewEventKey(tcell.KeyRune, "]", tcell.ModNone)); err != nil {
		t.Fatal(err)
	}
	if application.currentFile != 1 {
		t.Fatalf("expected second file to remain selected, got %d", application.currentFile)
	}
	application.draw()
	_, activeStyle, _ := screen.Get(5, 3)
	if activeStyle != styleSidebarActive {
		t.Fatal("expected the second file to stay highlighted when scrolling is clamped")
	}
}

func TestSingleFileModeScopesReviewAndKeepsFileNavigation(t *testing.T) {
	screen := newTestScreen(t, 120, 10)
	defer screen.Fini()

	rows := parseUnifiedDiff(`diff --git a/README.md b/README.md
@@ -1 +1 @@
-old readme
+focused readme
diff --git a/src/app.go b/src/app.go
@@ -1 +1 @@
-old app
+focused app`)
	application := newApp(screen, rows, func() ([]diffRow, error) { return rows, nil }, 0)

	if err := application.handleKey(tcell.NewEventKey(tcell.KeyEnter, "", tcell.ModNone)); err != nil {
		t.Fatal(err)
	}
	if application.singleFilePath != "README.md" {
		t.Fatalf("expected README.md to open alone, got %q", application.singleFilePath)
	}
	files := changedFiles(application.rows)
	if len(files) != 1 || files[0].path != "README.md" {
		t.Fatalf("expected only README.md rows, got %#v", files)
	}
	application.setSearchQuery("focused app")
	if len(application.searchMatches) != 0 {
		t.Fatal("search should be scoped to the focused file")
	}
	application.draw()
	lines := screenLines(screen, 120, 10)
	if !strings.Contains(lines[0], "SINGLE FILE · 1/2") {
		t.Fatalf("expected a persistent single-file indicator, got %q", lines[0])
	}

	if err := application.handleKey(tcell.NewEventKey(tcell.KeyRune, "]", tcell.ModNone)); err != nil {
		t.Fatal(err)
	}
	if application.singleFilePath != "src/app.go" || len(application.searchMatches) != 1 {
		t.Fatal("next-file navigation should stay focused and rebuild scoped search")
	}
	application.showSidebar = true
	application.draw()
	lines = screenLines(screen, 120, 10)
	if !strings.Contains(lines[0], "FILES 2") || !strings.Contains(lines[0], "SINGLE FILE · 2/2") {
		t.Fatalf("expected all-file navigation context around the focused diff, got %q", lines[0])
	}

	if err := application.handleKey(tcell.NewEventKey(tcell.KeyEscape, "", tcell.ModNone)); err != nil {
		t.Fatalf("escape should return to all files: %v", err)
	}
	if application.singleFilePath != "" || len(changedFiles(application.rows)) != 2 || application.currentFile != 1 {
		t.Fatal("leaving single-file mode should restore the complete diff and selection")
	}
}

func TestSingleFileModeSurvivesRefreshAndExitsWhenFileDisappears(t *testing.T) {
	screen := newTestScreen(t, 100, 8)
	defer screen.Fini()

	initial := parseUnifiedDiff(`diff --git a/one.txt b/one.txt
@@ -1 +1 @@
-old one
+new one
diff --git a/two.txt b/two.txt
@@ -1 +1 @@
-old two
+new two`)
	nextRows := parseUnifiedDiff(`diff --git a/one.txt b/one.txt
@@ -1 +1 @@
-old one
+new one
diff --git a/two.txt b/two.txt
@@ -1 +1 @@
-old two
+newer two`)
	application := newApp(screen, initial, func() ([]diffRow, error) { return nextRows, nil }, 0)
	application.jumpToFile(true)
	application.toggleSingleFileMode()

	if !application.refresh(false) || application.singleFilePath != "two.txt" {
		t.Fatal("refresh should preserve an existing focused file")
	}
	if !strings.Contains(strings.Join(rowSearchValues(application.rows[len(application.rows)-1]), " "), "newer two") {
		t.Fatal("focused rows should update after refresh")
	}

	nextRows = parseUnifiedDiff(`diff --git a/one.txt b/one.txt
@@ -1 +1 @@
-old one
+new one`)
	if !application.refresh(false) {
		t.Fatal("removing the focused file should redraw")
	}
	if application.singleFilePath != "" || len(changedFiles(application.rows)) != 1 {
		t.Fatal("a file with no remaining changes should return to the complete diff")
	}
	if application.status != "focused file no longer has changes" {
		t.Fatalf("expected clear focus-exit feedback, got %q", application.status)
	}
}

func TestViewedMarkersAreVisibleOnFilesDirectoriesAndTheDiff(t *testing.T) {
	screen := newTestScreen(t, 120, 8)
	defer screen.Fini()

	rows := parseUnifiedDiff(`diff --git a/README.md b/README.md
@@ -1 +1 @@
-before
+after
diff --git a/src/app.go b/src/app.go
@@ -1 +1 @@
-old
+new`)
	application := newApp(screen, rows, func() ([]diffRow, error) { return rows, nil }, 0)
	application.showSidebar = true

	if err := application.handleKey(tcell.NewEventKey(tcell.KeyRune, "v", tcell.ModNone)); err != nil {
		t.Fatal(err)
	}
	application.draw()
	lines := screenLines(screen, 120, 8)
	if !strings.Contains(lines[0], "FILES 2 · 1 VIEWED") || !strings.Contains(lines[0], "VIEWED") {
		t.Fatalf("expected viewed totals and a viewed diff heading, got %q", lines[0])
	}
	if !strings.Contains(lines[1], "✓") {
		t.Fatalf("expected a viewed marker on README.md, got %q", lines[1])
	}
	if !strings.Contains(lines[2], "0/1") {
		t.Fatalf("expected directory viewed progress, got %q", lines[2])
	}

	if err := application.handleKey(tcell.NewEventKey(tcell.KeyRune, "]", tcell.ModNone)); err != nil {
		t.Fatal(err)
	}
	if err := application.handleKey(tcell.NewEventKey(tcell.KeyRune, "v", tcell.ModNone)); err != nil {
		t.Fatal(err)
	}
	application.draw()
	lines = screenLines(screen, 120, 8)
	if !strings.Contains(lines[2], "1/1") || !strings.Contains(lines[3], "✓") {
		t.Fatalf("expected the expanded directory and its file to show viewed state, got %q / %q", lines[2], lines[3])
	}

	if err := application.handleKey(tcell.NewEventKey(tcell.KeyRune, "v", tcell.ModNone)); err != nil {
		t.Fatal(err)
	}
	if application.fileIsViewed("src/app.go") {
		t.Fatal("second v press should mark the focused file unviewed")
	}
}

func TestViewedMarkClearsOnlyWhenThatFilesDiffChanges(t *testing.T) {
	screen := newTestScreen(t, 80, 8)
	defer screen.Fini()

	initial := parseUnifiedDiff(`diff --git a/one.txt b/one.txt
@@ -1 +1 @@
-old one
+new one
diff --git a/two.txt b/two.txt
@@ -1 +1 @@
-old two
+new two`)
	updated := parseUnifiedDiff(`diff --git a/one.txt b/one.txt
@@ -1 +1 @@
-old one
+new one
diff --git a/two.txt b/two.txt
@@ -1 +1 @@
-old two
+newer two`)
	application := &app{
		rows:        initial,
		viewedFiles: make(map[string][]diffRow),
		reload:      func() ([]diffRow, error) { return updated, nil },
		screen:      screen,
	}
	application.toggleCurrentFileViewed()
	application.jumpToFile(true)
	application.toggleCurrentFileViewed()

	if !application.refresh(false) {
		t.Fatal("changed diff should request a redraw")
	}
	if !application.fileIsViewed("one.txt") {
		t.Fatal("unchanged file should remain viewed")
	}
	if application.fileIsViewed("two.txt") {
		t.Fatal("changed file should automatically become unviewed")
	}
}

func TestShortcutOverlayOpensAndClosesWithoutQuitting(t *testing.T) {
	screen := newTestScreen(t, 80, 16)
	defer screen.Fini()

	application := newApp(screen, nil, func() ([]diffRow, error) { return nil, nil }, 0)
	if err := application.handleKey(tcell.NewEventKey(tcell.KeyRune, "?", tcell.ModNone)); err != nil {
		t.Fatal(err)
	}
	application.draw()
	lines := strings.Join(screenLines(screen, 80, 16), "\n")
	if !strings.Contains(lines, "SHORTCUTS") || !strings.Contains(lines, "toggle file viewed") {
		t.Fatal("expected the shortcut overlay to describe available actions")
	}

	if err := application.handleKey(tcell.NewEventKey(tcell.KeyRune, "q", tcell.ModNone)); err != nil {
		t.Fatalf("q should close the shortcut overlay without quitting: %v", err)
	}
	if application.shortcutsOpen {
		t.Fatal("q should close the shortcut overlay")
	}
	if err := application.handleKey(tcell.NewEventKey(tcell.KeyRune, "q", tcell.ModNone)); !errors.Is(err, errQuit) {
		t.Fatalf("q should quit after the overlay is closed, got %v", err)
	}
}

func TestDistractionFreeModeHidesChromeAndUsesTheFullScreen(t *testing.T) {
	screen := newTestScreen(t, 120, 7)
	defer screen.Fini()

	rows := []diffRow{
		{kind: rowFile, left: diffCell{text: "file.txt"}, right: diffCell{text: "file.txt"}},
		{kind: rowContext, left: diffCell{text: "one"}, right: diffCell{text: "one"}},
		{kind: rowContext, left: diffCell{text: "two"}, right: diffCell{text: "two"}},
		{kind: rowContext, left: diffCell{text: "three"}, right: diffCell{text: "three"}},
		{kind: rowContext, left: diffCell{text: "four"}, right: diffCell{text: "four"}},
		{kind: rowContext, left: diffCell{text: "five"}, right: diffCell{text: "five"}},
	}
	application := newApp(screen, rows, func() ([]diffRow, error) { return rows, nil }, 0)
	application.showSidebar = true

	if err := application.handleKey(tcell.NewEventKey(tcell.KeyRune, "z", tcell.ModNone)); err != nil {
		t.Fatal(err)
	}
	application.draw()
	lines := screenLines(screen, 120, 7)
	joined := strings.Join(lines, "\n")
	if strings.Contains(joined, "FILES") || strings.Contains(joined, "j/k scroll") {
		t.Fatal("focus mode should hide the sidebar and status bar")
	}
	if !strings.Contains(lines[6], "five") {
		t.Fatalf("expected the diff to use the final terminal row below the pane guide, got %q", lines[6])
	}

	if err := application.handleKey(tcell.NewEventKey(tcell.KeyRune, "z", tcell.ModNone)); err != nil {
		t.Fatal(err)
	}
	application.draw()
	lines = screenLines(screen, 120, 7)
	if !strings.Contains(lines[0], "FILES") || !strings.Contains(lines[6], "j/k scroll") {
		t.Fatal("leaving focus mode should restore the sidebar and status bar")
	}
}

func TestSearchInputTemporarilyAppearsInDistractionFreeMode(t *testing.T) {
	screen := newTestScreen(t, 80, 6)
	defer screen.Fini()

	rows := []diffRow{{kind: rowContext, left: diffCell{text: "needle"}}}
	application := newApp(screen, rows, func() ([]diffRow, error) { return rows, nil }, 0)
	application.distractionFree = true
	application.startSearch()
	if err := application.handleSearchKey(tcell.NewEventKey(tcell.KeyRune, "needle", tcell.ModNone)); err != nil {
		t.Fatal(err)
	}
	application.draw()

	lines := screenLines(screen, 80, 6)
	if !strings.Contains(lines[5], "/needle") {
		t.Fatalf("expected search input on the final row, got %q", lines[5])
	}
}

func TestRunRefreshesAutomatically(t *testing.T) {
	screen := newTestScreen(t, 60, 8)
	defer screen.Fini()

	var reloadCount atomic.Int32
	application := newApp(screen, nil, func() ([]diffRow, error) {
		reloadCount.Add(1)
		return nil, nil
	}, 10*time.Millisecond)

	done := make(chan error, 1)
	go func() {
		done <- application.run()
	}()

	deadline := time.NewTimer(time.Second)
	defer deadline.Stop()
	for reloadCount.Load() == 0 {
		select {
		case <-deadline.C:
			t.Fatal("automatic refresh did not run")
		case <-time.After(5 * time.Millisecond):
		}
	}

	screen.EventQ() <- tcell.NewEventKey(tcell.KeyRune, "q", tcell.ModNone)
	select {
	case err := <-done:
		if !errors.Is(err, errQuit) {
			t.Fatalf("expected clean quit, got %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("application did not stop")
	}
}

func TestAutomaticRefreshOnlyRedrawsForChanges(t *testing.T) {
	screen := newTestScreen(t, 60, 8)
	defer screen.Fini()

	rows := []diffRow{{kind: rowFile, left: diffCell{text: "file.txt"}}}
	nextRows := rows
	application := newApp(screen, rows, func() ([]diffRow, error) {
		return nextRows, nil
	}, 0)

	if application.refresh(false) {
		t.Fatal("unchanged automatic refresh should not request a redraw")
	}
	nextRows = append(nextRows, diffRow{kind: rowHunk, text: "@@ -1 +1 @@"})
	if !application.refresh(false) {
		t.Fatal("changed automatic refresh should request a redraw")
	}
	if !application.refresh(true) || application.status != "refreshed" {
		t.Fatal("manual refresh should always redraw and show confirmation")
	}
}

func TestJumpToHunksAndFiles(t *testing.T) {
	rows := []diffRow{
		{kind: rowFile},
		{kind: rowHunk},
		{kind: rowContext},
		{kind: rowHunk},
		{kind: rowFile},
		{kind: rowHunk},
	}
	application := &app{rows: rows}

	application.jumpTo(rowHunk, true)
	if application.vertical != 1 {
		t.Fatalf("expected first hunk at 1, got %d", application.vertical)
	}
	application.jumpTo(rowHunk, true)
	if application.vertical != 3 {
		t.Fatalf("expected next hunk at 3, got %d", application.vertical)
	}
	application.jumpTo(rowFile, true)
	if application.vertical != 4 {
		t.Fatalf("expected next file at 4, got %d", application.vertical)
	}
	application.jumpTo(rowHunk, false)
	if application.vertical != 3 {
		t.Fatalf("expected previous hunk at 3, got %d", application.vertical)
	}
}

func TestRegexSearchAndMatchNavigation(t *testing.T) {
	screen := newTestScreen(t, 60, 3)
	defer screen.Fini()

	rows := []diffRow{
		{kind: rowFile, left: diffCell{text: "first.txt"}},
		{kind: rowContext, left: diffCell{text: "target one"}},
		{kind: rowContext, left: diffCell{text: "no match"}},
		{kind: rowHunk, left: diffCell{text: "-4,2 target hunk"}},
		{kind: rowContext, right: diffCell{text: "target two"}},
		{kind: rowFile, right: diffCell{text: "last.txt"}},
		{kind: rowContext, left: diffCell{text: "no match"}},
		{kind: rowContext, right: diffCell{text: "target three"}},
	}
	application := newApp(screen, rows, func() ([]diffRow, error) { return rows, nil }, 0)

	if err := application.handleKey(tcell.NewEventKey(tcell.KeyRune, "/", tcell.ModNone)); err != nil {
		t.Fatal(err)
	}
	if !application.searching {
		t.Fatal("slash should open search input")
	}
	if err := application.handleKey(tcell.NewEventKey(tcell.KeyRune, `target (one|two|three)`, tcell.ModNone)); err != nil {
		t.Fatal(err)
	}
	if got := len(application.searchMatches); got != 3 {
		t.Fatalf("expected 3 matching rows, got %d", got)
	}
	if application.searchCurrent != 0 {
		t.Fatalf("expected first match to be selected, got %d", application.searchCurrent)
	}
	if err := application.handleKey(tcell.NewEventKey(tcell.KeyEnter, "", tcell.ModNone)); err != nil {
		t.Fatal(err)
	}
	if application.searching {
		t.Fatal("enter should apply a valid search")
	}

	if err := application.handleKey(tcell.NewEventKey(tcell.KeyRune, "n", tcell.ModNone)); err != nil {
		t.Fatal(err)
	}
	if application.searchCurrent != 1 || application.vertical != 4 {
		t.Fatalf("expected next match at row 4, got match %d and row %d", application.searchCurrent, application.vertical)
	}
	if err := application.handleKey(tcell.NewEventKey(tcell.KeyRune, "n", tcell.ModNone)); err != nil {
		t.Fatal(err)
	}
	if application.searchCurrent != 2 {
		t.Fatalf("expected final match to stay selected after viewport clamping, got %d", application.searchCurrent)
	}
	if err := application.handleKey(tcell.NewEventKey(tcell.KeyRune, "N", tcell.ModNone)); err != nil {
		t.Fatal(err)
	}
	if application.searchCurrent != 1 || application.vertical != 4 {
		t.Fatalf("expected previous match at row 4, got match %d and row %d", application.searchCurrent, application.vertical)
	}
}

func TestInvalidRegexStaysOpenUntilFixedOrCancelled(t *testing.T) {
	screen := newTestScreen(t, 60, 5)
	defer screen.Fini()

	rows := []diffRow{{kind: rowContext, left: diffCell{text: "value"}}}
	application := newApp(screen, rows, func() ([]diffRow, error) { return rows, nil }, 0)
	application.startSearch()

	if err := application.handleSearchKey(tcell.NewEventKey(tcell.KeyRune, "[", tcell.ModNone)); err != nil {
		t.Fatal(err)
	}
	if application.searchError == "" {
		t.Fatal("expected invalid regex feedback")
	}
	if err := application.handleSearchKey(tcell.NewEventKey(tcell.KeyEnter, "", tcell.ModNone)); err != nil {
		t.Fatal(err)
	}
	if !application.searching {
		t.Fatal("invalid regex should keep search input open")
	}

	if err := application.handleSearchKey(tcell.NewEventKey(tcell.KeyEscape, "", tcell.ModNone)); err != nil {
		t.Fatal(err)
	}
	if application.searching || application.searchQuery != "" || application.searchError != "" {
		t.Fatal("escape should cancel the search and clear its error")
	}
}

func TestSearchSurvivesAutomaticRefresh(t *testing.T) {
	screen := newTestScreen(t, 60, 5)
	defer screen.Fini()

	rows := []diffRow{{kind: rowContext, left: diffCell{text: "needle"}}}
	application := newApp(screen, rows, func() ([]diffRow, error) { return rows, nil }, 0)
	application.setSearchQuery("needle")

	if application.refresh(false) {
		t.Fatal("unchanged refresh should not request a redraw")
	}
	if application.searchRegex == nil || len(application.searchMatches) != 1 {
		t.Fatal("automatic refresh should preserve the active search")
	}
}

func TestDrawSearchTextHighlightsOnlyMatches(t *testing.T) {
	screen := newTestScreen(t, 30, 3)
	defer screen.Fini()

	drawSearchText(screen, 0, 0, 30, 0, "before needle after", styleDeletion, regexp.MustCompile("needle"))
	_, beforeStyle, _ := screen.Get(0, 0)
	_, matchStyle, _ := screen.Get(7, 0)
	_, afterStyle, _ := screen.Get(14, 0)
	if beforeStyle != styleDeletion || afterStyle != styleDeletion {
		t.Fatal("expected non-matching text to keep its diff style")
	}
	if matchStyle != styleSearchMatch {
		t.Fatal("expected matching text to use the search highlight")
	}
}

func screenLines(screen tcell.Screen, width, height int) []string {
	lines := make([]string, height)
	for y := range height {
		var line strings.Builder
		for x := range width {
			content, _, _ := screen.Get(x, y)
			if content == "" {
				line.WriteByte(' ')
				continue
			}
			line.WriteString(content)
		}
		lines[y] = line.String()
	}
	return lines
}

func newTestScreen(t *testing.T, width, height int) tcell.Screen {
	t.Helper()
	terminal := vt.NewMockTerm(vt.MockOptSize{X: vt.Col(width), Y: vt.Row(height)})
	screen, err := tcell.NewTerminfoScreenFromTty(
		terminal,
		tcell.OptTerm("ansi"),
		tcell.OptNegotiation(false),
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	return screen
}
