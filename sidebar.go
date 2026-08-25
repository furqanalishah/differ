package main

import "path"

const (
	minimumSidebarTerminalWidth = 72
	minimumSidebarWidth         = 22
	maximumSidebarWidth         = 34
	openFolderIcon              = "📂"
	closedFolderIcon            = "📁"
)

type changedFile struct {
	row       int
	path      string
	directory string
	name      string
	status    string
}

type sidebarLine struct {
	text      string
	fileIndex int
	directory bool
	expanded  bool
}

func changedFiles(rows []diffRow) []changedFile {
	files := make([]changedFile, 0)
	for rowIndex, row := range rows {
		if row.kind != rowFile {
			continue
		}

		filePath, status := changedFilePathAndStatus(row)
		directory := path.Dir(filePath)
		if directory == "." {
			directory = ""
		}
		files = append(files, changedFile{
			row:       rowIndex,
			path:      filePath,
			directory: directory,
			name:      path.Base(filePath),
			status:    status,
		})
	}
	return files
}

func changedFilePathAndStatus(row diffRow) (string, string) {
	switch {
	case row.left.text == "∅":
		return row.right.text, "A"
	case row.right.text == "∅":
		return row.left.text, "D"
	case row.left.text != row.right.text:
		return row.right.text, "R"
	default:
		return row.right.text, "M"
	}
}

func diffRowsForFile(rows []diffRow, files []changedFile, fileIndex int) []diffRow {
	if fileIndex < 0 || fileIndex >= len(files) {
		return nil
	}
	start := files[fileIndex].row
	end := len(rows)
	if fileIndex+1 < len(files) {
		end = files[fileIndex+1].row
	}
	return rows[start:end]
}

func makeSidebarLines(files []changedFile, currentFile int) []sidebarLine {
	lines := make([]sidebarLine, 0, len(files)*2)
	directories, groups := groupFilesByDirectory(files)
	activeDirectory := ""
	if currentFile >= 0 && currentFile < len(files) {
		activeDirectory = files[currentFile].directory
	}
	for _, directory := range directories {
		expanded := directory != "" && directory == activeDirectory
		if directory != "" {
			lines = append(lines, sidebarLine{text: directory + "/", fileIndex: -1, directory: true, expanded: expanded})
			if !expanded {
				continue
			}
		}
		for _, fileIndex := range groups[directory] {
			lines = append(lines, sidebarLine{text: files[fileIndex].name, fileIndex: fileIndex})
		}
	}
	return lines
}

func groupFilesByDirectory(files []changedFile) ([]string, map[string][]int) {
	directories := make([]string, 0)
	groups := make(map[string][]int)
	for fileIndex, file := range files {
		if _, exists := groups[file.directory]; !exists {
			directories = append(directories, file.directory)
		}
		groups[file.directory] = append(groups[file.directory], fileIndex)
	}
	return directories, groups
}

func sidebarFileOrder(files []changedFile) []int {
	directories, groups := groupFilesByDirectory(files)
	order := make([]int, 0, len(files))
	for _, directory := range directories {
		order = append(order, groups[directory]...)
	}
	return order
}

func filePosition(order []int, fileIndex int) int {
	for position, candidate := range order {
		if candidate == fileIndex {
			return position
		}
	}
	return -1
}

func currentFileIndex(files []changedFile, row int) int {
	if len(files) == 0 {
		return -1
	}
	current := 0
	for index, file := range files {
		if file.row > row {
			break
		}
		current = index
	}
	return current
}

func fileSidebarWidth(terminalWidth int) int {
	if terminalWidth < minimumSidebarTerminalWidth {
		return 0
	}
	return min(max(terminalWidth/4, minimumSidebarWidth), maximumSidebarWidth)
}
