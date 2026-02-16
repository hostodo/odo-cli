# Requirements: Hostodo CLI v2.0

**Defined:** 2026-02-15
**Core Value:** CLI users can securely authenticate and manage VPS instances using intuitive commands with hostnames

## v1 Requirements

Requirements for this release. Each maps to roadmap phases.

### Security & Session Management

- [ ] **SEC-01**: Device codes use 12-character alphanumeric format (e.g., A3K9-M7P2-X4Q8) with 71.75 bits entropy
- [ ] **SEC-02**: CLI sessions track `last_used` datetime field, updated on every API call
- [ ] **SEC-03**: Session revocation uses soft-delete with `revoked_at` timestamp field (audit compliance)
- [ ] **SEC-04**: User can list active CLI sessions with `hostodo auth sessions`
- [ ] **SEC-05**: User can revoke specific session with `hostodo auth revoke <session-id>`
- [ ] **SEC-06**: Existing 8-digit device codes continue working during 30-day migration period

### Hostname Support

- [ ] **HOST-01**: User can get instance details by hostname: `hostodo get myserver`
- [ ] **HOST-02**: User can start instance by hostname: `hostodo start myserver`
- [ ] **HOST-03**: User can stop instance by hostname: `hostodo stop myserver`
- [ ] **HOST-04**: User can reboot instance by hostname: `hostodo reboot myserver`
- [ ] **HOST-05**: Instance list displays hostname column in output table
- [ ] **HOST-06**: Instance get command shows hostname in details
- [ ] **HOST-07**: CLI caches hostname-to-ID resolution for current session
- [ ] **HOST-08**: CLI handles duplicate hostnames with interactive disambiguation prompt

### Command Usability

- [ ] **CMD-01**: User can list instances with `hostodo list` (default instance scope)
- [ ] **CMD-02**: User can get instance with `hostodo get <name|id>` (default instance scope)
- [ ] **CMD-03**: Existing `hostodo instances list` commands continue working (backward compat)
- [ ] **CMD-04**: Shell completion works for bash, zsh, fish (Cobra built-in)
- [ ] **CMD-05**: Non-interactive mode via `--yes` flag skips confirmation prompts
- [ ] **CMD-06**: Commands output machine-readable YAML format with `--output yaml`

### Instance Deployment

- [ ] **DEPLOY-01**: User can deploy instance with interactive wizard: `hostodo deploy myserver`
- [ ] **DEPLOY-02**: Deployment wizard prompts for plan selection with pricing display
- [ ] **DEPLOY-03**: Deployment wizard prompts for region selection
- [ ] **DEPLOY-04**: Deployment wizard prompts for template/OS selection
- [ ] **DEPLOY-05**: Deployment wizard prompts for SSH key (existing or upload)
- [ ] **DEPLOY-06**: User can deploy with flags for automation: `hostodo deploy --plan X --region Y --template Z`
- [ ] **DEPLOY-07**: Deployment shows progress indicators with status polling
- [ ] **DEPLOY-08**: Deployment displays instance details when provisioning completes

## v2 Requirements

Deferred to future release. Tracked but not in current roadmap.

### Advanced Filtering

- **FILTER-01**: Filter instances by status: `hostodo list --status running`
- **FILTER-02**: Filter instances by region: `hostodo list --region nyc1`
- **FILTER-03**: Filter instances by tag: `hostodo list --tag env:prod`

### Configuration & Defaults

- **CONFIG-01**: User can set default region in config file
- **CONFIG-02**: User can set default plan in config file
- **CONFIG-03**: User can set preferred output format in config file

### Advanced Operations

- **OPS-01**: User can bulk stop instances by tag
- **OPS-02**: User can resize instance (change plan)
- **OPS-03**: User can take instance snapshot
- **OPS-04**: User can restore instance from snapshot

## Out of Scope

Explicitly excluded. Documented to prevent scope creep.

| Feature | Reason |
|---------|--------|
| Embedded SSH client | Users already have SSH tools, adds maintenance burden |
| Plugin system | Premature - no demand signal, high complexity |
| Real-time streaming dashboard | Niche use case, conflicts with CLI simplicity |
| Multi-cloud support | Hostodo-only product, would dilute focus |
| Volume/snapshot management | Defer to v2+ based on demand |
| Billing/invoice commands | Web UI is better experience for financial data |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| SEC-01 | TBD | Pending |
| SEC-02 | TBD | Pending |
| SEC-03 | TBD | Pending |
| SEC-04 | TBD | Pending |
| SEC-05 | TBD | Pending |
| SEC-06 | TBD | Pending |
| HOST-01 | TBD | Pending |
| HOST-02 | TBD | Pending |
| HOST-03 | TBD | Pending |
| HOST-04 | TBD | Pending |
| HOST-05 | TBD | Pending |
| HOST-06 | TBD | Pending |
| HOST-07 | TBD | Pending |
| HOST-08 | TBD | Pending |
| CMD-01 | TBD | Pending |
| CMD-02 | TBD | Pending |
| CMD-03 | TBD | Pending |
| CMD-04 | TBD | Pending |
| CMD-05 | TBD | Pending |
| CMD-06 | TBD | Pending |
| DEPLOY-01 | TBD | Pending |
| DEPLOY-02 | TBD | Pending |
| DEPLOY-03 | TBD | Pending |
| DEPLOY-04 | TBD | Pending |
| DEPLOY-05 | TBD | Pending |
| DEPLOY-06 | TBD | Pending |
| DEPLOY-07 | TBD | Pending |
| DEPLOY-08 | TBD | Pending |

**Coverage:**
- v1 requirements: 28 total
- Mapped to phases: 0 (awaiting roadmap)
- Unmapped: 28 ⚠️

---
*Requirements defined: 2026-02-15*
*Last updated: 2026-02-15 after initial definition*
