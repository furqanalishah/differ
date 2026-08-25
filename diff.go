package main

import (
	"regexp"
	"strconv"
	"strings"
)

type rowKind uint8

const (
	rowContext rowKind = iota
	rowChange
	rowFile
	rowHunk
	rowMeta
)

type cellKind uint8

const (
	cellEmpty cellKind = iota
	cellContext
	cellDeletion
	cellAddition
)

type diffCell struct {
	line int
	text string
	kind cellKind
}

type diffRow struct {
	kind  rowKind
	left  diffCell
	right diffCell
	text  string
}

type pendingChanges struct {
	deletions []diffCell
	additions []diffCell
}

var hunkPattern = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@(.*)$`)

func parseUnifiedDiff(input string) []diffRow {
	input = strings.TrimSuffix(input, "\n")
	if input == "" {
		return nil
	}

	lines := strings.Split(input, "\n")
	rows := make([]diffRow, 0, len(lines))
	changes := pendingChanges{}
	oldLine := 0
	newLine := 0
	inHunk := false
	currentFileRow := -1

	flushChanges := func() {
		count := max(len(changes.deletions), len(changes.additions))
		for index := range count {
			row := diffRow{kind: rowChange}
			if index < len(changes.deletions) {
				row.left = changes.deletions[index]
			}
			if index < len(changes.additions) {
				row.right = changes.additions[index]
			}
			rows = append(rows, row)
		}
		changes = pendingChanges{}
	}

	for _, line := range lines {
		if strings.HasPrefix(line, "diff --git ") {
			flushChanges()
			inHunk = false
			oldPath, newPath := diffGitPaths(line)
			rows = append(rows, diffRow{
				kind:  rowFile,
				left:  diffCell{text: oldPath},
				right: diffCell{text: newPath},
				text:  strings.TrimPrefix(line, "diff --git "),
			})
			currentFileRow = len(rows) - 1
			continue
		}

		if match := hunkPattern.FindStringSubmatch(line); match != nil {
			flushChanges()
			oldLine, _ = strconv.Atoi(match[1])
			newLine, _ = strconv.Atoi(match[3])
			inHunk = true
			rows = append(rows, diffRow{
				kind:  rowHunk,
				left:  diffCell{text: formatHunkSide("-", match[1], match[2], match[5])},
				right: diffCell{text: formatHunkSide("+", match[3], match[4], match[5])},
				text:  line,
			})
			continue
		}

		if !inHunk {
			if currentFileRow >= 0 && strings.HasPrefix(line, "--- ") {
				rows[currentFileRow].left.text = displayPath(strings.TrimPrefix(line, "--- "))
				continue
			}
			if currentFileRow >= 0 && strings.HasPrefix(line, "+++ ") {
				rows[currentFileRow].right.text = displayPath(strings.TrimPrefix(line, "+++ "))
				continue
			}
			if currentFileRow >= 0 && strings.HasPrefix(line, "rename from ") {
				rows[currentFileRow].left.text = displayPath(strings.TrimPrefix(line, "rename from "))
			}
			if currentFileRow >= 0 && strings.HasPrefix(line, "rename to ") {
				rows[currentFileRow].right.text = displayPath(strings.TrimPrefix(line, "rename to "))
			}
			if shouldShowMetadata(line) {
				rows = append(rows, diffRow{kind: rowMeta, text: line})
			}
			continue
		}

		if line == `\ No newline at end of file` {
			flushChanges()
			rows = append(rows, diffRow{kind: rowMeta, text: line})
			continue
		}

		if line == "" {
			continue
		}

		switch line[0] {
		case ' ':
			flushChanges()
			text := line[1:]
			rows = append(rows, diffRow{
				kind:  rowContext,
				left:  diffCell{line: oldLine, text: text, kind: cellContext},
				right: diffCell{line: newLine, text: text, kind: cellContext},
			})
			oldLine++
			newLine++
		case '-':
			changes.deletions = append(changes.deletions, diffCell{
				line: oldLine,
				text: line[1:],
				kind: cellDeletion,
			})
			oldLine++
		case '+':
			changes.additions = append(changes.additions, diffCell{
				line: newLine,
				text: line[1:],
				kind: cellAddition,
			})
			newLine++
		default:
			flushChanges()
			rows = append(rows, diffRow{kind: rowMeta, text: line})
		}
	}

	flushChanges()
	return rows
}

func formatHunkSide(prefix, start, length, context string) string {
	rangeText := prefix + start
	if length != "" {
		rangeText += "," + length
	}
	return rangeText + context
}

func diffGitPaths(line string) (string, string) {
	raw := strings.TrimPrefix(line, "diff --git ")
	parts := strings.Fields(raw)
	if len(parts) == 2 {
		return displayPath(parts[0]), displayPath(parts[1])
	}
	return raw, raw
}

func displayPath(path string) string {
	path = strings.TrimSpace(path)
	if strings.HasPrefix(path, `"`) {
		if unquoted, err := strconv.Unquote(path); err == nil {
			path = unquoted
		}
	}
	if path == "/dev/null" {
		return "∅"
	}
	path = strings.TrimPrefix(path, "a/")
	path = strings.TrimPrefix(path, "b/")
	return path
}

func shouldShowMetadata(line string) bool {
	return strings.HasPrefix(line, "new file mode ") ||
		strings.HasPrefix(line, "deleted file mode ") ||
		strings.HasPrefix(line, "old mode ") ||
		strings.HasPrefix(line, "new mode ") ||
		strings.HasPrefix(line, "similarity index ") ||
		strings.HasPrefix(line, "dissimilarity index ") ||
		strings.HasPrefix(line, "rename from ") ||
		strings.HasPrefix(line, "rename to ") ||
		strings.HasPrefix(line, "copy from ") ||
		strings.HasPrefix(line, "copy to ") ||
		strings.HasPrefix(line, "Binary files ") ||
		strings.HasPrefix(line, "Submodule ")
}
