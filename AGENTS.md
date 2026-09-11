# tracedoc agent instructions

## Scope

These instructions apply to the entire repository.

`tracedoc` is a standalone, dependency-free Go CLI for validating,
rendering, and cross-version-comparing governance documents (requirements
matrices and system threat models). It owns reusable mechanics only;
project policy belongs in consumer configuration files, never in this
codebase.

## Constraints

- Go standard library only. Do not add module dependencies; CI enforces an
  empty dependency graph and the absence of `go.sum`.
- The CLI contract ([docs/cli.md](docs/cli.md)), document schemas
  ([docs/schema.md](docs/schema.md)), and configuration schema
  ([docs/config.md](docs/config.md)) are versioned public contracts. Apply
  the compatibility rules in [docs/versioning.md](docs/versioning.md) before
  changing accepted inputs, exit codes, or rendered output.
- Do not add consumer-specific policy (standard lists, hosts, vocabularies,
  URLs) to Go code or the default templates. New policy needs must become
  new bounded configuration members — never a general-purpose rule
  language. The dividing line is whether a validation rule keys off the
  value: vocabularies that coupling or coverage rules depend on
  (applicability, evidence status, likelihood, severity, priority,
  treatment, decision status) are schema-owned, while vocabularies that
  merely describe a project's workflow (document, control, and evidence
  statuses; evidence levels) are configuration.
- Keep decoding strict: any relaxation of size, depth, duplicate-member,
  unknown-field, or single-value rules is a security-relevant contract
  change.
- Rendering must stay deterministic and injection-resistant; document
  content always flows through the escaping template functions.
- `.github/workflows/` holds the CI and release workflows.
- Documentation is gated, not advisory: `internal/docscheck` fails the
  test suite on a dead internal link or anchor, on a backticked repository
  path named in prose that does not exist, on a changelog section with no
  correctly dated entry for the released version, and on self-check command
  lists that have drifted apart. See [Documentation checks](#documentation-checks).
  Behavioural claims in the contract documents cite the test that pins
  them; see [Claims cite their tests](#claims-cite-their-tests).

## Validation

Run before committing:

```sh
gofmt -l .
go vet ./...
go test -race -count=1 ./...
go run ./cmd/tracedoc validate -config testdata/config.json -doc testdata/matrix.json
go run ./cmd/tracedoc render -config testdata/config.json -doc testdata/matrix.json -output testdata/matrix.md -check
go run ./cmd/tracedoc compare -config testdata/config.json -baseline testdata/matrix.json -candidate testdata/matrix.json
go run ./cmd/tracedoc validate -config testdata/config.json -doc testdata/threats.json -requirements testdata/matrix.json
go run ./cmd/tracedoc render -config testdata/config.json -doc testdata/threats.json -output testdata/threats.md -check
go run ./cmd/tracedoc compare -config testdata/config.json -baseline testdata/threats.json -candidate testdata/threats.json
```

If an intentional rendering change makes a golden check fail, regenerate
the affected `testdata/*.md` with the same render command without `-check`
and commit the result, noting the output change in `CHANGELOG.md`.

The "Self-check fixture documents" step in `.github/workflows/ci.yml` is
the authoritative list of these commands. They are no longer allowed to
diverge: `internal/docscheck` asserts that this block and both workflows
run the same commands in the same order, so a change to one that misses
the others fails the test suite.

### Documentation checks

`internal/docscheck` treats this repository's own Markdown as testable.
The `go test` line above already runs it; to run only these checks, or to
read a failure on its own:

```sh
go test ./internal/docscheck/
```

It fails the build when a relative Markdown link or `#anchor` does not
resolve, when a backticked repository path named in prose does not exist,
when `CHANGELOG.md` has no section dated `YYYY-MM-DD` for the version
`cmd/tracedoc/main.go` reports, or when the self-check command lists
above have drifted apart. In CI these run inside the existing
"Test with race detector" step, as `TestRepositoryDocumentation`.

Fixing a failure means correcting the claim, not the checker: repoint the
link, restore the path, date the changelog section, or bring the command
lists back together. The checker is deliberately conservative about what
it treats as a repository path — a backticked candidate must contain a
slash, carry no glob or shell metacharacters, and begin with a segment
that names a real entry at the repository root — so a false report is
more likely a real mistake than a checker bug. Its known blind spots are
listed in the package comment.

These checks cover claims that are mechanically false, never writing
quality. A claim about *behavior* — that a template anchors every entity,
that a field rejects control characters — is beyond the reach of any
documentation linter and belongs in a test next to the behavior it
describes. The contract documents then cite that test, as the next
section describes.

### Claims cite their tests

Every behavioural claim in a versioned contract document —
`docs/schema.md`, `docs/schema-requirements.md`,
`docs/schema-threat-model.md`, `docs/config.md`, and `docs/cli.md` —
cites the test that pins it, the way `CHANGELOG.md` already cites issue
numbers. A citation is the test's name in backticks and the file that
holds it, written into the paragraph or list item that makes the claim,
so a reader who finds the claim finds the test without leaving the page:

> The rendered companion anchors every declared record in one namespace,
> so a reused identifier would silently collapse two anchors into one.
> [...] `TestEveryDeclaredEntityIsAnchored` in
> `internal/render/threats/threats_test.go` pins the anchoring.

A test *pins* a claim when the change that makes the claim false makes
the test fail. `TestEveryDeclaredEntityIsAnchored` enumerates the
fixture's own collections and asserts an anchor for each identifier, so
a collection added to the schema without an anchor fails the test the
day the fixture declares one. The claim it pins was stated in
`docs/schema-threat-model.md`, echoed in `CHANGELOG.md` and in two
comments in `internal/threats/validate.go`, and used to justify the
document-wide identifier rule — while seven of thirteen collections were
not anchored. Four mutually consistent statements gave the reader no
reason to check, and a link checker would have passed all four; a
reviewer reading the template is what caught it. The citation is what
makes that reading repeatable.

- Cite only a test that fails when the claim stops being true. A test
  that exercises the code path without asserting the claimed property is
  not a citation. When no test pins a claim, leave it uncited and treat
  the gap as the finding: write the test, or open an issue for it.
  Citing the nearest test is worse than citing none, because it converts
  a visible gap into a false assurance.
- A document with no behavioural claims carries no citations. Naming a
  field, listing a vocabulary, or defining a format is a definition, not
  a claim that can drift from the code, and inventing a citation for one
  adds noise without protection.
- A change that renames or removes a cited test updates the citation in
  the same commit. Nothing enforces this: whether a claim is verified is
  not mechanically decidable, and a check that guesses cries wolf and
  gets switched off. The file path in a citation is a backticked
  repository path, so `internal/docscheck` does fail the build when it
  stops existing; the test name it does not read. If the convention
  proves its worth, a check that a backticked `TestXxx` name exists in
  the Go source is the narrow follow-on to consider, and no more than
  that.

### Test conventions

Two conventions the filesystem-error tests in `internal/docscheck`
established. Both apply to every test in this repository, and neither
is enforced by a check.

- **A test `fs.FS` wrapper intercepts only the method the target branch
  calls.** `unreadableRepository` in `internal/docscheck/docscheck_test.go`
  embeds `fstest.MapFS` and overrides `ReadDir` alone, because `fs.WalkDir`
  reads each directory as it descends and `rootEntries` reads the root
  directly, and nothing else on those paths needs to fail. Read the branch
  before writing the wrapper, because the wrong method fails silently: a
  wrapper that overrides `Open` does not intercept `fs.ReadFile`, since
  `MapFS` has its own `ReadFile` method, which promotes onto the wrapper,
  satisfies `fs.ReadFileFS`, and reaches the embedded fixture's unwrapped
  `Open`. The test then passes without the injected failure ever being
  observed. Override the method the production code calls, and say in
  the wrapper's type comment which one and why.
- **A new error-branch test is validated by mutation.** Before committing
  a test for an error path, make the production change it exists to
  catch — swallow the error, return `fs.SkipDir` instead of propagating,
  drop an entry from a skip list — and confirm that exactly that test
  fails and no other. Without this, an error-path test proves only that
  a line executed. Record the mutations tried in the pull request
  description, so the reviewer can see what the test was shown to catch.
