# Browser Server Roadmap

Browser Server is evolving into a privacy-first, self-hosted browser companion. The roadmap prioritizes trust and data reliability before adding features that improve daily use and discovery.

## Product principles

- **Private by default** — personal data remains self-hosted and protected at rest and in transit.
- **Reliable without constant connectivity** — browser activity must not disappear when the server is offline.
- **Easy to operate** — installation, updates, backups, and recovery should not require database expertise.
- **Useful every day** — new features should strengthen capture, retrieval, and action rather than add unrelated data domains.
- **Shared contracts** — the web app and browser extensions use the canonical shared types and API client.

## Completed foundation

- [x] Go REST API for todos, bookmarks, browsing history, wallet entries, users, screenshots, and usage analytics
- [x] Modular Go project structure with domain handlers and separate SQLite databases
- [x] Astro, Vue, and TailwindCSS web app with pages for all primary domains
- [x] Chromium and Firefox extensions with shared Vue UI and runtime logic
- [x] Automatic history capture and per-domain usage tracking
- [x] Bookmark, history, and wallet imports with duplicate handling
- [x] Todo screenshot capture and storage
- [x] Combined bookmark/history omnibox search with balanced source results
- [x] Shared TypeScript packages for API types, client methods, utilities, and extension code
- [x] Operator-level Bearer-token authentication with token generation and rotation
- [x] Public health endpoint plus logging and CORS middleware
- [x] Structured API errors and request validation
- [x] Configurable server port, token path, and data directory
- [x] Static frontend bundling through the PowerShell build script

## Phase 1 — Quality and security baseline

These tasks reduce the risk of regressions before storage and synchronization behavior changes.

### Automated tests

- [ ] Add unit tests for validation, query helpers, token handling, and pure utilities
- [ ] Add handler-level tests for authentication, structured errors, and CRUD operations
- [ ] Add integration tests for imports, bookmark tags, grouped history, search, and analytics
- [ ] Add migration tests using databases created by older application versions
- [ ] Add focused tests for extension synchronization queues and retry behavior

### Continuous integration

- [ ] Run `go test ./...`, `go vet ./...`, and the Go build in GitHub Actions
- [ ] Run frontend and extension type checks and production builds
- [ ] Fail on API type drift or package build failures
- [ ] Keep release publishing separate from required pull-request checks

### Encrypted password wallet

The wallet must not be promoted for sensitive credentials until encryption at rest is complete.

- [ ] Derive a vault key from a master password using Argon2id
- [ ] Encrypt passwords with authenticated encryption such as XChaCha20-Poly1305 or AES-GCM
- [ ] Never store or log the master password or derived plaintext key
- [ ] Add explicit vault lock, unlock, timeout, and locked-start behavior
- [ ] Migrate existing plaintext entries without silently losing data
- [ ] Keep wallet metadata hidden where practical while preserving useful website matching
- [ ] Ensure exports and backups do not expose plaintext credentials by default
- [ ] Document the threat model, recovery limitations, and safe deployment expectations

## Phase 2 — Data safety and dependable synchronization

### Backup, export, and restore

- [ ] Add a consistent server-side backup operation for every SQLite database and screenshot file
- [ ] Provide one-click backup download and restore from the web app
- [ ] Validate backup format and application compatibility before restoring
- [ ] Support encrypted backups, including wallet data
- [ ] Add scheduled backups with configurable retention
- [ ] Show backup date, size, status, and restore warnings
- [ ] Document manual recovery for damaged or partially restored data directories

### Durable offline synchronization

- [ ] Persist unsent history events in extension storage instead of discarding failed requests
- [ ] Generalize the existing usage buffer pattern into a durable write queue
- [ ] Retry automatically with bounded exponential backoff after the server reconnects
- [ ] Preserve ordering where it affects behavior and make retried writes idempotent
- [ ] Set queue size and age limits with clear retention behavior
- [ ] Distinguish connectivity, authentication, validation, and permanent server failures
- [ ] Show last successful sync, pending item count, and failed item count in the extension
- [ ] Add **Sync now** and safe retry/discard controls

## Phase 3 — Installation and operations

### Docker and first-run setup

- [ ] Add a multi-stage Dockerfile for the frontend and Go server
- [ ] Add Docker Compose with persistent data and token volumes
- [ ] Include health checks, restart policy, and documented environment configuration
- [ ] Add a first-run setup flow for server URL, token, user, and connection testing
- [ ] Document safe remote access through a private network or trusted reverse proxy
- [ ] Provide upgrade and rollback instructions that preserve user data

### Release automation

- [ ] Build versioned binaries and extension packages for supported platforms
- [ ] Publish checksums and concise release notes
- [ ] Surface the running version in the health response or settings UI
- [ ] Warn before upgrades that require a data migration or backup

## Phase 4 — Faster capture and retrieval

### Unified search and command palette

- [ ] Search bookmarks, grouped history, todos, and non-secret wallet metadata through one API
- [ ] Return a normalized, source-labeled result model with relevance and recency signals
- [ ] Add a web command palette with keyboard navigation
- [ ] Add quick actions such as open URL, create todo, complete todo, and copy username
- [ ] Extend omnibox search without removing source labels or balanced bookmark/history results
- [ ] Keep search local and avoid sending personal data to third-party services

### One-click browser capture

- [x] Add context-menu actions for **Save bookmark** and **Create todo from page**
- [x] Add configurable keyboard shortcuts for common capture actions
- [x] Pre-fill the current title, URL, domain, and selected text
- [x] Optionally attach a screenshot to a new todo
- [x] Detect duplicate bookmarks before creating another copy
- [x] Show immediate success or queued-for-sync feedback

### Bookmark cleanup and organization

- [ ] Detect duplicate URLs across folders and normalized URL variants
- [ ] Find dead links and redirects without overwhelming remote websites
- [ ] Offer bulk delete, move, tag, and merge operations
- [ ] Identify untagged and uncategorized bookmarks
- [ ] Add optional local folder and tag suggestions based on existing organization
- [ ] Require confirmation and provide a preview before destructive bulk changes

## Phase 5 — Actionable productivity insights

### Goals, limits, and focus sessions

- [ ] Let users classify domains as productive, neutral, or distracting
- [ ] Add daily and weekly browsing-time goals
- [ ] Add configurable domain limits and optional browser notifications
- [ ] Add focus sessions with temporary distracting-site alerts or blocking
- [ ] Show progress against goals in the web app and extension popup
- [ ] Keep controls opt-in and allow pause, snooze, and per-domain exceptions

### Reports and retention controls

- [ ] Add weekly summaries with trends, top domains, and goal progress
- [ ] Compare periods without exposing raw browsing data externally
- [ ] Add configurable history and analytics retention windows
- [ ] Allow users to exclude domains from capture and delete their existing records
- [ ] Export analytics in a portable format such as CSV or JSON

## Phase 6 — Optional intelligence

Only pursue these features after backups, encryption, reliable sync, and core search are stable.

- [ ] Suggest bookmark tags and folders using local rules first
- [ ] Detect stale bookmarks and repeated browsing patterns
- [ ] Generate optional local activity summaries
- [ ] Require explicit consent before using any remote AI provider
- [ ] Clearly display what data would leave the server and allow the feature to remain fully disabled

## Not currently planned

- Public social features or shared activity feeds
- Mandatory cloud accounts or hosted synchronization
- Advertising, behavioral profiling, or sale of browsing data
- Additional CRUD domains without a clear connection to browser capture, retrieval, or productivity
- Public internet exposure as the default deployment model
