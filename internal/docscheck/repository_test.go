package docscheck

import (
	"io/fs"
	"os"
	"strings"
	"testing"

	"github.com/sofired/tracedoc/internal/testsupport"
)

// TestRepositoryDocumentation is the gate itself: it runs every
// documentation check over this repository's working tree, so `go test
// ./...` — which both workflows already run — fails on a dead link, a
// dead anchor, a prose reference to a path that does not exist, an
// undated changelog section for the released version, or a self-check
// command list that has drifted apart between AGENTS.md and the
// workflows.
func TestRepositoryDocumentation(t *testing.T) {
	if errs := CheckAll(os.DirFS(testsupport.Path(t))); len(errs) > 0 {
		for _, err := range errs {
			t.Errorf("documentation check: %s", err)
		}
	}
}

// TestRepositoryDocumentFiles guards the file set the gate runs over. A
// check that silently stopped collecting documents would pass forever, so
// this asserts the contract documents are in scope and the rendered
// fixtures are not.
func TestRepositoryDocumentFiles(t *testing.T) {
	files, err := DocumentFiles(os.DirFS(testsupport.Path(t)))
	if err != nil {
		t.Fatalf("collect documents: %v", err)
	}
	collected := make(map[string]bool, len(files))
	for _, file := range files {
		collected[file] = true
	}

	for _, want := range []string{
		"AGENTS.md",
		"CHANGELOG.md",
		"README.md",
		"docs/cli.md",
		"docs/config.md",
		"docs/schema.md",
		"docs/schema-requirements.md",
		"docs/schema-threat-model.md",
		"docs/versioning.md",
	} {
		if !collected[want] {
			t.Errorf("document set omits %s", want)
		}
	}
	for _, unwanted := range []string{"testdata/matrix.md", "testdata/threats.md"} {
		if collected[unwanted] {
			t.Errorf("document set includes rendered fixture %s", unwanted)
		}
	}
}

// TestRepositoryCodeSpansStayInsideScanBound guards the claim
// maxCodeSpanScanBytes rests on: that no document here carries a code span
// anywhere near the ceiling, so the bound cannot change how a real document
// is read. If one ever approaches it, this fails while the margin is still
// wide rather than after a span silently stops closing.
func TestRepositoryCodeSpansStayInsideScanBound(t *testing.T) {
	const widest = maxCodeSpanScanBytes / 8

	root := os.DirFS(testsupport.Path(t))
	files, err := DocumentFiles(root)
	if err != nil {
		t.Fatalf("collect documents: %v", err)
	}
	for _, name := range files {
		data, err := fs.ReadFile(root, name)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		for number, line := range strings.Split(string(data), "\n") {
			for index := 0; index < len(line); index++ {
				if line[index] != '`' {
					continue
				}
				if width := codeSpanAt(line, index); width > widest {
					t.Errorf(
						"%s:%d has a %d-byte code span, approaching the %d-byte scan bound",
						name, number+1, width, maxCodeSpanScanBytes,
					)
				}
			}
		}
	}
}
