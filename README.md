# Offline-First Backend (Go + Postgres)

This backend implements a **deterministic, offline-first sync engine**
used by the iOS Knowledge Vault app.

It is designed for clients that:

- work offline for long periods
- apply local changes optimistically
- later reconcile safely with the server

Correctness and data integrity are prioritized over convenience.

---

## 🧠 Architecture Overview

### Single Source of Truth

- The server is the **source of truth for ordering**
- Clients are stateful and version-aware
- All writes are validated against server versions

The backend never guesses client intent.
All ambiguity is resolved explicitly via conflicts.

---

## 🔢 Global Versioning Model

- Every mutation allocates a **monotonically increasing global version**
- Versions are:
  - global (not per-item)
  - strictly increasing
  - never reused

The global version is stored in the `sync_state` table and is used for:

- conflict detection
- incremental sync
- deterministic ordering across devices

---

## 🗄️ Data Model

### `items`

| Column     | Purpose           |
| ---------- | ----------------- |
| id         | UUID              |
| user_id    | Item owner        |
| type       | Item type         |
| title      | Title             |
| content    | Content           |
| version    | Global version    |
| deleted    | Soft delete flag  |
| created_at | Creation time     |
| updated_at | Last modification |

### `sync_state`

| Column         | Purpose                |
| -------------- | ---------------------- |
| latest_version | Global version counter |

---

## 🔄 Mutation Semantics

### Create

- Inserts a new item
- Allocates a new global version
- `deleted = false`

---

### Update (Resurrection Supported)

- Requires client to send `version`
- Fails with conflict if versions mismatch
- Always:
  - allocates a new global version
  - clears `deleted = false`

This allows **resurrection of soft-deleted items**, which is required
for conflict resolution (Keep Local / Merge).

---

### Delete (Soft Delete)

- Requires client version
- Marks item as `deleted = true`
- Allocates a new global version
- Item is never physically removed

---

## 🔌 API Endpoints

### Create Item

POST /items

---

### Update Item

PUT /items/{id}

Request must include:
{
"version": <last_known_version>
}

---

### Delete Item

DELETE /items/{id}?version=<version>

---

### Incremental Sync

GET /changes?since_version=<version>

Response:
{
"latest_version": 42,
"items": [ ... ]
}

Returns **all changes** where `version > since_version`.

---

## ⚔️ Conflict Handling

If client version ≠ server version, the server responds with:

{
"error": "version_conflict",
"server_item": { ... }
}

The server does **not** auto-merge.
The client must resolve conflicts explicitly.

---

## 🧪 Critical Invariants (DO NOT BREAK)

- Every mutation must allocate a global version
- UPDATE must resurrect deleted items
- Reads and writes during mutations must use the same transaction
- `/changes` must return all items with `version > since_version`
- Deleted items must continue to appear in `/changes`

Breaking any invariant will cause client-side sync corruption.

---

## 🚀 Running Locally

docker-compose up
go run cmd/server/main.go

---

## 🧠 Design Philosophy

This backend assumes:

- conflicts are normal
- clients are intelligent
- correctness beats convenience
- explicit failure is better than silent corruption

This strictness is intentional.
