# Changelog

Notable changes to `differ` are recorded here.

## 0.12.0 - 2026-08-25

### Changed

- The focused file now uses a brighter banner background that follows file
  navigation while inactive file banners remain subdued.

## 0.11.0 - 2026-08-25

### Added

- A full-width unified diff layout with old and new line-number columns.
- Runtime split/unified switching with `s`.
- `--layout split|unified` for choosing the initial layout without consuming
  Git's own `--unified` argument.

### Changed

- Single-file focus, search, navigation, viewed state, refresh, sidebar, and
  distraction-free mode now work in either layout.

## 0.10.0 - 2026-08-25

### Added

- Single-file review mode, toggled with Enter.
- Scoped search and hunk navigation while reviewing one file.
- `[` / `]` file switching without leaving single-file mode.
- A persistent `SINGLE FILE` position indicator.

### Changed

- Escape returns to the complete diff from single-file mode before quitting.
- Automatic refresh preserves the focused file or safely returns to all files
  when that file no longer has changes.

## 0.9.3 - 2026-08-25

### Changed

- Every file banner now uses the same clear blue-gray background previously
  reserved for the focused file.

## 0.9.2 - 2026-08-25

### Changed

- File banners now use a slightly lighter blue-gray surface so their
  boundaries are distinct from both code and hunk backgrounds.

## 0.9.1 - 2026-08-25

### Changed

- File boundaries now use an explicit `FILE` label, high-clarity colored Git
  status badges, and a more distinct banner surface.

## 0.9.0 - 2026-08-25

### Changed

- Full-width file banners now combine the path, Git status, change type, and
  review state into one clear boundary between files.
- A persistent pane guide replaces repeated `BEFORE` and `AFTER` file labels.
- Addition, deletion, hunk, and header colors use a calmer low-glare palette.
- The footer now shows only core review actions; `?` retains the complete list.

## 0.8.0 - 2026-08-25

### Added

- Session-scoped viewed and unviewed file markers, toggled with `v`.
- Automatic clearing of a viewed mark when that file's diff changes.
- Viewed progress on sidebar directory rows and the sidebar header.
- A compact shortcut overlay, toggled with `?`.

## 0.7.0 - 2026-08-25

### Added

- A distraction-free diff mode, toggled with `z`.
- Temporary search input visibility while distraction-free mode is active.

## 0.6.1 - 2026-08-25

### Fixed

- `[` / `]` file navigation now follows the sidebar's top-to-bottom order.

## 0.6.0 - 2026-08-25

### Added

- Automatic sidebar expansion for the focused file's directory.
- Open and closed folder icons for expanded and inactive directories.

## 0.5.1 - 2026-08-25

### Changed

- Directory groups now use an open-folder icon.
- Files within directories are visibly indented beneath their group.

## 0.5.0 - 2026-08-25

### Added

- A minimal changed-files sidebar, toggled with `f`.
- Directory grouping, file status markers, and active-file highlighting.

### Changed

- Keyboard guidance now emphasizes Vim-style navigation.

## 0.4.0 - 2026-08-25

### Added

- Inline regular-expression search with `/`.
- Highlighted matches and `n` / `N` navigation between matching rows.
- Search guidance in the terminal footer and command help.

## 0.3.0 - 2026-08-25

### Added

- Automatic diff refresh every second.
- `--refresh <duration>` for a custom refresh interval.
- `--no-refresh` for a fully manual view.
- Clear Differ-specific options and Git argument pass-through in `--help`.

## 0.2.0 - 2026-08-25

### Added

- Explicit before and after pane headings.
- Visible addition and deletion markers.
- Keyboard navigation between files and hunks.
- Direct installation with `go install`.

## 0.1.0 - 2026-08-25

### Added

- Initial side-by-side terminal diff viewer.
