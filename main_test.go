package main

import (
	"reflect"
	"testing"
	"time"
)

func TestParseCLIArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantGitArgs []string
		wantRefresh time.Duration
		wantLayout  diffLayout
		wantHelp    bool
		wantVersion bool
	}{
		{
			name:        "defaults",
			wantRefresh: time.Second,
		},
		{
			name:        "git arguments pass through",
			args:        []string{"--cached", "HEAD~1", "HEAD"},
			wantGitArgs: []string{"--cached", "HEAD~1", "HEAD"},
			wantRefresh: time.Second,
		},
		{
			name:        "custom refresh",
			args:        []string{"--refresh", "250ms", "--cached"},
			wantGitArgs: []string{"--cached"},
			wantRefresh: 250 * time.Millisecond,
		},
		{
			name:        "inline custom refresh",
			args:        []string{"--refresh=2s"},
			wantRefresh: 2 * time.Second,
		},
		{
			name:        "unified layout",
			args:        []string{"--layout", "unified", "--cached"},
			wantGitArgs: []string{"--cached"},
			wantRefresh: time.Second,
			wantLayout:  layoutUnified,
		},
		{
			name:        "inline split layout",
			args:        []string{"--layout=split"},
			wantRefresh: time.Second,
			wantLayout:  layoutSplit,
		},
		{
			name:        "Git unified option passes through",
			args:        []string{"--unified=3"},
			wantGitArgs: []string{"--unified=3"},
			wantRefresh: time.Second,
		},
		{
			name:        "disabled refresh",
			args:        []string{"--no-refresh"},
			wantRefresh: 0,
		},
		{
			name:        "path separator stops option parsing",
			args:        []string{"--", "--refresh", "file.txt"},
			wantGitArgs: []string{"--", "--refresh", "file.txt"},
			wantRefresh: time.Second,
		},
		{
			name:        "help",
			args:        []string{"--help"},
			wantRefresh: time.Second,
			wantHelp:    true,
		},
		{
			name:        "version",
			args:        []string{"--version"},
			wantRefresh: time.Second,
			wantVersion: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			options, err := parseCLIArgs(test.args)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(options.gitArgs, test.wantGitArgs) {
				t.Fatalf("expected Git args %v, got %v", test.wantGitArgs, options.gitArgs)
			}
			if options.refreshInterval != test.wantRefresh {
				t.Fatalf("expected refresh %s, got %s", test.wantRefresh, options.refreshInterval)
			}
			if options.layout != test.wantLayout {
				t.Fatalf("expected %s layout, got %s", test.wantLayout, options.layout)
			}
			if options.showHelp != test.wantHelp || options.showVersion != test.wantVersion {
				t.Fatalf("unexpected help/version state: %#v", options)
			}
		})
	}
}

func TestParseCLIArgsRejectsInvalidRefresh(t *testing.T) {
	for _, args := range [][]string{{"--refresh"}, {"--refresh", "often"}, {"--refresh=-1s"}} {
		if _, err := parseCLIArgs(args); err == nil {
			t.Fatalf("expected %v to fail", args)
		}
	}
}

func TestParseCLIArgsRejectsInvalidLayout(t *testing.T) {
	for _, args := range [][]string{{"--layout"}, {"--layout", "stacked"}, {"--layout="}} {
		if _, err := parseCLIArgs(args); err == nil {
			t.Fatalf("expected %v to fail", args)
		}
	}
}
