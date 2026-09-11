# Threat-model schema, version 1

The threat model is a single JSON object with
`"document_type": "threat_model"`. It obeys the
[shared lexical contract and structural conventions](schema.md); this
document specifies the threat-model-specific members.

The schema is built so a reviewer can answer four questions from the
document alone: what the system is made of, who attacks it and with what
capabilities, which surfaces they can reach, and — for every threat — what
will demonstrate it is handled and who carries whatever risk remains.

## Top level

| Member             | Type   | Rules                                          |
| ------------------ | ------ | ---------------------------------------------- |
| `document_type`    | string | must be `"threat_model"`                       |
| `schema_version`   | number | must be `1`                                    |
| `document_version` | string | Semantic Versioning 2.0.0                      |
| `last_reviewed`    | string | RFC 3339 full date                             |
| `status`           | string | configured `document_statuses` vocabulary      |
| `owner`            | string | accountable principal; matches `owner_pattern` |
| `summary`          | string | non-empty                                      |
| `scope`            | object | see below                                      |
| `assumptions`      | array  | required, may be empty; see below              |
| `open_questions`   | array  | required, may be empty; unique strings         |
| `diagrams`         | array  | required, may be empty; see below              |
| `components`       | array  | non-empty; see below                           |
| `actors`           | array  | non-empty; see below                           |
| `attacker_model`   | object | see below                                      |
| `assets`           | array  | non-empty; see below                           |
| `trust_boundaries` | array  | non-empty; see below                           |
| `data_flows`       | array  | non-empty; see below                           |
| `entry_points`     | array  | non-empty; see below                           |
| `decisions`        | array  | required, may be empty; see below              |
| `risks`            | array  | required, may be empty; see below              |
| `controls`         | array  | non-empty; see below                           |
| `planned_evidence` | array  | non-empty; see below                           |
| `observability`    | array  | required, may be empty; see below              |
| `threats`          | array  | non-empty; see below                           |
| `criticality`      | array  | required, may be empty; see below              |
| `top_abuse_path_links` | array | required, may be empty; declared threat IDs   |
| `focus_paths`      | array  | required, may be empty; see below              |
| `supersessions`    | array  | required, may be empty; shared supersession rules with threat IDs |

Every array member is **required to be present**, even when empty. `compare`
distinguishes an omitted array from an empty one, so spelling out an empty
array today keeps the next revision from reading as a document change.

## Identifiers

Each collection has its own schema-owned prefix, so a link is unambiguous
about which collection it resolves against:

| Collection         | Format               |
| ------------------ | -------------------- |
| `assumptions`      | `^ASM-[0-9]{3}$`     |
| `components`       | `^COMP-[0-9]{3}$`    |
| `actors`           | `^ACTOR-[0-9]{3}$`   |
| `assets`           | `^AST-[0-9]{3}$`     |
| `trust_boundaries` | `^TB-[0-9]{3}$`      |
| `data_flows`       | `^DF-[0-9]{3}$`      |
| `entry_points`     | `^EP-[0-9]{3}$`      |
| `decisions`        | `^ADR-[0-9]{1,6}$`   |
| `controls`         | `^CTRL-[0-9]{3}$`    |
| `planned_evidence` | `^EVD-[0-9]{3}$`     |
| `observability`    | `^OBS-[0-9]{3}$`     |
| `risks`            | configured `risk_pattern` |
| `threats`          | stable-ID format; must not use any of the fixed prefixes above |

Decision IDs (`ADR-`, for *architecture decision record*) allow one to six
digits rather than exactly three, because projects commonly number decision
records from an existing issue or ADR sequence that has already passed 999.

Risk IDs are the one collection without a fixed prefix, so they are not
part of the threat-ID exclusion; document-wide uniqueness is what keeps a
risk ID from colliding with anything else.

Identifiers are unique **across the whole document**, not only within their
collection. The rendered companion anchors every declared record in one
namespace, so a reused identifier would silently collapse two anchors into
one. Distinct prefixes make a collision impossible between schema-owned
collections; the document-wide check is what governs consumer-patterned risk
IDs. `TestEveryDeclaredEntityIsAnchored` in
`internal/render/threats/threats_test.go` pins the anchoring, and
`TestCrossCollectionIDCollisionIsRejected` in
`internal/threats/validate_test.go` pins the document-wide check.

Anchors are case-folded, so uniqueness is too: `R1` and `r1` are rejected as
a collision even though they are different strings. Only a risk pattern can
express this, because every other identifier format is schema-owned and
uppercase. `TestCaseOnlyIDCollisionIsRejected` in
`internal/threats/validate_test.go` pins the folded comparison.

## References

`diagrams[]`, `decisions[]`, and `risks[]` each carry a reference to
supporting material. A reference declares **exactly one** of:

| Member | Rules                                                            |
| ------ | ---------------------------------------------------------------- |
| `path` | repository-relative; no scheme, backslash, whitespace, [invisible code point](schema.md#invisible-code-points), or leading `/` |
| `url`  | HTTPS, absolute path, host declared in the configured `reference_hosts` allowlist |

Supplying both, or neither, is an error.

A `path` is version-pinned with the document: the reviewed snapshot stays
reproducible, and `compare` sees it change. A `url` does not — an externally
hosted diagram can be edited or removed after review — which is why an
external host must be declared in configuration before one is accepted. A
consumer that declares no `reference_hosts` accepts repository-relative
references only.

This tool never opens either form. Parent segments (`../plan.md`) are
deliberately allowed because there is no local traversal to prevent while
the path is only echoed into a rendered link.

Diagram *generation* and diagram-source parsing are out of scope: the
schema carries the reference, and the renderer emits it as a link.

## `diagrams[]`

| Member       | Type   | Rules                                      |
| ------------ | ------ | ------------------------------------------ |
| `caption`    | string | non-empty                                  |
| `path`/`url` | string | exactly one; see [References](#references) |

A diagram entry has no identifier: nothing links to a diagram, so there is
nothing to resolve against. The caption is the link text in the rendered
companion, and the reference is its destination.

## `scope`

| Member         | Type  | Rules                    |
| -------------- | ----- | ------------------------ |
| `in_scope`     | array | non-empty, unique strings |
| `out_of_scope` | array | non-empty, unique strings |

Both halves are required. A scope that only says what was covered leaves a
reader unable to tell an unexamined area from an excluded one.

## `assumptions[]`

| Member      | Type   | Rules     |
| ----------- | ------ | --------- |
| `id`        | string | see above |
| `statement` | string | non-empty |
| `effect`    | string | non-empty; what follows if the assumption does not hold |

## `components[]`

| Member     | Type   | Rules                                    |
| ---------- | ------ | ---------------------------------------- |
| `id`       | string | see above                                |
| `name`     | string | non-empty                                |
| `zone`     | string | non-empty; the deployment or trust zone  |
| `purpose`  | string | non-empty                                |
| `evidence` | string | non-empty; where the component is described |

## `actors[]`

| Member        | Type   | Rules                                |
| ------------- | ------ | ------------------------------------ |
| `id`          | string | see above                            |
| `name`        | string | non-empty                            |
| `trust`       | string | non-empty; the trust placed in the actor |
| `description` | string | non-empty                            |

## `attacker_model`

| Member             | Type  | Rules                     |
| ------------------ | ----- | ------------------------- |
| `capabilities`     | array | non-empty, unique strings |
| `non_capabilities` | array | non-empty, unique strings |

`non_capabilities` is required, not decorative. A threat model without a
stated upper bound on the attacker cannot be reviewed for completeness,
because no reader can tell which omissions were decisions.

## `assets[]`

| Member        | Type   | Rules                                       |
| ------------- | ------ | ------------------------------------------- |
| `id`          | string | see above                                   |
| `name`        | string | non-empty                                   |
| `description` | string | non-empty                                   |
| `objective`   | string | non-empty; the security objective protection is measured against |

## `trust_boundaries[]`

| Member                 | Type   | Rules                                     |
| ---------------------- | ------ | ----------------------------------------- |
| `id`                   | string | see above                                 |
| `name`                 | string | non-empty                                 |
| `description`          | string | non-empty                                 |
| `source`               | string | declared component ID                     |
| `destination`          | string | declared component ID                     |
| `data`                 | array  | non-empty; what crosses the boundary      |
| `channels`             | array  | non-empty; how it crosses                 |
| `security_guarantees`  | array  | non-empty; what the boundary is meant to guarantee |
| `validation`           | array  | non-empty; how each guarantee is checked  |
| `implementation_state` | string | non-empty; how far the guarantees are actually built |
| `evidence`             | string | non-empty                                 |

`implementation_state` is free text rather than a vocabulary because it
records a project's own build progress, which no rule here keys off. It
exists so a boundary's *planned* guarantees can never be mistaken for
implemented ones.

## `data_flows[]`

| Member       | Type   | Rules                                   |
| ------------ | ------ | --------------------------------------- |
| `id`         | string | see above                               |
| `name`       | string | non-empty                               |
| `sequence`   | array  | non-empty, ordered steps                |
| `boundaries` | array  | non-empty; declared trust-boundary IDs  |
| `data`       | array  | non-empty; what the flow carries        |

## `entry_points[]`

| Member     | Type   | Rules                               |
| ---------- | ------ | ----------------------------------- |
| `id`       | string | see above                           |
| `surface`  | string | non-empty                           |
| `reached`  | string | non-empty; how the surface is reached |
| `boundary` | string | declared trust-boundary ID          |
| `flows`    | array  | non-empty; declared data-flow IDs   |
| `notes`    | string | non-empty                           |
| `evidence` | string | non-empty                           |

## `decisions[]`

| Member      | Type   | Rules                                                 |
| ----------- | ------ | ----------------------------------------------------- |
| `id`        | string | see above                                             |
| `title`     | string | non-empty                                             |
| `status`    | string | `proposed`, `accepted`, `rejected`, or `superseded` (schema-owned) |
| `path`/`url`| string | exactly one; see [References](#references)            |

## `risks[]`

| Member      | Type   | Rules                                      |
| ----------- | ------ | ------------------------------------------ |
| `id`        | string | configured `risk_pattern`                  |
| `title`     | string | non-empty                                  |
| `path`/`url`| string | exactly one; see [References](#references) |

Risk identifiers follow a consumer pattern because they come from the
project's existing risk register, which this tool does not own.

## `controls[]`

| Member                | Type   | Rules                                              |
| --------------------- | ------ | -------------------------------------------------- |
| `id`                  | string | see above                                          |
| `title`               | string | non-empty                                          |
| `description`         | string | non-empty                                          |
| `status`              | string | configured `control_statuses` vocabulary           |
| `owner`               | object | see [Ownership](#ownership)                        |
| `requirement_links`   | array  | required, may be empty; requirement IDs resolved against the requirements matrix by `validate -requirements` |
| `decision_links`      | array  | required, may be empty; declared decision IDs      |
| `risk_links`          | array  | required, may be empty; declared risk IDs          |
| `evidence_links`      | array  | required, may be empty; declared planned-evidence IDs |
| `implementation_note` | string | non-empty                                          |

Each link category may be empty, but a control must carry **at least one**
link across all four. A control that traces to nothing records no obligation
and cannot be reviewed.

Requirement links live here rather than on threats: a threat is handled
because a control addresses it, and the control is what a requirement
obliges the project to build. Links must name *active* requirement IDs;
links to retired IDs are rejected with their replacements named.

## `planned_evidence[]`

| Member         | Type   | Rules                                       |
| -------------- | ------ | ------------------------------------------- |
| `id`           | string | see above                                   |
| `title`        | string | non-empty                                   |
| `level`        | string | configured `evidence_levels` vocabulary     |
| `status`       | string | configured `evidence_statuses` vocabulary   |
| `description`  | string | non-empty                                   |
| `owner`        | object | see [Ownership](#ownership)                 |
| `threat_links` | array  | non-empty; declared threat IDs              |

## `observability[]`

| Member            | Type   | Rules                                        |
| ----------------- | ------ | -------------------------------------------- |
| `id`              | string | see above                                    |
| `surface`         | string | non-empty                                    |
| `signals`         | array  | non-empty; what operations must be able to see |
| `redaction`       | array  | non-empty; what must never be logged          |
| `alert_condition` | string | non-empty                                    |
| `control_links`   | array  | non-empty; declared control IDs               |

`redaction` is required alongside `signals` so an observability requirement
cannot be written as "log everything about this surface".

## `threats[]`

| Member                    | Type   | Rules                                            |
| ------------------------- | ------ | ------------------------------------------------ |
| `id`                      | string | [stable-ID format](schema.md#shared-structural-conventions), unique, never renumbered or reused; must not use a reserved collection prefix |
| `title`                   | string | non-empty                                        |
| `source`                  | string | non-empty; who or what originates the threat     |
| `prerequisites`           | string | non-empty; what the attacker needs first         |
| `action`                  | string | non-empty; what the attacker does                |
| `impact`                  | string | non-empty; what it costs if it succeeds          |
| `abuse_path`              | array  | non-empty, ordered steps                         |
| `likelihood`              | string | `low`, `medium`, or `high` (schema-owned)        |
| `likelihood_rationale`    | string | non-empty                                        |
| `severity`                | string | `low`, `medium`, or `high` (schema-owned)        |
| `impact_rationale`        | string | non-empty; the rationale **for `severity`**, which is assessed on impact — not a rationale for the `impact` field above |
| `priority`                | string | `critical`, `high`, `medium`, or `low` (schema-owned); see below |
| `treatment`               | string | `mitigate`, `accept`, `avoid`, or `transfer` (schema-owned) |
| `treatment_rationale`     | string | required for `accept`, `avoid`, and `transfer`; optional otherwise |
| `owner`                   | object | see [Ownership](#ownership)                      |
| `residual_risk`           | string | non-empty; what remains after treatment          |
| `existing_controls`       | string | non-empty                                        |
| `gaps`                    | string | non-empty                                        |
| `recommended_mitigations` | string | non-empty                                        |
| `detection_ideas`         | string | non-empty                                        |
| `actor_links`             | array  | non-empty; declared actor IDs                    |
| `asset_links`             | array  | non-empty; declared asset IDs                    |
| `boundary_links`          | array  | non-empty; declared trust-boundary IDs           |
| `flow_links`              | array  | required, may be empty; declared data-flow IDs   |
| `control_links`           | array  | required, may be empty; declared control IDs     |
| `risk_links`              | array  | required, may be empty; declared risk IDs        |
| `evidence_links`          | array  | required, may be empty; declared planned-evidence IDs |

`owner`, `treatment`, and `residual_risk` are unconditionally required. That
subsumes the common governance rule that no critical or high threat lacks an
owner: making the rule conditional on priority would mean the priority
assessment itself decides whether anyone is accountable.

`priority` is the author's overall ranking, informed by `likelihood` and
`severity` together. The tool does **not** derive it or check it against
them: there is no fixed mapping from a three-by-three grid onto four
priority levels, and which cell deserves which ranking is a judgement a
project makes for itself. The document records that judgement, and the
rationale fields are where it is justified. `priority` is what orders the
rendered companion's sections.

### Treatment coupling

A treatment records a decision, so the record that decision implies must be
present:

- `mitigate` requires at least one `control_links` entry — naming what will
  do the mitigating.
- `accept` requires at least one `risk_links` entry — charging the residual
  risk to the register rather than leaving it unowned — in addition to the
  rationale.
- `accept`, `avoid`, and `transfer` each require `treatment_rationale`:
  all three record a decision not to build a control.

## `criticality[]`

| Member       | Type   | Rules                                              |
| ------------ | ------ | -------------------------------------------------- |
| `level`      | string | a `priority` value (schema-owned); unique          |
| `definition` | string | non-empty; what this level means for this project  |
| `examples`   | array  | non-empty, unique; worked examples at this level   |

`priority` is a schema-owned vocabulary, but what separates one level from
the next is a project's own judgement about blast radius. Without that
judgement written down, a reader cannot tell a miscalibrated ranking from an
honest disagreement about the scale — which makes the model's most
consequential field the one nobody can check.

The level is the record's key, so there is no separate identifier — nothing
links to a calibration entry. Duplicated levels are rejected.

Two bounded configuration switches govern how much calibration a project
expects: `require_criticality_for_every_priority` demands an entry for each
of the four levels, and `min_criticality_examples` sets a floor on
`examples`. Both are [configuration](config.md#threat_model), because the
right answer differs between a small service and a platform.

## `top_abuse_path_links`

A required array, possibly empty, of unique declared threat IDs: the abuse
paths a reviewer should follow first. Bounded by the configured
`min_top_abuse_paths` and `max_top_abuse_paths`.

This is deliberately **not** derived from `priority`. "The paths to read
first" is an editorial judgement about narrative order, and a list that
merely repeats every `critical` threat has made no such judgement. A model
with fifteen threats and three genuinely instructive attack narratives
should say so.

## `focus_paths[]`

| Member         | Type   | Rules                                        |
| -------------- | ------ | -------------------------------------------- |
| `path`         | string | repository-relative, unique; same rules as a [reference path](#references) |
| `why`          | string | non-empty; why this location deserves scrutiny |
| `threat_links` | array  | non-empty, unique; declared threat IDs        |

The model's link from a threat to where that threat actually lives. No
other collection carries it: `components` describe what the system is made
of and `planned_evidence` describes what will test it, but neither tells a
reviewer what to read. The path is the record's key, and as with every path
this schema carries, the tool never opens it.

## Ownership

`threats[]`, `controls[]`, and `planned_evidence[]` each carry an `owner`
object:

| Member       | Type            | Rules                                       |
| ------------ | --------------- | ------------------------------------------- |
| `principal`  | string          | the accountable principal; matches `owner_pattern` |
| `milestone`  | string          | matches `milestone_pattern`                 |
| `issue`      | string \| null  | matches `issue_pattern` when present        |
| `workstream` | string          | configured `workstreams` vocabulary         |

`principal` is separated from the routing members deliberately. Milestone,
issue, and workstream change as plans change; the person answerable for the
residual risk does not, and an unattributed residual risk is one nobody has
agreed to carry.

`principal`, `milestone`, and `issue` are each first checked non-blank,
bounded, and free of
[invisible code points](schema.md#invisible-code-points), *then* matched
against the configured pattern. A permissive consumer pattern therefore
cannot admit a value the renderer must not receive. `workstream` is checked
by membership in the configured list alone — it needs no lexical pass
because a closed vocabulary cannot carry a value the project did not
write.

## Coverage

Coverage rules ask whether something the document declared was actually
analysed. Each is a named switch in
[configuration](config.md#threat_model), so a consumer writing a model
incrementally can turn one off rather than deleting the entity:

| Switch                          | Rejects                                                |
| ------------------------------- | ------------------------------------------------------ |
| `require_asset_coverage`        | an asset no threat links to                            |
| `require_boundary_coverage`     | a trust boundary no threat links to                    |
| `require_flow_coverage`         | a data flow no threat links to                         |
| `require_entry_point_coverage`  | an entry point no threat reaches (see below)           |
| `require_control_coverage`      | a control no threat links to                           |
| `require_risk_coverage`         | a risk no threat links to                              |
| `require_evidence_per_threat`   | a threat no planned evidence names                     |
| `require_criticality_for_every_priority` | a `priority` level with no [calibration entry](#criticality) |

Coverage is credited **only** from `threats[]` and, for the last rule, from
`planned_evidence[].threat_links`. A boundary named by a data flow, or a
control named by an observability record, has been wired into the model but
not reasoned about as an attack surface. Counting those would make every
rule vacuously true, because the architecture graph references its own
members by construction.

An **entry point** counts as analysed only when one threat both crosses its
`boundary` *and* travels one of its `flows`. Matching either half alone is a
false positive worth rejecting: a threat that crosses the same boundary by a
different route has not examined this surface, and accepting it would
certify an entry point nobody reviewed.

## Why there is no `quality_checks`

A threat model may be tempted to carry its own completeness self-assessment
— a list of claims like "all entry points are covered", each marked
complete. This schema deliberately has no such collection, because for the
claims worth making the tool already proves them, and a hand-maintained
claim can go stale and contradict the validator that ran beside it:

| A claim of this kind | What already enforces it |
| -------------------- | ------------------------ |
| All entry points are covered | `require_entry_point_coverage` |
| Every trust boundary appears in a threat | `require_boundary_coverage` |
| Every asset, flow, control, and risk is analysed | the remaining [coverage switches](#coverage) |
| Every threat has planned evidence | `require_evidence_per_threat` |
| Requirement traceability resolves | `validate -requirements` |
| Assumptions and open questions are explicit | `assumptions` and `open_questions` are required members |

The claims left over — "the review reflected a conversation with the
architect", "both deployment shapes were considered" — are process notes
about how a review was conducted rather than statements about the system.
They belong in the pull request that changed the model, where they can be
read against the diff.

Note that this is a schema-or-nothing choice, not a schema-versus-template
one. Unknown members are rejected
([shared lexical contract](schema.md#shared-lexical-contract)), so a
collection absent from the schema cannot appear in a document at all, and a
consumer template has nothing to render. Anything a project needs to record
has to be a schema member.

## Which vocabularies are schema-owned

`likelihood`, `severity`, `priority`, `treatment`, and `decisions[].status`
are fixed by this schema. `document_statuses`, `control_statuses`,
`evidence_levels`, and `evidence_statuses` are
[configured](config.md#threat_model), as is the `risks[]` identifier
format.

[schema.md](schema.md#schema-owned-versus-configured-vocabularies) states
the rule that decides which side a vocabulary falls on, and why.
