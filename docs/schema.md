# Document schemas

The tool reads two document types, each with its own independently
versioned schema:

- [schema-requirements.md](schema-requirements.md) — the requirements
  matrix (`"document_type": "requirements"`), schema version 1.
- [schema-threat-model.md](schema-threat-model.md) — the system threat
  model (`"document_type": "threat_model"`), schema version 1.

Schema changes follow the policy in [versioning.md](versioning.md).

## Shared lexical contract

Every document, regardless of type, must satisfy before structural
validation:

- at most 8 MiB;
- nesting depth at most 16;
- object member names match `^[a-z][a-z0-9_]*$`;
- no duplicate member names within an object;
- no members outside the document's schema;
- exactly one top-level JSON value; and
- a top-level `document_type` member naming the schema family.

Every declared array of strings rejects duplicate entries, so a list can
never assert the same thing twice.

Every validated string field is non-blank (a value of only whitespace is
rejected), limited to 16 KiB, and free of the code points below. All string fields are
plain text: authored Markdown and HTML are not supported, and the renderer
escapes content so it cannot introduce Markdown structure or raw HTML.
Formatted content requires a future, explicitly named field and a separate
validation contract.

### Invisible code points

Two layers keep code points that survive Markdown escaping from changing
how output is interpreted or displayed:

- **Validation** rejects control characters (Unicode category Cc, which
  includes NEL) and the line and paragraph separators (Zl/Zp) in every
  validated string — scalar fields and every free-form list item alike.
  The diagnostic reads `contains a control or line-separator character`.
  `TestControlFreeString` and `TestStringListRejectsControlBearingItems`
  in `internal/check/check_test.go` pin the rejection and its diagnostic
  for a scalar and for a list item.
- **Rendering** additionally neutralizes those code points *and* the
  bidirectional embedding, override, and isolate controls
  (U+202A–U+202E, U+2066–U+2069) in every emission context: prose, table
  cells, inline code, link labels, HTML text, and link destinations
  (percent-encoded there rather than dropped). This is defense in depth
  for the validated classes and the primary defense for bidirectional
  controls. `TestEscapersNeutralizeInvisibleRunes` in
  `internal/render/render_test.go` pins this for each of those contexts.

Bidirectional controls are deliberately **not** rejected at validation:
right-to-left content is legitimate in prose, so the boundary is drawn at
rendering, where reordering would mislead a reader of the generated
document. Other format-category (Cf) code points, such as zero-width
joiners, are permitted in both layers.

## Words the two document types use differently

Two terms appear in both schemas meaning different things. Neither pair is
linked, and no value of one is ever resolved against the other:

| Term | Requirements matrix | Threat model |
| ---- | ------------------- | ------------ |
| evidence | `planned_verification.evidence`, a list of free-form identifiers naming what will demonstrate a requirement is met | `planned_evidence[]`, declared records with identifiers, levels, owners, and resolved threat links |
| owner | routing only — milestone, issue, workstream | routing plus an accountable `principal` who carries the residual risk |

The configured vocabularies follow the same split: `verification_levels`
belongs to the requirements section and `evidence_levels` to the
threat-model section. A project may give them the same values; nothing
derives one from the other.

## Schema-owned versus configured vocabularies

Some enumerated vocabularies are fixed by a schema; others are declared in
[consumer configuration](config.md). The split is not arbitrary, and it is
the same for every document type:

- **Schema-owned** when the tool itself depends on the exact values —
  either a validation rule branches on them (a threat model's `treatment`
  decides which link category becomes mandatory) or the renderer does
  (`priority` and `treatment` drive the fixed section order and grouping of
  the generated companion). A vocabulary is also schema-owned when it is a
  fixed scale or an external standard the tool reports rather than a
  project's own labels: a threat model's `likelihood` and `severity`
  ratings mean the same thing in every project that uses this tool, and
  `decisions[].status` is the ordinary decision-record lifecycle.
- **Configured** when the values are a project's own workflow labels and
  nothing in the tool reads them beyond membership — a threat model's
  document, control, and evidence statuses, its evidence levels, and a
  requirements matrix's verification levels.

The practical test when adding one: if you can rename every value without
changing any behaviour except which strings are accepted, it belongs in
configuration. Adding a vocabulary that a rule branches on is a schema
revision.

This is the same policy-versus-mechanics boundary that keeps a
general-purpose rule language out of the configuration: the tool owns
mechanics, the consumer owns policy.

## Shared structural conventions

Both schemas share these members and semantics:

| Member             | Rules                                                        |
| ------------------ | ------------------------------------------------------------ |
| `document_type`    | `"requirements"` or `"threat_model"`, fixed per schema       |
| `schema_version`   | the schema version the document conforms to                  |
| `document_version` | Semantic Versioning 2.0.0                                    |
| `last_reviewed`    | RFC 3339 full date (`YYYY-MM-DD`)                            |
| `supersessions`    | required array (may be empty), append-only across accepted revisions: `retired_id` (unique, not active), required `replacement_ids` (all entries active; an explicitly empty array records withdrawal without a successor), non-empty `rationale` |

The `owner` member is not one shape across the document types. A
requirements matrix's `owner` object carries routing only (milestone,
issue, workstream). A threat model's `owner` object adds a required
accountable `principal`, and its top-level `owner` is a bare principal
string. Only the threat model has a notion of an accountable person, which
is why `owner_pattern` is configured in that document type's section
rather than shared.

Stable IDs (requirement and threat IDs) share the format
`^[A-Z][A-Z0-9]*-[0-9]{3}$` and are never renumbered or reused. The threat
model additionally declares several other entity collections, each with its
own reserved prefix within that same format
([schema-threat-model.md](schema-threat-model.md#identifiers)); retiring
one requires a retained supersession — naming its replacements when the
obligation was split, merged, or replaced, or with an empty replacement
list when it was withdrawn outright. Withdrawals are as immutable as any
other ledger entry: later granting a withdrawn ID replacements is a
rejected ledger edit. The `compare` command enforces these
guarantees across revisions for both document types. Retiring an ID that
is itself listed as another entry's replacement is not expressible in
schema 1 for either document type; see
[schema-requirements.md](schema-requirements.md) for the worked example
and [issue #3](https://github.com/sofired/tracedoc/issues/3) for the
schema-2 candidate
([cli.md](cli.md#tracedoc-compare--config-path--baseline-path--candidate-path)).
