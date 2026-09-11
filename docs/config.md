# Consumer configuration, version 1

The configuration file carries every project-specific rule the tool
applies: shared settings at the top level plus one optional section per
document type. It is a bounded set of declarative knobs, deliberately not
a general-purpose validation language: new kinds of rules require a tool
change and a configuration schema revision, not cleverer configuration.

The file obeys the same lexical contract as the documents (strict
decoding, canonical member names, unknown members rejected) with a 1 MiB
size limit.

Example:

```json
{
  "config_version": 1,
  "milestone_pattern": "^M(?:[0-9]|1[01])$",
  "issue_pattern": "^#[1-9][0-9]*$",
  "risk_pattern": "^R(?:[1-9]|1[0-2])$",
  "workstreams": ["Protocol", "Platform"],
  "issue_url_base": "https://github.com/example/project/issues/",
  "generator_name": "tracedoc",
  "version_transitions": {
    "require_version_increase_on_change": true,
    "require_review_date_advance_on_change": true,
    "require_major_on_schema_change": true
  },
  "requirements": {
    "required_standards": ["EXAMPLE-CORE", "LOCAL-PLAN"],
    "standard_sources": [
      { "key": "EXAMPLE-CORE", "host": "standards.example.org" },
      { "key": "LOCAL-PLAN", "path": "../plan.md" }
    ],
    "verification_levels": ["unit", "integration", "adversarial"],
    "render": {
      "source_name": "matrix.json",
      "regenerate_command": "tracedoc render -config ... -doc matrix.json -output matrix.md",
      "check_command": "tracedoc render -config ... -doc matrix.json -output matrix.md -check"
    }
  },
  "threat_model": {
    "owner_pattern": "^@[a-z][a-z0-9-]*$",
    "document_statuses": ["draft", "accepted"],
    "control_statuses": ["planned", "in-progress", "implemented"],
    "evidence_levels": ["adversarial", "integration", "operational"],
    "evidence_statuses": ["planned", "deferred"],
    "reference_hosts": ["diagrams.example.org"],
    "coverage": {
      "require_asset_coverage": true,
      "require_boundary_coverage": true,
      "require_flow_coverage": true,
      "require_entry_point_coverage": true,
      "require_control_coverage": true,
      "require_risk_coverage": true,
      "require_evidence_per_threat": true,
      "require_criticality_for_every_priority": true
    },
    "limits": {
      "min_criticality_examples": 2,
      "min_top_abuse_paths": 3,
      "max_top_abuse_paths": 10
    },
    "render": {
      "source_name": "threats.json",
      "regenerate_command": "tracedoc render -config ... -doc threats.json -output threats.md",
      "check_command": "tracedoc render -config ... -doc threats.json -output threats.md -check"
    }
  }
}
```

## Shared top-level members

These describe the project — its work tracking, risk register, and issue
tracker — and apply identically to every document type:

| Member                | Rules                                                                        |
| --------------------- | ---------------------------------------------------------------------------- |
| `config_version`      | must be `1`                                                                  |
| `milestone_pattern`   | anchored regular expression (`^...$`, at most 256 bytes) for `owner.milestone` |
| `issue_pattern`       | anchored regular expression for `owner.issue`                                |
| `risk_pattern`        | anchored regular expression for risk-record identifiers                      |
| `workstreams`         | non-empty list of allowed `owner.workstream` values                          |
| `issue_url_base`      | HTTPS URL on a lowercase DNS host, path ending in `/`; no user information, port, query, or fragment, and no whitespace, backslash, or control characters. Issue references are appended without their `#` |
| `generator_name`      | generator name shown in generated headers; non-empty, at most 256 bytes, no control characters |
| `version_transitions` | boolean switches for the `compare` command; see [cli.md](cli.md), including the reachability caveat on `require_major_on_schema_change` |

## Per-document-type sections

Sections are optional; running a command against a document whose type has
no section is exit `2`.

### `requirements`

| Member                | Rules                                                                       |
| --------------------- | --------------------------------------------------------------------------- |
| `required_standards`  | required array (may be empty); unique standard keys; each key needs a `standard_sources` entry |
| `standard_sources`    | non-empty array; each entry has a unique `key` plus exactly one of `host` or `path` |
| `verification_levels` | non-empty list of allowed `planned_verification.levels` values (`^[a-z][a-z0-9-]*$`) |
| `render`              | presentation strings; see below                                             |

`standard_sources[]`:

- `host` — a lowercase multi-label DNS name. Citations and standard URIs for
  this key must be `https://` URIs on exactly this host, with no user
  information, port, or query.
- `path` — an exact relative path (no scheme, backslash, whitespace, or
  leading `/`). Citations for this key must use exactly this path, with an
  optional fragment. Use this for standards defined by a document inside the
  consuming repository; the path is resolved by Markdown viewers relative to
  the rendered file, not by the tool.

### `threat_model`

| Member              | Rules                                                                 |
| ------------------- | --------------------------------------------------------------------- |
| `owner_pattern`     | anchored regular expression for the accountable principal (`owner.principal` and the top-level `owner`). Only this document type has one — a requirements matrix's `owner` carries routing alone |
| `document_statuses` | non-empty list of allowed top-level `status` values (`^[a-z][a-z0-9-]*$`) |
| `control_statuses`  | non-empty list of allowed `controls[].status` values                  |
| `evidence_levels`   | non-empty list of allowed `planned_evidence[].level` values. Separate from the requirements section's `verification_levels`: different document types, unrelated vocabularies, neither derived from the other |
| `evidence_statuses` | non-empty list of allowed `planned_evidence[].status` values          |
| `reference_hosts`   | optional list of lowercase multi-label DNS names; hosts an external reference may use |
| `coverage`          | boolean switches for the declared-entity coverage rules; see below    |
| `limits`            | quantitative bounds on the review-guidance collections; see below     |
| `render`            | presentation strings; see below                                       |

These four vocabularies are a project's own workflow labels. The ones the
tool itself depends on — `likelihood`, `severity`, `priority`, `treatment`,
and `decisions[].status` — are schema-owned, as is every collection's
identifier format except risks.
[schema.md](schema.md#schema-owned-versus-configured-vocabularies) states
the rule that decides which side a vocabulary falls on.

`reference_hosts` is the allowlist for the `url` form of a diagram,
decision, or risk reference: a document may only point at a host declared
here. Omitting it, or leaving it empty, is legal and accepts
repository-relative references only. See
[schema-threat-model.md](schema-threat-model.md#references) for why that is
the reproducible default.

`coverage` holds seven switches — `require_asset_coverage`,
`require_boundary_coverage`, `require_flow_coverage`,
`require_entry_point_coverage`, `require_control_coverage`,
`require_risk_coverage`, and `require_evidence_per_threat`. What each one
rejects, and what "analysed" means, is specified once in
[schema-threat-model.md](schema-threat-model.md#coverage).

`require_criticality_for_every_priority` is the eighth switch: it demands a
`criticality` entry for each of the four `priority` levels.

Each switch defaults to `false` when omitted, so a project can adopt the
document type first and tighten coverage as the model fills in.

`limits`:

| Member                     | Bounds                                        |
| -------------------------- | --------------------------------------------- |
| `min_criticality_examples` | worked examples required per `criticality` entry |
| `min_top_abuse_paths`      | fewest entries in `top_abuse_path_links`      |
| `max_top_abuse_paths`      | most entries in `top_abuse_path_links`        |

Each defaults to `0`, which disables that bound. Negative values are
rejected, as is a maximum below its own minimum — a bound no document can
satisfy is a configuration error worth catching at load time rather than as
a puzzling rejection later. The schema says what these collections are; how
much of them a project expects is its own call.

### `render` (per section)

| Member               | Used for                                                                  |
| -------------------- | ------------------------------------------------------------------------- |
| `source_name`        | the source file name shown in the generated header                        |
| `regenerate_command` | shown in the generated footer and in `render -check` failure messages     |
| `check_command`      | shown in the generated footer                                             |

Every entry in a configured value list — `workstreams`,
`verification_levels`, and the four threat-model vocabularies — is also
non-blank, at most 256 bytes, free of control characters, and unique within
its list.

All render strings are single-line and bounded. Together with the shared
`issue_url_base` and `generator_name`, they are trusted consumer content:
the default templates insert them as inline code or link destinations
without content escaping, so review them like code.

## Templates

Each document type has an embedded default template. A consumer template
supplied with `render -template` replaces presentation entirely while
keeping validation and data mechanics. It must define a `document`
template whose body is more than whitespace — a file that puts all its
content in other `{{define}}` blocks is rejected rather than silently
rendering nothing — and the template file is limited to 1 MiB. It
receives:

- `.Document` — the validated document;
- `.Render` — the resolved presentation options;
- the document type's precomputed sections — for requirements:
  `.ApplicabilityCounts`, `.EvidenceStatusCounts`, `.BoundaryRequirements`,
  `.Ownership`, `.Standards`; for threat models: `.PriorityCounts`,
  `.TreatmentCounts`, `.Diagrams`, `.Assets`, `.Boundaries`, `.Flows`,
  `.EntryPoints`, `.Decisions`, `.Risks`, `.Controls`, `.Evidence`,
  `.Sections`, `.TopAbusePathLinks`, `.FocusPaths`; and
- the template functions `anchor`, `anchorHref`, `htmlText`, `inlineCode`,
  `inlineValues`, `issueURL`, `join`, `linkDestination`, `linkLabel`,
  `lower`, `owner`, `prose`, and `table`, plus `add1` for threat models
  (used to number ordered abuse paths and data-flow sequences from one).

Free-text document fields must always pass through the escaping functions
(`htmlText`, `prose`, `table`, `linkLabel`, `inlineValues`,
`linkDestination`, `inlineCode`, `anchor`); emitting document fields raw
lets document authors inject Markdown or HTML into the rendered output.
`TestRenderingEscapesUntrustedContent`, in both
`internal/render/requirements/requirements_test.go` and
`internal/render/threats/threats_test.go`, pins that the default templates
do so.

Anchors take two functions, one per context: `anchor` writes the `id`
attribute and `anchorHref` writes a same-document `#` destination. Use them
in that pairing and a link resolves — both case-fold, `anchor` escapes for
an HTML attribute, and `anchorHref` percent-encodes everything outside the
unreserved set, which a browser decodes before matching it against the
`id`. Swapping them silently breaks the link: an `id` is not percent-decoded,
and a Markdown destination ends at the first `)` or space. Both matter only
for `risks[].id`, the one consumer-patterned identifier — every other format
is schema-owned and passes through both functions unchanged. Do not build an
anchor out of `lower`: it does not escape. `TestAnchorPairAgrees` in
`internal/render/render_test.go` pins the agreement for identifiers that
would end either context, and `TestAnchorHrefLeavesSchemaOwnedIDsIntact`
beside it pins that a schema-owned identifier is not encoded.

A value may be emitted bare **only** when its character set is fixed by
something the document author cannot change. That covers stable ID fields
(`^[A-Z][A-Z0-9]*-[0-9]{3}$`), the other schema-owned collection
identifiers, standard keys (`^[A-Z][A-Z0-9]*(?:[-.][A-Z0-9]+)*$`), the
schema-owned enum vocabularies (`applicability`, `evidence_status`,
`likelihood`, `severity`, `priority`, `treatment`, `decisions[].status`),
and the configured value lists (`document_statuses`, `control_statuses`,
`evidence_levels`, `evidence_statuses`), whose entries config validation
constrains to `^[a-z][a-z0-9-]*$`.

It does **not** cover a value shaped by a consumer-supplied pattern.
`risk_pattern` is checked only for anchoring and length, so a risk ID can
legitimately contain backticks, pipes, or brackets; the default template
therefore routes `risks[].id` through `inlineCode` like any other free-form
value, and through `anchor` or `anchorHref` where it becomes an anchor;
`TestRiskIDsAreEscaped` and `TestRiskIDAnchorsAreEscaped` in
`internal/render/threats/threats_test.go` pin the two routes.
Treat every consumer-patterned field this way: the pattern is policy, not a
character-set guarantee. Anything else emitted raw is an injection risk. Section
layout and precomputed-section membership may change in minor releases;
pin the tool version to keep byte-identical output, and re-verify consumer
templates when updating. Every escaping function also neutralizes
invisible code points; see
[the shared lexical contract](schema.md#invisible-code-points).
