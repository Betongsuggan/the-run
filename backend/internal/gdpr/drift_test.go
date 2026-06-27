package gdpr_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestNoGdprDocDrift re-runs cmd/gen-gdpr into a tempdir and verifies the
// output equals the committed docs/gdpr/ropa.md + docs/gdpr/dpia-screening.md.
// A PR that adds/removes/renames a tagged field, edits a purpose description,
// or tweaks a retention constant will fail this test — operator runs
// `just gen-gdpr` to refresh.
//
// The DPIA test copies the existing committed file into the tempdir first so
// the generator's prose-preservation pass has the operator-written content to
// round-trip. If prose markers landed in different positions the test still
// catches drift.
func TestNoGdprDocDrift(t *testing.T) {
	backendDir, err := findBackendDir()
	if err != nil {
		t.Fatalf("locate backend dir: %v", err)
	}
	repoRoot := filepath.Dir(backendDir)
	gdprDir := filepath.Join(repoRoot, "docs", "gdpr")
	committedROPA := filepath.Join(gdprDir, "ropa.md")
	committedDPIA := filepath.Join(gdprDir, "dpia-screening.md")
	committedSubproc := filepath.Join(gdprDir, "subprocessors.md")
	committedRetention := filepath.Join(gdprDir, "retention.md")

	tmpDir := t.TempDir()
	tmpROPA := filepath.Join(tmpDir, "ropa.md")
	tmpDPIA := filepath.Join(tmpDir, "dpia-screening.md")
	tmpSubproc := filepath.Join(tmpDir, "subprocessors.md")
	tmpRetention := filepath.Join(tmpDir, "retention.md")

	// Copy existing DPIA into tmp so prose markers can round-trip.
	if existing, err := os.ReadFile(committedDPIA); err == nil {
		if err := os.WriteFile(tmpDPIA, existing, 0o644); err != nil {
			t.Fatalf("seed tmp dpia: %v", err)
		}
	}

	cmd := exec.Command("go", "run", "./cmd/gen-gdpr",
		"-models", "internal/models",
		"-ropa", tmpROPA,
		"-dpia", tmpDPIA,
		"-subprocessors", tmpSubproc,
		"-retention", tmpRetention,
	)
	cmd.Dir = backendDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("generator failed: %v\noutput:\n%s", err, out)
	}

	checkDriftFile(t, "ropa.md", committedROPA, tmpROPA)
	checkDriftFile(t, "dpia-screening.md", committedDPIA, tmpDPIA)
	checkDriftFile(t, "subprocessors.md", committedSubproc, tmpSubproc)
	checkDriftFile(t, "retention.md", committedRetention, tmpRetention)
}

func checkDriftFile(t *testing.T, name, committedPath, regeneratedPath string) {
	t.Helper()
	want, err := os.ReadFile(committedPath)
	if err != nil {
		t.Fatalf("read committed %s: %v", name, err)
	}
	got, err := os.ReadFile(regeneratedPath)
	if err != nil {
		t.Fatalf("read regenerated %s: %v", name, err)
	}
	if bytes.Equal(want, got) {
		return
	}
	t.Errorf("%s is stale. Run `just gen-gdpr` to refresh.\n\n%s", name, miniDiff(want, got))
}

// findBackendDir walks upward from the test's working directory until it
// finds a `go.mod`. Tests run with the package directory as CWD, so we have
// to climb out of `internal/gdpr/`.
func findBackendDir() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found")
		}
		dir = parent
	}
}

// miniDiff returns a short, line-oriented diff hint. Not as polished as
// `diff -u` but enough to find the first divergence point.
func miniDiff(want, got []byte) string {
	wantLines := strings.Split(string(want), "\n")
	gotLines := strings.Split(string(got), "\n")
	max := len(wantLines)
	if len(gotLines) > max {
		max = len(gotLines)
	}
	var b strings.Builder
	const window = 3
	shown := 0
	for i := 0; i < max && shown < 12; i++ {
		w := lineAt(wantLines, i)
		g := lineAt(gotLines, i)
		if w == g {
			continue
		}
		fmt.Fprintf(&b, "line %d:\n  -committed:  %q\n  +regenerated: %q\n", i+1, w, g)
		_ = window
		shown++
	}
	if shown == 0 {
		fmt.Fprintf(&b, "(no per-line diff — files differ in length: committed=%d bytes, regenerated=%d bytes)\n", len(want), len(got))
	}
	return b.String()
}

func lineAt(lines []string, i int) string {
	if i >= len(lines) {
		return "<EOF>"
	}
	return lines[i]
}
