package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gdamore/tcell/v3"
)

const version = "0.12.0"

const defaultRefreshInterval = time.Second

type cliOptions struct {
	gitArgs         []string
	refreshInterval time.Duration
	layout          diffLayout
	showHelp        bool
	showVersion     bool
}

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "differ: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	options, err := parseCLIArgs(args)
	if err != nil {
		return err
	}
	if options.showHelp {
		printHelp()
		return nil
	}
	if options.showVersion {
		fmt.Printf("differ %s\n", version)
		return nil
	}

	workingDirectory, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	load := func() ([]diffRow, error) {
		output, loadErr := loadGitDiff(context.Background(), workingDirectory, options.gitArgs)
		if loadErr != nil {
			return nil, loadErr
		}
		return parseUnifiedDiff(output), nil
	}

	rows, err := load()
	if err != nil {
		return err
	}

	screen, err := tcell.NewScreen()
	if err != nil {
		return fmt.Errorf("create terminal screen: %w", err)
	}
	if err := screen.Init(); err != nil {
		return fmt.Errorf("initialize terminal screen: %w", err)
	}
	defer screen.Fini()

	application := newApp(screen, rows, load, options.refreshInterval)
	application.layout = options.layout
	application.rebuildVisibleRows()
	if err := application.run(); err != nil && !errors.Is(err, errQuit) {
		return err
	}
	return nil
}

func parseCLIArgs(args []string) (cliOptions, error) {
	options := cliOptions{refreshInterval: defaultRefreshInterval, layout: layoutSplit}
	for index := 0; index < len(args); index++ {
		argument := args[index]
		switch {
		case argument == "-h" || argument == "--help":
			options.showHelp = true
		case argument == "-v" || argument == "--version":
			options.showVersion = true
		case argument == "--no-refresh":
			options.refreshInterval = 0
		case argument == "--refresh":
			if index+1 >= len(args) {
				return cliOptions{}, errors.New("--refresh requires a duration, for example 2s")
			}
			index++
			interval, err := parseRefreshInterval(args[index])
			if err != nil {
				return cliOptions{}, err
			}
			options.refreshInterval = interval
		case strings.HasPrefix(argument, "--refresh="):
			interval, err := parseRefreshInterval(strings.TrimPrefix(argument, "--refresh="))
			if err != nil {
				return cliOptions{}, err
			}
			options.refreshInterval = interval
		case argument == "--layout":
			if index+1 >= len(args) {
				return cliOptions{}, errors.New("--layout requires split or unified")
			}
			index++
			layout, err := parseDiffLayout(args[index])
			if err != nil {
				return cliOptions{}, err
			}
			options.layout = layout
		case strings.HasPrefix(argument, "--layout="):
			layout, err := parseDiffLayout(strings.TrimPrefix(argument, "--layout="))
			if err != nil {
				return cliOptions{}, err
			}
			options.layout = layout
		case argument == "--":
			options.gitArgs = append(options.gitArgs, args[index:]...)
			return options, nil
		default:
			options.gitArgs = append(options.gitArgs, argument)
		}
	}
	return options, nil
}

func parseDiffLayout(value string) (diffLayout, error) {
	switch value {
	case "split":
		return layoutSplit, nil
	case "unified":
		return layoutUnified, nil
	default:
		return layoutSplit, fmt.Errorf("invalid layout %q (use split or unified)", value)
	}
}

func parseRefreshInterval(value string) (time.Duration, error) {
	interval, err := time.ParseDuration(value)
	if err != nil || interval < 0 {
		return 0, fmt.Errorf("invalid refresh duration %q", value)
	}
	return interval, nil
}

func printHelp() {
	fmt.Print(`differ - a focused Git diff viewer

Usage:
  differ [options] [git diff arguments]

Options:
  --refresh <duration>  Auto-refresh interval (default: 1s; 0 disables)
  --no-refresh          Disable automatic refresh
  --layout <mode>       Start in split or unified layout (default: split)
  -h, --help            Show this help
  -v, --version         Show the version

Examples:
  differ                       All changes, including untracked files
  differ --cached              Staged changes
  differ HEAD~1 HEAD           Compare revisions
  differ -- path/to/file       Limit the diff to a path
  differ --refresh 2s          Refresh every two seconds
  differ --layout unified      Start with one full-width diff column

Keys:
  Enter                        Toggle the focused file alone / all files
  /                            Search with a regex (an empty search clears it)
  n, N                         Next / previous match, or hunk without a search
  ], [                         Next / previous file
  v                            Toggle the focused file viewed / unviewed
  f                            Toggle the auto-expanding file sidebar
  s                            Toggle split / unified layout
  z                            Toggle distraction-free mode
  ?                            Show the shortcut overlay
  j, k, up, down               Scroll vertically
  h, l, left, right            Pan horizontally
  g, G                         Jump to the first / last row
  r                            Refresh immediately
  Escape                       Return to all files, cancel search, or quit
  q, Ctrl-C                    Quit

Arguments not listed above are passed to git diff.
`)
}
