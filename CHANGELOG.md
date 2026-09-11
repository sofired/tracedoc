# Changelog

All notable changes to this project are documented here. The format is
based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this
project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## Unreleased

### Changed

- The contract documents cite the tests that pin their behavioural
  claims: the anchoring and identifier-uniqueness rules in
  `docs/schema-threat-model.md`, the invisible-code-point guarantees in
  `docs/schema.md`, and the escaping rules in `docs/config.md` each name
  the test that fails when the claim stops being true. `AGENTS.md` states
  the convention, with the anchoring claim as its worked example, and
  records the two test conventions the filesystem-error tests in
  `internal/docscheck` established
  ([#16](https://github.com/sofired/tracedoc/issues/16)).
- Requirements-matrix documentation gains `Traceability`, `Ownership`, and
  `Evidence` sections, so the two schema documents read as siblings rather
  than one carrying its rules inline in table cells. States plainly that
  `traceability.adrs` and `traceability.threats` have no format rule and
  are not resolved — a requirements matrix is validated on its own, and
  `validate -requirements` runs the other way — and that the two document
  types' "evidence" and "owner" are unrelated despite the shared words
  ([#10](https://github.com/sofired/tracedoc/issues/10)).


- **Threat-model schema (breaking, pre-adoption).** The threat model
  expands from a list of threats into a reviewable architecture-level
  model, in place, keeping `document_type`, `schema_version` 1,
  `config_version` 1, and CLI contract 1
  ([#7](https://github.com/sofired/tracedoc/issues/7)). No released
  consumer had adopted the previous shape.

  Added to the document: `status`, accountable `owner`, `summary`, `scope`,
  `assumptions`, `open_questions`, `diagrams`, `components`, `actors`,
  `attacker_model`, `data_flows`, `entry_points`, `decisions`, `risks`,
  `controls`, `planned_evidence`, and `observability`. Assets gain
  `objective`; trust boundaries gain endpoints, data, channels, planned
  guarantees, validation, implementation state, and evidence.

  Threats replace `description` with `source`, `prerequisites`, `action`,
  `impact`, and an ordered `abuse_path`, and add `likelihood`, `severity`,
  and `priority` with rationales, `residual_risk`, `existing_controls`,
  `gaps`, `recommended_mitigations`, `detection_ideas`, and typed link
  lists for actors, assets, boundaries, flows, controls, risks, and planned
  evidence.

  - `affected_assets` becomes `asset_links`, and the threat-level
    `trust_boundaries` list becomes `boundary_links` — the rename that
    frees `trust_boundaries` to name the new top-level collection.
  - `disposition` becomes `treatment`, and its vocabulary changes from the
    outcome states `open`/`mitigated`/`accepted`/`transferred`/`avoided`
    to the decisions `mitigate`/`accept`/`avoid`/`transfer`. A threat now
    records what the project decided, not what has already happened.
  - `severity` narrows to `low`/`medium`/`high`; the former `critical`
    level moves to the new `priority` vocabulary. Authors set `priority`
    informed by likelihood and severity together; the tool does not derive
    it or check it against them.
  - `owner` gains a required `principal`, separating the accountable
    person from the milestone, issue, and workstream that route the work.
  - The `mitigations` object is removed. Its members move rather than
    disappear: `requirements` becomes `controls[].requirement_links` and
    `adrs` becomes `controls[].decision_links`, so an obligation attaches
    to the control that handles a threat rather than to the threat itself;
    `tests` becomes the declared `planned_evidence[]` collection, which
    names the threats it covers; and `risks` becomes `risk_links` on both
    threats and controls, resolved against declared `risks[]` records. A
    threat no longer links to a decision directly — it reaches one through
    a control.
  - Every array member must be present, even when empty, because `compare`
    distinguishes an omitted array from an empty one.

- **Coverage is credited only from threats.** Declared assets, boundaries,
  flows, controls, and risks must be linked by a threat, and threats by
  planned evidence. References from the architecture graph to its own
  members no longer count, which previously would have made the rules
  vacuously true.

- **Configuration.** The `threat_model` section gains `owner_pattern`,
  `document_statuses`, `control_statuses`, `evidence_levels`,
  `evidence_statuses`, `reference_hosts`, and a `coverage` object of named
  switches. `owner_pattern` is scoped to this section rather than shared at
  the top level because only the threat model has an accountable
  principal: a project keeping only a requirements matrix should not have
  to supply a pattern for a concept its documents do not contain.

### Added

- Documentation checks over this repository's own Markdown, in
  `internal/docscheck`, run by the existing test suite in both workflows
  ([#11](https://github.com/sofired/tracedoc/issues/11)). The build now
  fails on a relative link or `#anchor` that does not resolve, a
  backticked repository path named in prose that does not exist, a
  `CHANGELOG.md` with no correctly dated section for the version
  `cmd/tracedoc/main.go` reports, and self-check command lists that have
  drifted apart between `AGENTS.md`, `.github/workflows/ci.yml`, and
  `.github/workflows/release.yml`. Three false claims had previously
  survived multiple merges — a dead anchor left by the `matrix` →
  `tracedoc` rename, a reference to a deleted `workflows-staged`
  directory, and a changelog section still marked unreleased three weeks
  after the tag was published — and each is now reproduced as a failing
  test. The checks are limited to claims that are mechanically false and
  add no module dependency; the package comment records what they
  deliberately do not cover.
- Entry-point coverage: an entry point counts as analysed only when one
  threat both crosses its trust boundary and travels one of its data flows.
  Matching either half alone is rejected as a false positive that would
  certify an unreviewed surface.
- Document-wide identifier uniqueness across every declared collection, so
  a reused ID cannot collapse two anchors in the rendered companion.
  Uniqueness is case-folded, because anchors are: `R1` and `r1` are two
  identifiers addressing one anchor. Only a consumer `risk_pattern` can
  express this; every other identifier format is schema-owned and
  uppercase.
- `anchor` and `anchorHref` template functions, and an anchor on every
  declared record in the rendered companion. Assumptions, components,
  actors, decisions, risks, planned evidence, and observability records are
  now anchored alongside the collections that already were, so each is
  addressable by its identifier. `anchor` escapes an identifier for an
  `id` attribute and `anchorHref` percent-encodes it for a same-document
  destination; both case-fold, so the pair resolves. Only `risks[].id`
  needs either, being the one consumer-patterned identifier.
- Observability records now render their `OBS-` identifier, which the
  companion previously validated but never showed.
- Provenance-checked references for diagrams, decisions, and risks: exactly
  one of a repository-relative `path` or an HTTPS `url` on a host declared
  in the new `reference_hosts` allowlist. Diagram generation and
  diagram-source parsing remain out of scope.
- `check.LexicalURI`, now shared by both document types so they cannot
  drift apart on what a safe URI looks like, and `check.RepoRelativePath`
  for the threat model's repository-relative references.

- **Threat-model review guidance.** Three collections that help a reader
  navigate a model rather than describe the system
  ([#9](https://github.com/sofired/tracedoc/issues/9)):

  - `criticality[]` — what each `priority` level means for this project,
    with worked examples. `priority` is schema-owned, but where its
    boundaries fall is a project's judgement; recording it is what makes a
    ranking reviewable rather than merely asserted.
  - `top_abuse_path_links` — the abuse paths to read first, as declared
    threat IDs. Deliberately not derived from `priority`: editorial order
    is a judgement, and repeating every `critical` threat makes none.
  - `focus_paths[]` — where in the repository a threat actually lives,
    with the threats that make each location worth reading. No other
    collection carries the link from a threat to an artifact.

- Configuration: `threat_model.limits` (`min_criticality_examples`,
  `min_top_abuse_paths`, `max_top_abuse_paths`), each defaulting to `0`
  meaning unbounded, and an eighth coverage switch,
  `require_criticality_for_every_priority`. Quantitative policy is
  configuration rather than schema rule, because the right numbers differ
  between a small service and a platform. A negative bound, or a maximum
  below its own minimum, is rejected when the configuration loads.

- **No `quality_checks` collection**, deliberately. A model's own
  completeness self-assessment duplicates what this tool already proves —
  entry-point, boundary, asset, flow, control, risk and evidence coverage,
  requirement resolution, and required assumptions and open questions are
  each enforced — and a hand-maintained claim can go stale and contradict
  the validator beside it. `docs/schema-threat-model.md` carries the
  mapping. The claims that remain are process notes about how a review was
  run, which belong in the pull request that changed the model.

  This is schema-or-nothing rather than schema-versus-template: unknown
  members are rejected, so a collection absent from the schema cannot be in
  a document for a consumer template to render.

### Fixed

- **Configuration paths now observe the same lexical rule as document
  paths.** `standard_sources[].path` had its own copy of the
  repository-relative rules, and the copies had diverged: the
  configuration's copy swept only ASCII space and tab, so a path carrying a
  non-breaking space or a Unicode line separator passed configuration
  validation while the identical text in a document was rejected. Both now
  run `check.RepoRelativePath`; configuration keeps only its tighter
  256-byte bound, which is the limit worth reporting to someone editing a
  config file. A configuration path carrying non-ASCII whitespace is now
  rejected ([#10](https://github.com/sofired/tracedoc/issues/10)).


- **Control characters are now rejected in every validated string, as
  documented.** `docs/schema.md` and the 0.1.0 changelog both stated that
  validation rejects control characters and the Unicode line and paragraph
  separators in every validated string. List items were checked; the
  roughly sixty scalar prose fields were not, so a title, rationale, or
  summary could carry a vertical tab or an ESC byte and still validate.
  This affects **both** document types, including the requirements matrix
  as released in 0.1.0. Rendered output was never exposed — every escaping
  function neutralizes these code points — but a document that previously
  smuggled a control character through a prose field is now rejected.
- `risks[].id` is escaped in the default threat-model template. Risk
  identifiers follow the consumer-supplied `risk_pattern`, which is checked
  only for anchoring and length, so a permissive pattern let a risk ID
  break out of its code span and inject Markdown — including a link to a
  destination the `reference_hosts` allowlist would have refused.

## 0.1.0 - 2026-08-02

Initial release. The requirements-matrix mechanics were externalized from
the `tools/reqmatrix` utility in the
[Kivaar](https://github.com/sofired/kivaar) repository
([kivaar#35](https://github.com/sofired/kivaar/issues/35)); the
threat-model document type was added for
[kivaar#5](https://github.com/sofired/kivaar/issues/5) (tracked as
[#2](https://github.com/sofired/tracedoc/issues/2)) before the first
tag, so the multi-document design ships from the start.

### Added

- Document-type-generic `tracedoc` CLI: every document declares a required
  top-level `document_type`; `validate`, `render`, and `compare` dispatch
  on it, and `compare` requires matching types.
- Requirements-matrix schema v1: strict lexical decoding (size, depth,
  canonical and unique member names, unknown-field and trailing-value
  rejection) plus structural and cross-reference validation under
  consumer policy (required standards, per-standard source-host and
  local-path rules, verification-level vocabulary).
- Threat-model schema v1: assets, trust boundaries, and threats with
  schema-owned severity and disposition vocabularies, disposition
  coupling rules (rationale for accepted/transferred/avoided, mitigation
  evidence for mitigated, risk records for accepted), unconditional
  ownership, coverage rules for declared assets and boundaries, and the
  shared append-only supersession ledger. Ledger entries either name their
  active replacements or carry an explicitly empty replacement list to
  record withdrawal without a successor; withdrawals are immutable like
  every other entry.
- Cross-document link resolution: `validate -requirements` resolves a
  threat model's requirement links against a validated requirements
  matrix, rejecting unknown IDs and retired IDs (naming replacements);
  the flag is mandatory whenever links exist.
- Cross-version `compare` for both document types: deleted-ID, reused-ID,
  dropped- and changed-supersession rejection, and configurable
  document-version, review-date, and schema-version transition rules.
- Deterministic, injection-resistant Markdown rendering with per-type
  default templates, consumer template override via `-template`, `-check`
  freshness verification, and atomic output replacement.
- Single consumer configuration (version 1): shared project settings
  (owner/risk formats, workstreams, issue URL base, generator name,
  version-transition switches) plus per-document-type sections.
- `version` command reporting the tool, CLI contract, both document
  schemas, and configuration schema versions.
- Versioned contracts: [docs/cli.md](docs/cli.md),
  [docs/schema.md](docs/schema.md) (with per-type schema documents),
  [docs/config.md](docs/config.md), and the release and update policy in
  [docs/versioning.md](docs/versioning.md), including tag-triggered
  release verification.
- Binary distribution for non-Go consumers: the release workflow
  cross-compiles static binaries (linux, macOS, windows; amd64 and arm64)
  from the verified tag, generates a `SHA256SUMS` file, and attaches both
  to a draft GitHub release.

### Fixed (pre-release review hardening)

Defects found and fixed by multi-agent review before the first tag; none
ever shipped in a release:

- Link destinations percent-encode every control rune and the Unicode
  line and paragraph separators, closing a demonstrated Markdown
  structure-injection path and making the escape safe by construction.
- Every validated string — scalar fields and every free-form list item
  alike — is checked non-blank, bounded, and free of control and
  line-separator characters before any consumer-configured pattern runs,
  so permissive patterns can no longer admit values the renderer must not
  receive.
- Every escaping function neutralizes control runes, line and paragraph
  separators, and bidirectional overrides, so no document field can carry
  a terminal escape sequence or reorder surrounding text in generated
  Markdown regardless of which context it is emitted in.
- Renderers return a descriptive error instead of panicking when handed a
  document that skipped validation.
- Semantic-version components are compared as strings, eliminating an
  integer-overflow conflation of distinct oversized versions.
- A citation naming an unknown standard keeps its lexical URI checks and
  no longer cascades a redundant policy diagnostic.
