package main

import (
	"testing"
)

func TestParseUnifiedDiffAlignsChangedLines(t *testing.T) {
	diff := `diff --git a/example.txt b/example.txt
index 1234567..7654321 100644
--- a/example.txt
+++ b/example.txt
@@ -2,4 +2,5 @@ heading
 same
-old one
-old two
+new one
+new two
+new three
 tail`

	rows := parseUnifiedDiff(diff)
	if len(rows) != 7 {
		t.Fatalf("expected 7 rows, got %d: %#v", len(rows), rows)
	}

	assertRow(t, rows[2], 2, "same", 2, "same")
	assertRow(t, rows[3], 3, "old one", 3, "new one")
	assertRow(t, rows[4], 4, "old two", 4, "new two")
	assertRow(t, rows[5], 0, "", 5, "new three")
	assertRow(t, rows[6], 5, "tail", 6, "tail")
	if rows[4].right.kind != cellAddition {
		t.Fatalf("expected right side to be an addition, got %v", rows[4].right.kind)
	}
	if rows[0].left.text != "example.txt" || rows[0].right.text != "example.txt" {
		t.Fatalf("expected clean file paths, got %q and %q", rows[0].left.text, rows[0].right.text)
	}
	if rows[1].left.text != "-2,4 heading" || rows[1].right.text != "+2,5 heading" {
		t.Fatalf("expected split hunk labels, got %q and %q", rows[1].left.text, rows[1].right.text)
	}
}

func TestParseUnifiedDiffPadsUnequalChangeBlocks(t *testing.T) {
	diff := `diff --git a/example.txt b/example.txt
@@ -10,2 +10,3 @@
-before
+after one
+after two`

	rows := parseUnifiedDiff(diff)
	if len(rows) != 4 {
		t.Fatalf("expected 4 rows, got %d", len(rows))
	}
	assertRow(t, rows[2], 10, "before", 10, "after one")
	assertRow(t, rows[3], 0, "", 11, "after two")
	if rows[3].left.kind != cellEmpty {
		t.Fatalf("expected empty padding cell, got %v", rows[3].left.kind)
	}
}

func TestParseUnifiedDiffKeepsUsefulFileMetadata(t *testing.T) {
	diff := `diff --git a/old.txt b/new.txt
similarity index 100%
rename from old.txt
rename to new.txt`

	rows := parseUnifiedDiff(diff)
	if len(rows) != 4 {
		t.Fatalf("expected 4 rows, got %d", len(rows))
	}
	if rows[0].kind != rowFile || rows[1].kind != rowMeta {
		t.Fatalf("expected file and metadata rows, got %#v", rows)
	}
	if rows[0].left.text != "old.txt" || rows[0].right.text != "new.txt" {
		t.Fatalf("expected rename paths, got %q and %q", rows[0].left.text, rows[0].right.text)
	}
}

func TestParseUnifiedDiffLabelsNewFileSides(t *testing.T) {
	diff := `diff --git a/new.txt b/new.txt
new file mode 100644
--- /dev/null
+++ b/new.txt
@@ -0,0 +1 @@
+new`

	rows := parseUnifiedDiff(diff)
	if rows[0].left.text != "∅" || rows[0].right.text != "new.txt" {
		t.Fatalf("expected empty before side and named after side, got %q and %q", rows[0].left.text, rows[0].right.text)
	}
}

func assertRow(t *testing.T, row diffRow, oldLine int, oldText string, newLine int, newText string) {
	t.Helper()
	if row.left.line != oldLine || row.left.text != oldText || row.right.line != newLine || row.right.text != newText {
		t.Fatalf(
			"unexpected row: left=(%d, %q), right=(%d, %q)",
			row.left.line,
			row.left.text,
			row.right.line,
			row.right.text,
		)
	}
}
