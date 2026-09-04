package docscheck

import (
	"errors"
	"io/fs"
	"maps"
	"strings"
	"testing"
	"testing/fstest"
)

// The fixture repository is a miniature of this one: the same file
// layout, the same self-check step in both workflows, and a changelog
// whose released section matches the embedded toolVersion. Every drift
// test starts from it and changes exactly one thing, so a failure names
// the drift rather than the fixture.
const (
	fixtureAgents = "# tracedoc agent instructions\n" +
		"\n" +
		"`.github/workflows/` holds the CI and release workflows.\n" +
		"\n" +
		"## Validation\n" +
		"\n" +
		"```sh\n" +
		"gofmt -l .\n" +
		"go run ./cmd/tracedoc validate -config testdata/config.json -doc testdata/matrix.json\n" +
		"go run ./cmd/tracedoc compare -config testdata/config.json -baseline testdata/matrix.json -candidate testdata/matrix.json\n" +
		"```\n"

	fixtureChangelog = "# Changelog\n" +
		"\n" +
		"## Unreleased\n" +
		"\n" +
		"## 0.1.0 - 2026-08-02\n" +
		"\n" +
		"Initial release.\n"

	fixtureMain = "package main\n" +
		"\n" +
		"const toolVersion = \"0.1.0\"\n"

	fixtureCLI = "# CLI contract\n" +
		"\n" +
		"## `tracedoc compare -config <path> -baseline <path> -candidate <path>`\n" +
		"\n" +
		"Compares two documents.\n"

	fixtureSchema = "# Document schemas\n" +
		"\n" +
		"See [compare](cli.md#tracedoc-compare--config-path--baseline-path--candidate-path).\n" +
		"\n" +
		"## `threat_model`\n" +
		"\n" +
		"Configured in [config.md](config.md#threat_model).\n"

	fixtureConfig = "# Consumer configuration\n" +
		"\n" +
		"### `threat_model`\n" +
		"\n" +
		"Threat-model section.\n"

	fixtureWorkflow = "name: CI\n" +
		"jobs:\n" +
		"  verify:\n" +
		"    steps:\n" +
		"      - name: Vet\n" +
		"        run: go vet ./...\n" +
		"      - name: Self-check fixture documents\n" +
		"        run: |\n" +
		"          go run ./cmd/tracedoc validate -config testdata/config.json -doc testdata/matrix.json\n" +
		"          go run ./cmd/tracedoc compare -config testdata/config.json -baseline testdata/matrix.json -candidate testdata/matrix.json\n" +
		"      - name: Done\n" +
		"        run: echo done\n"
)

// removed marks a fixture file that an override deletes rather than
// replaces.
const removed = "\x00removed"

// repository builds the fixture repository with overrides applied. An
// override whose content is removed deletes the file.
func repository(overrides map[string]string) fstest.MapFS {
	files := map[string]string{
		"AGENTS.md":                       fixtureAgents,
		"CHANGELOG.md":                    fixtureChangelog,
		"README.md":                       "# tracedoc\n\nSee [the CLI contract](docs/cli.md).\n",
		"cmd/tracedoc/main.go":            fixtureMain,
		"docs/cli.md":                     fixtureCLI,
		"docs/config.md":                  fixtureConfig,
		"docs/schema.md":                  fixtureSchema,
		".github/workflows/ci.yml":        fixtureWorkflow,
		".github/workflows/release.yml":   fixtureWorkflow,
		"testdata/matrix.md":              "# Fixture\n\nSee [plan](../plan.md#132-isolation).\n",
		"internal/docscheck/docscheck.go": "package docscheck\n",
	}
	maps.Copy(files, overrides)

	fsys := make(fstest.MapFS, len(files))
	for name, data := range files {
		if data == removed {
			continue
		}
		fsys[name] = &fstest.MapFile{Data: []byte(data)}
	}
	return fsys
}

// errUnreadable is the failure unreadableRepository injects. The tests
// assert on it by identity where the error is returned and by its message
// where a check turns it into a finding, so a report that names it proves
// the cause was carried out to the reader rather than replaced with a
// generic one along the way.
var errUnreadable = errors.New("simulated I/O failure")

// unreadableRepository is the fixture repository with one directory that
// cannot be read. fstest.MapFS alone cannot reach the checks' filesystem
// error branches: every path in a MapFS either exists or is cleanly
// absent, and a missing directory is not an error the walk reports. A
// repository whose tree becomes unreadable mid-check is what those
// branches are for, so the tests manufacture one.
//
// Only ReadDir is intercepted, because it is the single call all three
// branches pass through: fs.WalkDir reads each directory as it descends,
// and rootEntries reads the root directly. Everything else — Open, Stat,
// the file contents — is the embedded fixture, unchanged.
type unreadableRepository struct {
	fstest.MapFS

	// dir is the directory whose ReadDir fails, "." for the repository
	// root.
	dir string
}

func (r unreadableRepository) ReadDir(name string) ([]fs.DirEntry, error) {
	if name == r.dir {
		return nil, errUnreadable
	}
	return r.MapFS.ReadDir(name)
}

// unreadable builds the fixture repository with dir unreadable.
func unreadable(dir string) unreadableRepository {
	return unreadableRepository{MapFS: repository(nil), dir: dir}
}

// unstatableRepository is the fixture repository with one path whose Stat
// fails. CheckLinks's fs.Stat branch is what needs this: a path that exists
// in the tree but cannot be examined is not something MapFS can express,
// and intercepting ReadDir would never reach it.
//
// Only Stat is intercepted — the method that branch actually calls. Open
// and ReadFile stay on the embedded fixture, so a later readDocument of
// the same path would still succeed; that separation keeps this fixture
// aimed at the Stat message, not the "cannot be read" one beside it.
type unstatableRepository struct {
	fstest.MapFS

	// name is the path whose Stat fails.
	name string
}

func (r unstatableRepository) Stat(name string) (fs.FileInfo, error) {
	if name == r.name {
		return nil, errUnreadable
	}
	return fs.Stat(r.MapFS, name)
}

// unstatable builds the fixture repository with name un-Stat-able.
func unstatable(name string) unstatableRepository {
	return unstatableRepository{MapFS: repository(nil), name: name}
}

// checkAll runs every check over the fixture repository with overrides
// applied.
func checkAll(t *testing.T, overrides map[string]string) []string {
	t.Helper()
	return CheckAll(repository(overrides))
}

// requireReport fails unless exactly one finding mentions every fragment.
func requireReport(t *testing.T, errs []string, fragments ...string) {
	t.Helper()
	var matched []string
	for _, err := range errs {
		if containsAll(err, fragments) {
			matched = append(matched, err)
		}
	}
	if len(matched) != 1 {
		t.Fatalf("want exactly one finding mentioning %v, got %d in %v", fragments, len(matched), errs)
	}
}

// requireOnlyReport asserts that errs holds exactly one finding and that it
// mentions every fragment. The drift reproductions use it because each
// override introduces exactly one problem: a second finding would mean a
// check had grown a false positive alongside the true one, which
// requireReport alone would not notice.
func requireOnlyReport(t *testing.T, errs []string, fragments ...string) {
	t.Helper()
	if len(errs) != 1 {
		t.Fatalf("want exactly one finding, got %d: %v", len(errs), errs)
	}
	requireReport(t, errs, fragments...)
}

// requireClean asserts that the fixture produced no findings at all. Every
// use marks documentation that is correct and must not be reported.
func requireClean(t *testing.T, errs []string) {
	t.Helper()
	if len(errs) > 0 {
		t.Fatalf("correct documentation was reported: %v", errs)
	}
}

func containsAll(value string, fragments []string) bool {
	for _, fragment := range fragments {
		if !strings.Contains(value, fragment) {
			return false
		}
	}
	return true
}

func TestCleanRepositoryPassesEveryCheck(t *testing.T) {
	requireClean(t, checkAll(t, nil))
}

// TestCorrectDocumentationIsNeverReported collects the shapes that a
// contributor may legitimately write and that an earlier draft of these
// checks reported as broken. A false positive breaks CI on correct
// documentation, which is the failure mode that would get the whole gate
// switched off, so each one is pinned here.
func TestCorrectDocumentationIsNeverReported(t *testing.T) {
	t.Run("a multi-backtick code span quoting link syntax", func(t *testing.T) {
		// The only way to show a code span containing a backtick is to
		// open with two, which is exactly how prose about Markdown quotes
		// Markdown.
		requireClean(t, checkAll(t, map[string]string{
			"README.md": "# tracedoc\n\nWrite `` `[text](docs/nowhere.md)` `` to show link syntax literally.\n",
		}))
	})

	t.Run("a multi-backtick code span quoting a path", func(t *testing.T) {
		requireClean(t, checkAll(t, map[string]string{
			"README.md": "# tracedoc\n\nWrite `` `docs/nowhere.md` `` to show a path literally.\n",
		}))
	})

	t.Run("a Keep a Changelog bracketed release heading", func(t *testing.T) {
		// CHANGELOG.md names Keep a Changelog as its model, and that
		// format brackets the version.
		requireClean(t, checkAll(t, map[string]string{
			"CHANGELOG.md": "# Changelog\n\n## [Unreleased]\n\n## [0.1.0] - 2026-08-02\n\nInitial release.\n",
		}))
	})

	t.Run("a backticked path with a line reference", func(t *testing.T) {
		requireClean(t, checkAll(t, map[string]string{
			"README.md": "# tracedoc\n\nSee `internal/docscheck/docscheck.go#L120` for the fence logic.\n",
		}))
	})

	t.Run("a dead link left inside an HTML comment", func(t *testing.T) {
		requireClean(t, checkAll(t, map[string]string{
			"README.md": "# tracedoc\n\n<!-- TODO: restore [old link](docs/removed.md) later -->\n",
		}))
	})

	t.Run("an HTML comment spanning several lines", func(t *testing.T) {
		requireClean(t, checkAll(t, map[string]string{
			"README.md": "# tracedoc\n\n<!--\n[a](docs/gone.md)\n`docs/gone.md`\n-->\n\nSee [cli](docs/cli.md).\n",
		}))
	})

	t.Run("a comment delimiter quoted in a code span opens no comment", func(t *testing.T) {
		// GitHub renders this as literal code, so the link after it is a
		// live claim and must still be checked.
		errs := checkAll(t, map[string]string{
			"README.md": "# tracedoc\n\nUse `<!--` to hide a note, then see [cli](docs/cli.md).\n",
		})
		requireClean(t, errs)
	})

	t.Run("a fence marker carrying an info string does not close the block", func(t *testing.T) {
		// CommonMark closes a fence only on a marker followed by nothing
		// but spaces or tabs, so the ```sh line is content. Ending the block
		// there would put the example's dead link back into live prose
		// and report the document for a link it only quotes.
		requireClean(t, checkAll(t, map[string]string{
			"README.md": "# tracedoc\n\n```md\n```sh\nSee [gone](docs/gone.md).\n```\n",
		}))
	})

	t.Run("a code span does not carry across a blank line", func(t *testing.T) {
		// The backtick opens nothing, because a span cannot cross the end
		// of a paragraph, so the delimiter below it is a real comment and
		// the link inside it is a note rather than a claim.
		requireClean(t, checkAll(t, map[string]string{
			"README.md": "# tracedoc\n\nWrite `<!--\n\n` to open one, then see [gone](docs/gone.md).\n",
		}))
	})

	t.Run("an unrelated backtick pairing does not un-blank a comment", func(t *testing.T) {
		// The apostrophe backtick and the one opening `git status` close
		// on each other across the line ending, and the comment falls
		// inside the span they form. Keeping that span's contents would
		// hand the commented-out link back to the link check as a claim.
		requireClean(t, checkAll(t, map[string]string{
			"README.md": "# tracedoc\n\nDon`t forget: <!-- see [gone](docs/gone.md) for details --> then run\n`git status` to check.\n",
		}))
	})

	t.Run("a link quoted in a span crossing several lines is not a claim", func(t *testing.T) {
		// The span opens on the first line and closes on the third, so
		// the link on the second is code that GitHub renders literally.
		requireClean(t, checkAll(t, map[string]string{
			"README.md": "# tracedoc\n\nWrite `a\n[gone](docs/gone.md)\nb` to quote a link.\n",
		}))
	})

	t.Run("a code span does not carry across a whitespace-only line", func(t *testing.T) {
		// A line of spaces ends the paragraph exactly as an empty one
		// does, so the backtick opens nothing and the delimiter below it
		// is a real comment.
		requireClean(t, checkAll(t, map[string]string{
			"README.md": "# tracedoc\n\nWrite `<!--\n   \n` to open one, then see [gone](docs/gone.md).\n",
		}))
	})

	t.Run("a run block written with a chomping indicator", func(t *testing.T) {
		requireClean(t, checkAll(t, map[string]string{
			".github/workflows/ci.yml": strings.Replace(fixtureWorkflow, "run: |\n", "run: |-\n", 1),
		}))
	})

	t.Run("a dot-slash relative link", func(t *testing.T) {
		requireClean(t, checkAll(t, map[string]string{
			"README.md": "# tracedoc\n\nSee [cli](./docs/cli.md).\n",
		}))
	})

	t.Run("an image and a titled link that both resolve", func(t *testing.T) {
		requireClean(t, checkAll(t, map[string]string{
			"README.md": "# tracedoc\n\n![schema](docs/schema.md) and [cli](docs/cli.md \"The CLI\").\n",
		}))
	})
}

// The three tests below reproduce the drifts that motivated this check.
// Each one survived multiple merges in this repository and was found by a
// reader rather than by a gate; each one now fails a test.

// TestCatchesTheRenamedCommandAnchorDrift reproduces the dead anchor that
// docs/schema.md carried from the matrix -> tracedoc rename in fd550ce:
// the link kept the old command name after the heading it pointed at was
// renamed.
func TestCatchesTheRenamedCommandAnchorDrift(t *testing.T) {
	errs := checkAll(t, map[string]string{
		"docs/schema.md": strings.Replace(
			fixtureSchema,
			"cli.md#tracedoc-compare--config-path--baseline-path--candidate-path",
			"cli.md#matrix-compare--config-path--baseline-path--candidate-path",
			1,
		),
	})
	requireOnlyReport(t, errs, "docs/schema.md:3", "matrix-compare", "names no heading in docs/cli.md")
}

// TestCatchesTheStagedWorkflowsDrift reproduces AGENTS.md naming
// .github/workflows-staged/, a directory removed in ee1d529. AGENTS.md
// also named a file in that directory as the authoritative source for the
// self-check command list, so following the documented procedure led to a
// path that no longer existed.
func TestCatchesTheStagedWorkflowsDrift(t *testing.T) {
	errs := checkAll(t, map[string]string{
		"AGENTS.md": strings.Replace(
			fixtureAgents,
			"`.github/workflows/` holds",
			"`.github/workflows-staged/` holds",
			1,
		),
	})
	requireOnlyReport(t, errs, "AGENTS.md:3", ".github/workflows-staged/", "does not exist in the repository")
}

// TestCatchesTheUnreleasedChangelogDrift reproduces CHANGELOG.md still
// reading "0.1.0 - Unreleased" after v0.1.0 was tagged, released, and had
// binaries published — a claim that outlived the tag by about three weeks
// and contradicted step 1 of this project's own release process.
func TestCatchesTheUnreleasedChangelogDrift(t *testing.T) {
	errs := checkAll(t, map[string]string{
		"CHANGELOG.md": strings.Replace(fixtureChangelog, "## 0.1.0 - 2026-08-02", "## 0.1.0 - Unreleased", 1),
	})
	requireOnlyReport(t, errs, "CHANGELOG.md:5", "0.1.0", "Unreleased", "YYYY-MM-DD")
}

func TestCheckLinks(t *testing.T) {
	t.Run("missing file is reported", func(t *testing.T) {
		errs := checkAll(t, map[string]string{
			"README.md": "# tracedoc\n\nSee [the guide](docs/guide.md).\n",
		})
		requireOnlyReport(t, errs, "README.md:3", "docs/guide.md", "points at a file that does not exist")
	})

	t.Run("an unreadable link target reports the underlying error", func(t *testing.T) {
		// The clean fixture README links to docs/cli.md, which exists.
		// Stat alone fails; the finding must carry errUnreadable rather
		// than the missing-file message that used to hide every Stat error.
		errs := CheckLinks(unstatable("docs/cli.md"), []string{"README.md"})
		requireOnlyReport(t, errs, "README.md:3", "docs/cli.md", errUnreadable.Error())
		for _, err := range errs {
			if strings.Contains(err, "does not exist") {
				t.Fatalf("unreadable target reported as missing: %v", errs)
			}
		}
	})

	t.Run("same-file anchor is resolved against the document itself", func(t *testing.T) {
		errs := checkAll(t, map[string]string{
			"README.md": "# tracedoc\n\n## Usage\n\nSee [usage](#usage) and [none](#missing).\n",
		})
		requireReport(t, errs, "README.md:5", "#missing", "names no heading in README.md")
	})

	t.Run("a link escaping the repository is reported", func(t *testing.T) {
		errs := checkAll(t, map[string]string{
			"README.md": "# tracedoc\n\nSee [outside](../elsewhere.md).\n",
		})
		requireReport(t, errs, "README.md:3", "escapes the repository")
	})

	t.Run("external and root-absolute targets are left alone", func(t *testing.T) {
		errs := checkAll(t, map[string]string{
			"README.md": "# tracedoc\n\n[a](https://example.org/x#y) [b](mailto:x@example.org) [c](/sofired/tracedoc)\n",
		})
		requireClean(t, errs)
	})

	t.Run("a fragment on a non-Markdown file is left alone", func(t *testing.T) {
		errs := checkAll(t, map[string]string{
			"README.md": "# tracedoc\n\nSee [the constant](cmd/tracedoc/main.go#L3).\n",
		})
		requireClean(t, errs)
	})

	t.Run("anchors resolve into documents outside the checked set", func(t *testing.T) {
		errs := checkAll(t, map[string]string{
			"README.md": "# tracedoc\n\nSee [fixture](testdata/matrix.md#fixture) and [gone](testdata/matrix.md#absent).\n",
		})
		requireReport(t, errs, "README.md:3", "#absent", "names no heading in testdata/matrix.md")
	})

	t.Run("links inside fenced blocks and code spans are examples, not claims", func(t *testing.T) {
		errs := checkAll(t, map[string]string{
			"README.md": "# tracedoc\n\nWrite `[text](docs/nowhere.md)` like this:\n\n" +
				"```md\n[text](docs/also-nowhere.md)\n```\n",
		})
		requireClean(t, errs)
	})
}

func TestCheckNamedPaths(t *testing.T) {
	t.Run("a named file that does not exist is reported", func(t *testing.T) {
		errs := checkAll(t, map[string]string{
			"README.md": "# tracedoc\n\nThe entry point is `cmd/tracedoc/cli.go`.\n",
		})
		requireReport(t, errs, "README.md:3", "cmd/tracedoc/cli.go", "does not exist")
	})

	t.Run("a path quoted after an unclosed run is still found", func(t *testing.T) {
		// The leading run of two never closes, so it is literal text and
		// no span opens inside it. Resuming at its second backtick would
		// pair that one with the opening delimiter of the real span and
		// leave the path it quotes outside any span, unchecked.
		errs := checkAll(t, map[string]string{
			"README.md": "# tracedoc\n\nWrite ``unclosed and `cmd/tracedoc/cli.go` here.\n",
		})
		requireReport(t, errs, "README.md:3", "cmd/tracedoc/cli.go", "does not exist")
	})

	t.Run("a named directory that exists passes", func(t *testing.T) {
		errs := checkAll(t, map[string]string{
			"README.md": "# tracedoc\n\nSources live in `internal/docscheck/` and `cmd/tracedoc/main.go`.\n",
		})
		requireClean(t, errs)
	})
}

// TestDetectionIsNotDefeatedByShape covers drift written in a shape the
// simple cases above do not reach.
func TestDetectionIsNotDefeatedByShape(t *testing.T) {
	t.Run("a broken image link is reported", func(t *testing.T) {
		errs := checkAll(t, map[string]string{
			"README.md": "# tracedoc\n\n![diagram](docs/diagram.png)\n",
		})
		requireOnlyReport(t, errs, "README.md:3", "docs/diagram.png", "does not exist")
	})

	t.Run("only the broken link on a two-link line is reported", func(t *testing.T) {
		errs := checkAll(t, map[string]string{
			"README.md": "# tracedoc\n\nSee [cli](docs/cli.md) and [gone](docs/gone.md).\n",
		})
		requireOnlyReport(t, errs, "README.md:3", "docs/gone.md", "does not exist")
	})

	t.Run("a titled link to a missing file is reported", func(t *testing.T) {
		errs := checkAll(t, map[string]string{
			"README.md": "# tracedoc\n\nSee [gone](docs/gone.md \"Title\").\n",
		})
		requireOnlyReport(t, errs, "README.md:3", "docs/gone.md", "does not exist")
	})

	t.Run("a version that is a substring of another is not confused with it", func(t *testing.T) {
		// A prefix or substring match here would let "## 10.1.0" satisfy
		// the check for released version 0.1.0.
		errs := checkAll(t, map[string]string{
			"CHANGELOG.md": "# Changelog\n\n## Unreleased\n\n## 10.1.0 - 2026-09-02\n\n## 0.1.0 - Unreleased\n",
		})
		requireOnlyReport(t, errs, "CHANGELOG.md:7", "released version 0.1.0", "Unreleased", "YYYY-MM-DD")
	})

	t.Run("a dead link after a quoted comment delimiter is still reported", func(t *testing.T) {
		errs := checkAll(t, map[string]string{
			"README.md": "# tracedoc\n\nUse `<!--` to hide a note, then see [gone](docs/gone.md).\n",
		})
		requireOnlyReport(t, errs, "README.md:3", "docs/gone.md", "does not exist")
	})

	t.Run("a dead link after a comment delimiter quoted across lines is still reported", func(t *testing.T) {
		// A code span closes at the first matching run of backticks,
		// which may sit on a later line. Reading the `<!--` inside one as
		// prose opens a comment that swallows the live link below it, and
		// the gate goes quiet on a document it is meant to report.
		errs := checkAll(t, map[string]string{
			"README.md": "# tracedoc\n\nWrite `<!--\n` to open one, then see [gone](docs/gone.md).\n",
		})
		requireOnlyReport(t, errs, "README.md:4", "docs/gone.md", "does not exist")
	})

	t.Run("a span opened before a comment does not consume its terminator", func(t *testing.T) {
		// A comment's body is raw HTML, so the backtick inside it is text
		// rather than a delimiter. Pairing it with the one below would
		// eat the --> and leave the rest of the document commented out.
		errs := checkAll(t, map[string]string{
			"README.md": "# tracedoc\n\n<!-- a note `here --> and\nmore` prose\n[gone](docs/gone.md)\n",
		})
		requireOnlyReport(t, errs, "README.md:5", "docs/gone.md", "does not exist")
	})

	t.Run("a stray backtick inside a comment does not swallow its end", func(t *testing.T) {
		errs := checkAll(t, map[string]string{
			"README.md": "# tracedoc\n\n<!-- a`b --> see\nc`d and then [gone](docs/gone.md) too.\n",
		})
		requireOnlyReport(t, errs, "README.md:4", "docs/gone.md", "does not exist")
	})

	t.Run("a claim after a fence closed with trailing whitespace is checked", func(t *testing.T) {
		// CommonMark allows spaces and tabs after a closing marker, and an
		// editor stripping neither is ordinary. Requiring an exactly bare
		// line would leave the fence open to the end of the file and blank
		// every claim below it.
		errs := checkAll(t, map[string]string{
			"README.md": "# tracedoc\n\n```md\nexample\n``` \t\nSee [gone](docs/gone.md).\n",
		})
		requireOnlyReport(t, errs, "README.md:6", "docs/gone.md", "does not exist")
	})

	t.Run("an unclosed run does not steal a real span's delimiter", func(t *testing.T) {
		// The leading run of two never closes, so it is literal text.
		// Resuming inside it pairs its second backtick with the one that
		// should have opened the span around `<!-- fake `, leaving that
		// delimiter exposed as a comment opener that swallows the link.
		errs := checkAll(t, map[string]string{
			"README.md": "# tracedoc\n\nWrite ``oops and `<!-- fake ` then see [gone](docs/gone.md).\n",
		})
		requireOnlyReport(t, errs, "README.md:3", "docs/gone.md", "does not exist")
	})

	t.Run("a self-check command split across lines fails loudly", func(t *testing.T) {
		errs := checkAll(t, map[string]string{
			".github/workflows/ci.yml": strings.Replace(fixtureWorkflow,
				"          go run ./cmd/tracedoc validate -config testdata/config.json -doc testdata/matrix.json\n",
				"          go run ./cmd/tracedoc validate \\\n            -config testdata/config.json -doc testdata/matrix.json\n", 1),
		})
		requireReport(t, errs, "continued onto the next line with a backslash")
	})
}

// TestUnreadableAndMalformedInputsAreReported covers the branches reached
// when a file the checks depend on is missing or cannot be parsed. Each
// produces a distinct message rather than a silent pass, so a document
// deleted or renamed by mistake surfaces as drift like any other.
func TestUnreadableAndMalformedInputsAreReported(t *testing.T) {
	t.Run("a deleted AGENTS.md", func(t *testing.T) {
		errs := checkAll(t, map[string]string{"AGENTS.md": removed})
		requireReport(t, errs, "AGENTS.md", "read:")
	})

	t.Run("a deleted CHANGELOG.md", func(t *testing.T) {
		errs := checkAll(t, map[string]string{"CHANGELOG.md": removed})
		requireReport(t, errs, "CHANGELOG.md", "read:")
	})

	t.Run("a deleted main.go", func(t *testing.T) {
		errs := checkAll(t, map[string]string{"cmd/tracedoc/main.go": removed})
		requireReport(t, errs, "read cmd/tracedoc/main.go")
	})

	t.Run("a deleted CI workflow", func(t *testing.T) {
		errs := checkAll(t, map[string]string{".github/workflows/ci.yml": removed})
		requireReport(t, errs, "read .github/workflows/ci.yml")
	})

	t.Run("a deleted release workflow", func(t *testing.T) {
		errs := checkAll(t, map[string]string{".github/workflows/release.yml": removed})
		requireReport(t, errs, "read .github/workflows/release.yml")
	})

	t.Run("a main.go that does not parse", func(t *testing.T) {
		errs := checkAll(t, map[string]string{"cmd/tracedoc/main.go": "package main\n\nfunc main( {\n"})
		requireReport(t, errs, "parse cmd/tracedoc/main.go")
	})

	t.Run("a toolVersion that is not a string literal", func(t *testing.T) {
		errs := checkAll(t, map[string]string{"cmd/tracedoc/main.go": "package main\n\nconst toolVersion = 42\n"})
		requireReport(t, errs, "toolVersion constant", "is not a string literal")
	})
}

// TestAnUnreadableTreeIsReportedRatherThanIgnored covers what the checks
// do when the repository itself cannot be read, which no missing or
// malformed file produces: a directory that fails on ReadDir. The
// distinction matters because an unreadable tree makes a check's silence
// meaningless — no documents collected reads exactly like no drift found
// — so each of these branches has to fail loudly instead.
func TestAnUnreadableTreeIsReportedRatherThanIgnored(t *testing.T) {
	t.Run("a directory that fails part way through the walk", func(t *testing.T) {
		files, err := DocumentFiles(unreadable("docs"))
		if !errors.Is(err, errUnreadable) {
			t.Fatalf("collect documents = %v, want %v", err, errUnreadable)
		}
		if files != nil {
			t.Errorf("document set = %v, want none alongside the error", files)
		}
	})

	t.Run("a repository root that cannot be listed", func(t *testing.T) {
		files, err := DocumentFiles(unreadable("."))
		if !errors.Is(err, errUnreadable) {
			t.Fatalf("collect documents = %v, want %v", err, errUnreadable)
		}
		if files != nil {
			t.Errorf("document set = %v, want none alongside the error", files)
		}
	})

	t.Run("a root that stops CheckAll at document collection", func(t *testing.T) {
		errs := CheckAll(unreadable("."))
		requireOnlyReport(t, errs, "collect Markdown documents", errUnreadable.Error())
	})

	t.Run("a root that CheckNamedPaths cannot list", func(t *testing.T) {
		// Called directly rather than through CheckAll, which cannot
		// reach this branch: the same unreadable root stops it at
		// document collection first.
		errs := CheckNamedPaths(unreadable("."), []string{"README.md"})
		requireOnlyReport(t, errs, "read repository root", errUnreadable.Error())
	})

	t.Run("a skipped directory, which is never read at all", func(t *testing.T) {
		// The boundary of the three cases above. fs.WalkDir offers each
		// directory to the callback before reading it, so a directory
		// DocumentFiles skips is never a directory it can fail on --
		// which is why an unreadable testdata or dist cannot break the
		// walk. Reversing that order in DocumentFiles would make the
		// skip list depend on directories it deliberately does not read,
		// and nothing else here would notice.
		if _, err := DocumentFiles(unreadable("testdata")); err != nil {
			t.Fatalf("collect documents = %v, want no error for a skipped directory", err)
		}
	})
}

func TestIsRepositoryPath(t *testing.T) {
	roots := map[string]struct{}{
		".github": {}, "cmd": {}, "docs": {}, "internal": {}, "testdata": {},
	}
	for _, testCase := range []struct {
		candidate string
		want      bool
		reason    string
	}{
		{"docs/cli.md", true, "a plain repository path"},
		{".github/workflows/", true, "a directory with a trailing slash"},
		{".github/workflows-staged/ci.yml", true, "the drift this check exists for"},
		{"go.mod", false, "no slash, so not path-shaped enough to judge"},
		{"github.com/sofired/tracedoc", false, "a module path, not a directory here"},
		{"actions/setup-go", false, "a marketplace action reference"},
		{"linux/amd64", false, "a build platform"},
		{"sum.golang.org/lookup", false, "a host name"},
		{"testdata/*.md", false, "a glob"},
		{"https://", false, "a scheme"},
		{"../plan.md", false, "relative to a consumer document, not to this root"},
		{"./docs/cli.md", false, "not written as a repository-root path"},
		{"/etc/passwd", false, "absolute"},
		{"-config testdata/config.json", false, "a flag and its argument"},
		{"tracedoc_<version>_<os>_<arch>/x", false, "a placeholder"},
		{"/", false, "no segments at all"},
	} {
		t.Run(testCase.candidate, func(t *testing.T) {
			if got := isRepositoryPath(testCase.candidate, roots); got != testCase.want {
				t.Errorf("isRepositoryPath(%q) = %v, want %v (%s)", testCase.candidate, got, testCase.want, testCase.reason)
			}
		})
	}
}

func TestCheckChangelog(t *testing.T) {
	t.Run("no section for the released version is reported", func(t *testing.T) {
		errs := checkAll(t, map[string]string{
			"CHANGELOG.md": "# Changelog\n\n## Unreleased\n",
		})
		requireReport(t, errs, "CHANGELOG.md", "no section for released version 0.1.0")
	})

	t.Run("an unparseable date is reported", func(t *testing.T) {
		errs := checkAll(t, map[string]string{
			"CHANGELOG.md": strings.Replace(fixtureChangelog, "2026-08-02", "2026-13-02", 1),
		})
		requireReport(t, errs, "CHANGELOG.md:5", "2026-13-02", "YYYY-MM-DD")
	})

	t.Run("a duplicate section for the released version is reported", func(t *testing.T) {
		errs := checkAll(t, map[string]string{
			"CHANGELOG.md": fixtureChangelog + "\n## 0.1.0 - 2026-08-03\n",
		})
		requireReport(t, errs, "CHANGELOG.md:9", "duplicate section")
	})

	t.Run("the version is read from the toolVersion constant", func(t *testing.T) {
		errs := checkAll(t, map[string]string{
			"cmd/tracedoc/main.go": "package main\n\nconst toolVersion = \"0.2.0\"\n",
		})
		requireReport(t, errs, "CHANGELOG.md", "no section for released version 0.2.0")
	})

	t.Run("a missing toolVersion constant is reported", func(t *testing.T) {
		errs := checkAll(t, map[string]string{
			"cmd/tracedoc/main.go": "package main\n\nfunc main() {}\n",
		})
		requireReport(t, errs, "declares no toolVersion constant")
	})
}

func TestCheckSelfCheckCommands(t *testing.T) {
	t.Run("a command AGENTS.md omits is reported", func(t *testing.T) {
		errs := checkAll(t, map[string]string{
			"AGENTS.md": strings.Replace(fixtureAgents,
				"go run ./cmd/tracedoc compare -config testdata/config.json -baseline testdata/matrix.json -candidate testdata/matrix.json\n", "", 1),
		})
		requireReport(t, errs, "AGENTS.md", "missing self-check command 2 of 2")
	})

	t.Run("a command AGENTS.md words differently is reported", func(t *testing.T) {
		errs := checkAll(t, map[string]string{
			"AGENTS.md": strings.Replace(fixtureAgents, "-doc testdata/matrix.json", "-doc testdata/threats.json", 1),
		})
		requireReport(t, errs, "AGENTS.md", "self-check command 1 is", "but .github/workflows/ci.yml runs")
	})

	t.Run("a release workflow that has drifted from CI is reported", func(t *testing.T) {
		errs := checkAll(t, map[string]string{
			".github/workflows/release.yml": strings.Replace(fixtureWorkflow,
				"-baseline testdata/matrix.json -candidate testdata/matrix.json\n",
				"-baseline testdata/matrix.json -candidate testdata/matrix.json\n"+
					"          go run ./cmd/tracedoc validate -config testdata/config.json -doc testdata/threats.json\n", 1),
		})
		requireReport(t, errs, ".github/workflows/release.yml", "runs a self-check command .github/workflows/ci.yml does not")
	})

	t.Run("only the named step's run block is read", func(t *testing.T) {
		errs := checkAll(t, map[string]string{
			".github/workflows/ci.yml": strings.Replace(fixtureWorkflow,
				"        run: echo done\n",
				"        run: go run ./cmd/tracedoc version\n", 1),
		})
		requireClean(t, errs)
	})

	t.Run("a workflow without the step is reported", func(t *testing.T) {
		errs := checkAll(t, map[string]string{
			".github/workflows/ci.yml": "name: CI\njobs:\n  verify:\n    steps:\n      - name: Vet\n        run: go vet ./...\n",
		})
		requireReport(t, errs, "has no \"Self-check fixture documents\" step")
	})

	t.Run("a step that runs an action instead of commands is reported", func(t *testing.T) {
		errs := checkAll(t, map[string]string{
			".github/workflows/ci.yml": "name: CI\njobs:\n  verify:\n    steps:\n" +
				"      - name: Self-check fixture documents\n        uses: ./.github/actions/self-check\n" +
				"      - name: Done\n        run: |\n          echo done\n",
		})
		requireReport(t, errs, "has no \"run: |\" block")
	})

	t.Run("an unnamed step does not leak commands into the block", func(t *testing.T) {
		// The scan for the run block must stop at the next sequence item
		// even when that item has no name.
		errs := checkAll(t, map[string]string{
			".github/workflows/ci.yml": "name: CI\njobs:\n  verify:\n    steps:\n" +
				"      - name: Self-check fixture documents\n        uses: ./.github/actions/self-check\n" +
				"      - uses: actions/upload-artifact@v4\n        run: |\n          go run ./cmd/tracedoc version\n",
		})
		requireReport(t, errs, "has no \"run: |\" block")
	})

	t.Run("a CI step that runs no self-check command is reported", func(t *testing.T) {
		errs := checkAll(t, map[string]string{
			".github/workflows/ci.yml": strings.Replace(fixtureWorkflow,
				"          go run ./cmd/tracedoc validate -config testdata/config.json -doc testdata/matrix.json\n"+
					"          go run ./cmd/tracedoc compare -config testdata/config.json -baseline testdata/matrix.json -candidate testdata/matrix.json\n",
				"          echo nothing\n", 1),
		})
		requireReport(t, errs, "runs no go run ./cmd/tracedoc command")
	})

	t.Run("an AGENTS.md documenting no self-check command is reported", func(t *testing.T) {
		errs := checkAll(t, map[string]string{
			"AGENTS.md": "# tracedoc agent instructions\n\n## Validation\n\n```sh\ngofmt -l .\n```\n",
		})
		requireReport(t, errs, "AGENTS.md", "documents no go run ./cmd/tracedoc command")
	})

	t.Run("only the Validation block is read", func(t *testing.T) {
		// A command shown elsewhere in AGENTS.md — here a render example
		// in prose — is not part of the documented list, and reading it
		// as one would report drift the contributor did not introduce.
		errs := checkAll(t, map[string]string{
			"AGENTS.md": fixtureAgents +
				"\n## Rendering\n\n```sh\ngo run ./cmd/tracedoc render -config testdata/config.json" +
				" -doc testdata/matrix.json -output testdata/matrix.md\n```\n",
		})
		requireClean(t, errs)
	})

	t.Run("an AGENTS.md without a Validation block is reported", func(t *testing.T) {
		errs := checkAll(t, map[string]string{
			"AGENTS.md": "# tracedoc agent instructions\n\n## Scope\n\n```sh\n" +
				"go run ./cmd/tracedoc validate -config testdata/config.json -doc testdata/matrix.json\n```\n",
		})
		requireReport(t, errs, "AGENTS.md", "has no \"## Validation\" section")
	})

	t.Run("a shell comment in the block does not end it", func(t *testing.T) {
		// The commands sit inside a fence, where a "# " line is a shell
		// comment rather than a heading. Ending the block on one drops
		// every command below it and reports them as missing.
		errs := checkAll(t, map[string]string{
			"AGENTS.md": strings.Replace(fixtureAgents,
				"go run ./cmd/tracedoc compare",
				"# compare is the run that catches drift\ngo run ./cmd/tracedoc compare", 1),
		})
		requireClean(t, errs)
	})

	t.Run("a heading quoted in a fence does not start the block", func(t *testing.T) {
		errs := checkAll(t, map[string]string{
			"AGENTS.md": strings.Replace(fixtureAgents, "## Validation\n",
				"## Conventions\n\nA section heading is written like this:\n\n"+
					"```md\n## Validation\n```\n\n## Validation\n", 1),
		})
		requireClean(t, errs)
	})

	t.Run("a heading that merely starts with Validation is not the block", func(t *testing.T) {
		errs := checkAll(t, map[string]string{
			"AGENTS.md": strings.Replace(fixtureAgents, "## Validation\n", "## Validation Overview\n", 1),
		})
		requireReport(t, errs, "AGENTS.md", "has no \"## Validation\" section")
	})

	t.Run("a top-level heading ends the block", func(t *testing.T) {
		errs := checkAll(t, map[string]string{
			"AGENTS.md": fixtureAgents + "\n# Appendix\n\n```sh\ngo run ./cmd/tracedoc version\n```\n",
		})
		requireClean(t, errs)
	})

	t.Run("a subheading inside the block does not end it", func(t *testing.T) {
		// AGENTS.md groups the commands under subheadings. Ending the
		// block on one would report every command below the first group
		// as missing.
		errs := checkAll(t, map[string]string{
			"AGENTS.md": strings.Replace(fixtureAgents,
				"go run ./cmd/tracedoc compare",
				"```\n\n### Fixture documents\n\n```sh\ngo run ./cmd/tracedoc compare", 1),
		})
		requireClean(t, errs)
	})

	t.Run("a heading struck out by an HTML comment does not start the block", func(t *testing.T) {
		errs := checkAll(t, map[string]string{
			"AGENTS.md": strings.Replace(fixtureAgents, "## Validation\n",
				"<!--\n## Validation\n\nThe old block, kept until the commands settle.\n-->\n\n## Validation\n", 1),
		})
		requireClean(t, errs)
	})
}

func TestHeadingSlug(t *testing.T) {
	for _, testCase := range []struct {
		heading string
		want    string
	}{
		{"Shared top-level members", "shared-top-level-members"},
		{"Consumer configuration, version 1", "consumer-configuration-version-1"},
		// An underscore inside a code span is part of the name, not an
		// emphasis marker. Dropping it yields "threatmodel" and reports a
		// live anchor as dead.
		{"`threat_model`", "threat_model"},
		{"`attacker_model`", "attacker_model"},
		{"`actors[]`", "actors"},
		{"`render` (per section)", "render-per-section"},
		{
			"`tracedoc compare -config <path> -baseline <path> -candidate <path>`",
			"tracedoc-compare--config-path--baseline-path--candidate-path",
		},
		{"**Bold** and *italic* and ~~struck~~", "bold-and-italic-and-struck"},
		{"A [linked](https://example.org) word", "a-linked-word"},
	} {
		t.Run(testCase.heading, func(t *testing.T) {
			if got := headingSlug(testCase.heading); got != testCase.want {
				t.Errorf("headingSlug(%q) = %q, want %q", testCase.heading, got, testCase.want)
			}
		})
	}
}

func TestHeadingAnchorsNumbersRepeatedSlugs(t *testing.T) {
	anchors := headingAnchors(blankFencedCode(
		"# References\n\n## References\n\n## References\n\n```md\n## References\n```\n",
	))
	for _, want := range []string{"references", "references-1", "references-2"} {
		if _, ok := anchors[want]; !ok {
			t.Errorf("anchor %q missing from %v", want, anchors)
		}
	}
	if _, ok := anchors["references-3"]; ok {
		t.Errorf("a heading inside a fenced block was counted: %v", anchors)
	}
}

func TestBlankFencedCode(t *testing.T) {
	lines := blankFencedCode("a\n```go\nb\n```\nc\n~~~\nd\n~~~\ne\n")
	want := []string{"a", "", "", "", "c", "", "", "", "e", ""}
	if len(lines) != len(want) {
		t.Fatalf("got %d lines, want %d: %q", len(lines), len(want), lines)
	}
	for index := range want {
		if lines[index] != want[index] {
			t.Errorf("line %d = %q, want %q", index+1, lines[index], want[index])
		}
	}
}

func TestDocumentFilesSkipsFixturesAndToolDirectories(t *testing.T) {
	fsys := repository(map[string]string{
		".claude/notes.md": "# Notes\n\n[gone](nowhere.md)\n",
		"dist/README.md":   "# Built\n\n[gone](nowhere.md)\n",
	})
	files, err := DocumentFiles(fsys)
	if err != nil {
		t.Fatalf("collect documents: %v", err)
	}
	for _, file := range files {
		switch file {
		case "testdata/matrix.md", ".claude/notes.md", "dist/README.md":
			t.Errorf("document set includes %s", file)
		}
	}
	if len(files) == 0 {
		t.Fatal("document set is empty")
	}
}
