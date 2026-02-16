# Roadmap: Hostodo CLI v2.0 - Post-OAuth Improvements

## Overview

Transform the Hostodo CLI from a working OAuth prototype into a production-ready VPS management tool through three focused phases. First, harden security with RFC 8628-compliant device codes and audit-trail session management. Second, achieve UX parity with competitors by supporting hostname-based operations and intuitive command structure. Third, differentiate with an interactive deployment wizard that guides users through instance creation while supporting automation. Each phase delivers independently verifiable capabilities that build toward CLI users securely managing VPS instances using intuitive commands with hostnames instead of numeric IDs.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [ ] **Phase 1: Security Hardening & Session Management** - Fix device code entropy, add session audit trails, implement revocation commands
- [ ] **Phase 2: Hostname Support & Command Usability** - Enable hostname-based operations, add command aliases, implement shell completion
- [ ] **Phase 3: Interactive Deployment & Advanced Features** - Build deployment wizard with dual-mode support, add progress indicators and cost estimation

## Phase Details

### Phase 1: Security Hardening & Session Management
**Goal**: CLI authentication meets RFC 8628 security standards with full session audit capabilities
**Depends on**: Nothing (first phase)
**Requirements**: SEC-01, SEC-02, SEC-03, SEC-04, SEC-05, SEC-06
**Success Criteria** (what must be TRUE):
  1. User receives 12-character alphanumeric device codes formatted with dashes (e.g., A3K9-M7P2-X4Q8)
  2. User can view all active CLI sessions with last-used timestamps via `hostodo auth sessions`
  3. User can revoke specific session via `hostodo auth revoke <session-id>` and session remains in database with revoked_at timestamp
  4. Existing users with 8-digit device codes can complete login flow during 30-day migration period
  5. Backend updates last_used timestamp on every authenticated API request
**Plans**: TBD

Plans:
- [ ] 01-01: TBD
- [ ] 01-02: TBD
- [ ] 01-03: TBD

### Phase 2: Hostname Support & Command Usability
**Goal**: Users manage instances by memorable hostnames using streamlined commands with shell completion support
**Depends on**: Phase 1
**Requirements**: HOST-01, HOST-02, HOST-03, HOST-04, HOST-05, HOST-06, HOST-07, HOST-08, CMD-01, CMD-02, CMD-03, CMD-04, CMD-05, CMD-06
**Success Criteria** (what must be TRUE):
  1. User can operate on instances by hostname: `hostodo start myserver`, `hostodo stop myserver`, `hostodo reboot myserver`, `hostodo get myserver`
  2. User can list instances with `hostodo list` (default instance scope) and see hostname column in table output
  3. User with duplicate hostnames sees interactive disambiguation prompt with instance details when running hostname-based commands
  4. User can invoke shell completion (bash/zsh/fish) and get command suggestions
  5. User can skip confirmation prompts with `--yes` flag and output machine-readable YAML with `--output yaml`
  6. Existing commands `hostodo instances list` continue working without deprecation warnings (backward compatibility)
**Plans**: TBD

Plans:
- [ ] 02-01: TBD
- [ ] 02-02: TBD

### Phase 3: Interactive Deployment & Advanced Features
**Goal**: Users deploy instances through guided interactive wizard or automation-friendly flag-based mode with cost visibility
**Depends on**: Phase 2
**Requirements**: DEPLOY-01, DEPLOY-02, DEPLOY-03, DEPLOY-04, DEPLOY-05, DEPLOY-06, DEPLOY-07, DEPLOY-08
**Success Criteria** (what must be TRUE):
  1. User can run `hostodo deploy myserver` and be guided through interactive wizard prompting for plan, region, template, and SSH key
  2. User sees pricing for each plan during selection and cost estimation before confirming deployment
  3. User can deploy non-interactively with flags: `hostodo deploy myserver --plan X --region Y --template Z --ssh-key K`
  4. User sees real-time progress indicators while instance provisions and receives instance details when complete
  5. User deploying in non-TTY environment (CI/CD) sees error requiring flags when interactive prompts would block
**Plans**: TBD

Plans:
- [ ] 03-01: TBD
- [ ] 03-02: TBD

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Security Hardening & Session Management | 0/TBD | Not started | - |
| 2. Hostname Support & Command Usability | 0/TBD | Not started | - |
| 3. Interactive Deployment & Advanced Features | 0/TBD | Not started | - |

---
*Roadmap created: 2026-02-15*
*Last updated: 2026-02-15*
