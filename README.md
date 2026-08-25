# differ

A focused split and unified Git diff viewer for the terminal.

`differ` opens directly on your diff. There are no staging controls, branch
menus, commit screens, or file browser. The view refreshes automatically every
second.

## Install

Requires Git and Go 1.25 or newer:

```bash
go install github.com/furqanalishah/differ@latest
```

Make sure Go's binary directory is on your `PATH` (usually `~/go/bin`).

## Use

Run it from anywhere inside a Git repository:

```bash
differ
```

By default, it shows all working-tree changes against `HEAD`, including
untracked files. Any other arguments are passed to `git diff`:

```bash
differ --cached
differ HEAD~1 HEAD
differ -- path/to/file
```

Differ-specific options:

| Flag | Description |
| --- | --- |
| `--refresh <duration>` | Change the refresh interval, such as `2s` or `500ms` |
| `--no-refresh` | Disable automatic refresh |
| `--layout <mode>` | Start in `split` or `unified` layout |
| `-h`, `--help` | Show all options and examples |
| `-v`, `--version` | Show the installed version |

Each diff section starts with a full-width `FILE` banner. A colored `[M]`,
`[A]`, `[D]`, or `[R]` badge identifies its Git status beside the path, change
type, and review state. The focused file uses a brighter banner so it remains
clear among nearby changes. Split layout keeps `BEFORE` and `AFTER` visible in
the pane guide.

| Key | Action |
| --- | --- |
| `j` / `k`, up / down | Scroll vertically |
| `h` / `l`, left / right | Pan horizontally |
| `g` / `G` | Jump to the first / last row |
| Enter | Toggle the focused file alone / all files |
| `/` | Search with a regular expression |
| `n` / `N` | Next / previous match, or hunk when no search is active |
| `]` / `[` | Next / previous file |
| `v` | Toggle the focused file viewed / unviewed |
| `f` | Toggle the changed-files sidebar |
| `s` | Toggle split / unified layout |
| `z` | Toggle distraction-free mode |
| `?` | Show the shortcut overlay |
| `r` | Refresh immediately |
| Escape | Return to all files, cancel search, or quit |
| `q`, Ctrl-C | Quit |

Search updates as you type. Press `Enter` to keep it, or submit an empty search
to clear it.

Split layout shows BEFORE and AFTER panes. Unified layout uses one full-width
diff column with old and new line numbers. Press `s` to switch at any time, or
start with `differ --layout unified`.

Single-file mode limits the diff, search, and hunk navigation to the focused
file. Use `[` / `]` to switch files without leaving the mode. The pane guide
shows `SINGLE FILE` and the file's position; press Enter or Escape to return.

The sidebar automatically expands the directory containing the focused file
and closes the directory you leave. `✓` marks a viewed file and `○` marks an
unviewed file. Directory rows show viewed progress, such as `2/3`.

Viewed marks last for the current session. If a viewed file's diff changes,
Differ automatically marks it unviewed again.

Distraction-free mode hides the sidebar and status bar. Starting a search
temporarily shows its input at the bottom.

Sidebar status markers:

| Marker | Meaning |
| --- | --- |
| `M` | Modified file |
| `A` | Added file |
| `D` | Deleted file |
| `R` | Renamed file |

## Changelog

See [CHANGELOG.md](CHANGELOG.md).

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

MIT. See [LICENSE](LICENSE).

## Inspiration

Inspired by [Lazygit](https://github.com/jesseduffield/lazygit) and its
terminal-first approach to Git.
