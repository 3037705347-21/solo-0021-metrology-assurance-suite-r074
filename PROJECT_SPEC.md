# Metrology Assurance Suite

## Project Goal

Metrology Assurance Suite is a local HTTP service that keeps a traceable record
of laboratory measurement assurance work. It registers devices, opens a
verification case for one device, records technician measurements, requests a
repeat measurement when tolerance is exceeded, and seals the final evidence
record after review.

The service is intentionally self-contained. It uses an in-memory repository
with atomic copy-on-write mutations and does not require a database or an
external service.

## Users

- Laboratory steward: registers devices and opens verification cases.
- Measurement technician: records observed values for an open case.
- Quality verifier: reviews accepted measurements and seals the case.

## Core Entities

- Device: a laboratory asset with a unique asset tag, model, room, steward, and
  lifecycle state.
- VerificationCase: one bounded assurance activity for a device, with a nominal
  value, tolerance, current state, version, and assigned steward.
- Measurement: an observed value, computed deviation, acceptance decision,
  technician, and timestamp.
- EvidenceSeal: the closing record for a reviewed case, including the selected
  accepted measurement, a lifecycle status, and a traceable link to the seal it
  supersedes. Seals are never deleted; a revoked seal stays readable.
- SealRevocation: the independent fact that withdraws a seal, preserving the
  revoked seal, the responsible actor, the reason, and the count of
  measurements the revocation supersedes.
- AuditEvent: an append-only trace of meaningful case and device changes.

## Workflows

### 1. Register Device

The steward sends `POST /devices` with an asset tag, model, room, and steward.
The service validates the request, normalizes the asset tag, rejects duplicate
tags, and creates an active device.

### 2. Open Verification Case

The steward sends `POST /cases` with a device ID, nominal value, tolerance, and
owner. The service verifies that the device exists, is active, and has no
unfinished case. A valid request creates a case in `awaiting_measurement`.

### 3. Record Measurement

A technician sends `POST /cases/{id}/measurements`. The case must be
`awaiting_measurement`. The service computes the signed deviation from the
nominal value. An in-tolerance measurement moves the case to
`review_ready`; an out-of-tolerance measurement moves it to
`recheck_required`.

### 4. Reopen Verification Case

The steward sends `POST /cases/{id}/reopen` after an out-of-tolerance result.
Only `recheck_required` cases may move back to `awaiting_measurement`. The
service records the actor and reason in the audit trail.

### 5. Seal Verification

The verifier sends `POST /cases/{id}/seal` after an accepted measurement. Only
`review_ready` cases may be sealed. The service selects the accepted
measurement with the smallest absolute deviation, creates exactly one active
evidence seal, moves the case to `sealed`, and rejects later mutations. When
the case was previously revoked, only measurements recorded after the latest
revocation are eligible, and the new seal references the revoked seal it
supersedes.

### 6. Revoke Evidence Seal

The verifier sends `POST /cases/{id}/revoke-seal` with an actor and a reason
when the measurement basis or equipment status behind a seal is found
defective. Only `sealed` cases with an active seal may be revoked. In one
atomic commit the service records an independent seal revocation fact, marks
the seal `revoked` without deleting it, moves the case back to
`awaiting_measurement`, and appends a `seal_revoked` audit event. The case can
then be measured again and re-sealed; the replacement seal supersedes the
revoked one, so history explains how the new evidence succeeds the old.

## State And Rules

Device state:

- `active -> retired`
- `retired` is terminal.

VerificationCase state:

- `awaiting_measurement -> review_ready`
- `awaiting_measurement -> recheck_required`
- `recheck_required -> awaiting_measurement`
- `review_ready -> sealed`
- `sealed -> awaiting_measurement` (seal revocation)

EvidenceSeal state:

- `active -> revoked`
- `revoked` is terminal; revoked seals are preserved, never deleted.

Required invariants:

- Asset tags are unique after case-insensitive normalization.
- A device has at most one unfinished verification case.
- Measurements are accepted only while a case is awaiting measurement.
- A tolerance must be positive.
- A case has at most one active evidence seal at any time.
- A revocation requires an actor and a reason and forms exactly one fact per
  seal; a duplicate revocation is rejected.
- A seal created after a revocation supersedes the latest revoked seal and
  selects only measurements recorded after that revocation.
- A sealed case cannot accept measurements, reopen, or seal again until its
  seal is revoked.
- Every mutation that changes a case also appends an audit event in the same
  repository commit.

## Modules

- `cmd/metrology-service`: process entrypoint and graceful HTTP shutdown.
- `cmd/workflowcheck`: bounded smoke checks that call the public HTTP API.
- `internal/httpapi`: routing, request decoding, and stable error mapping.
- `internal/assurance`: workflow orchestration and atomic decisions.
- `internal/domain`: entities, validation, state transitions, and errors.
- `internal/repository`: mutex-protected in-memory persistence with copy-on-write
  commits.
- `internal/contracts`: request and response contracts shared by the HTTP layer.

## Public Interfaces

- `GET /healthz`
- `POST /devices`
- `GET /devices/{id}`
- `POST /cases`
- `GET /cases/{id}`
- `POST /cases/{id}/measurements`
- `POST /cases/{id}/reopen`
- `POST /cases/{id}/seal`
- `POST /cases/{id}/revoke-seal`
- `GET /cases/{id}/events`

## Validation Plan

The baseline build command is `go build ./...`. Each declared workflow has a
bounded production smoke command under `cmd/workflowcheck`. The command starts
an HTTP test server in the same process, calls the public routes, and verifies
both success and failure behavior. The seal check also races two concurrent
seal requests to prove that the repository mutation lock permits exactly one
seal. The revoke check revokes a seal, proves the revocation fact and the
preserved revoked seal stay readable, rejects a duplicate revocation, races
two concurrent re-seals, and verifies the replacement seal supersedes the
revoked one and selects only post-revocation evidence.

Tests are intentionally deferred for this initialization baseline. The later
engineering-task stage will add unit and integration tests, including red/green
verification for the workflows and their failure paths.
