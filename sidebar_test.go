package main

import (
	"slices"
	"testing"
)

func TestChangedFilesBuildsStatusesAndDirectories(t *testing.T) {
	rows := []diffRow{
		{kind: rowFile, left: diffCell{text: "README.md"}, right: diffCell{text: "README.md"}},
		{kind: rowContext},
		{kind: rowFile, left: diffCell{text: "∅"}, right: diffCell{text: "src/new.go"}},
		{kind: rowFile, left: diffCell{text: "docs/old.md"}, right: diffCell{text: "docs/new.md"}},
		{kind: rowFile, left: diffCell{text: "obsolete.txt"}, right: diffCell{text: "∅"}},
	}

	files := changedFiles(rows)
	if len(files) != 4 {
		t.Fatalf("expected 4 changed files, got %d", len(files))
	}
	wantStatuses := []string{"M", "A", "R", "D"}
	wantPaths := []string{"README.md", "src/new.go", "docs/new.md", "obsolete.txt"}
	for index := range files {
		if files[index].status != wantStatuses[index] || files[index].path != wantPaths[index] {
			t.Fatalf("file %d: got %s %s, want %s %s", index, files[index].status, files[index].path, wantStatuses[index], wantPaths[index])
		}
	}
	if files[1].directory != "src" || files[1].name != "new.go" {
		t.Fatalf("expected a split directory and filename, got %#v", files[1])
	}
}

func TestSidebarLinesGroupFilesByDirectory(t *testing.T) {
	files := []changedFile{
		{path: "README.md", name: "README.md", status: "M"},
		{path: "src/app.go", directory: "src", name: "app.go", status: "M"},
		{path: "LICENSE", name: "LICENSE", status: "M"},
		{path: "src/app_test.go", directory: "src", name: "app_test.go", status: "A"},
	}

	lines := makeSidebarLines(files, 1)
	if len(lines) != 5 {
		t.Fatalf("expected 5 sidebar lines, got %d", len(lines))
	}
	if lines[2].text != "src/" || !lines[2].directory || !lines[2].expanded {
		t.Fatalf("expected one src directory heading, got %#v", lines[2])
	}
	if lines[3].text != "app.go" || lines[3].fileIndex != 1 || lines[4].text != "app_test.go" {
		t.Fatalf("expected both src files below one directory, got %#v", lines[3:])
	}
	if order := sidebarFileOrder(files); !slices.Equal(order, []int{0, 2, 1, 3}) {
		t.Fatalf("expected navigation to follow the grouped sidebar, got %v", order)
	}
}

func TestSidebarLinesOnlyExpandTheFocusedDirectory(t *testing.T) {
	files := []changedFile{
		{path: "src/app.go", directory: "src", name: "app.go", status: "M"},
		{path: "docs/guide.md", directory: "docs", name: "guide.md", status: "M"},
	}

	srcFocused := makeSidebarLines(files, 0)
	if len(srcFocused) != 3 || !srcFocused[0].expanded || srcFocused[2].expanded {
		t.Fatalf("expected only src to be expanded, got %#v", srcFocused)
	}
	if srcFocused[1].text != "app.go" || srcFocused[2].text != "docs/" {
		t.Fatalf("expected the src child followed by closed docs, got %#v", srcFocused)
	}

	docsFocused := makeSidebarLines(files, 1)
	if len(docsFocused) != 3 || docsFocused[0].expanded || !docsFocused[1].expanded {
		t.Fatalf("expected src to close and docs to expand, got %#v", docsFocused)
	}
	if docsFocused[2].text != "guide.md" {
		t.Fatalf("expected the focused docs child to be visible, got %#v", docsFocused)
	}
}

func TestCurrentFileIndexTracksTheVisibleDiff(t *testing.T) {
	files := []changedFile{{row: 0}, {row: 5}, {row: 11}}
	if got := currentFileIndex(files, 8); got != 1 {
		t.Fatalf("expected second file to be active, got %d", got)
	}
	if got := currentFileIndex(nil, 0); got != -1 {
		t.Fatalf("expected no active file, got %d", got)
	}
}

func TestFileNavigationDoesNotSkipVisibleSidebarEntries(t *testing.T) {
	rows := []diffRow{
		{kind: rowFile, left: diffCell{text: "README.md"}, right: diffCell{text: "README.md"}},
		{kind: rowFile, left: diffCell{text: "docs/guide.md"}, right: diffCell{text: "docs/guide.md"}},
		{kind: rowFile, left: diffCell{text: "obsolete.txt"}, right: diffCell{text: "∅"}},
		{kind: rowFile, left: diffCell{text: "notes/team.md"}, right: diffCell{text: "notes/team.md"}},
		{kind: rowFile, left: diffCell{text: "∅"}, right: diffCell{text: "new-feature.md"}},
	}
	application := &app{rows: rows, currentFile: 2}

	application.jumpToFile(true)
	if application.currentFile != 4 {
		t.Fatalf("expected the next visible root file, got file index %d", application.currentFile)
	}
	application.jumpToFile(true)
	if application.currentFile != 1 {
		t.Fatalf("expected navigation to enter the next visible directory, got file index %d", application.currentFile)
	}
	application.jumpToFile(false)
	if application.currentFile != 4 {
		t.Fatalf("expected reverse navigation to return to the prior visible file, got file index %d", application.currentFile)
	}
}

func TestFileSidebarWidthProtectsTheDiffView(t *testing.T) {
	if got := fileSidebarWidth(60); got != 0 {
		t.Fatalf("expected no sidebar in a narrow terminal, got width %d", got)
	}
	if got := fileSidebarWidth(120); got != 30 {
		t.Fatalf("expected proportional sidebar width, got %d", got)
	}
	if got := fileSidebarWidth(200); got != maximumSidebarWidth {
		t.Fatalf("expected capped sidebar width, got %d", got)
	}
}
