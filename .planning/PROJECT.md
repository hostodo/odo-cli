# Hostodo CLI v2.0 - Post-OAuth Improvements

## What This Is

A set of security hardening and UX improvements to the Hostodo CLI following the OAuth device flow implementation (PR #189). This iteration addresses code review feedback from Nate and adds user-requested CLI enhancements including hostname-based operations, streamlined commands, and instance deployment capabilities.

## Core Value

CLI users can securely authenticate and manage their VPS instances using intuitive, production-ready commands with hostnames instead of remembering numeric IDs.

## Requirements

### Validated

- ✓ OAuth 2.0 Device Authorization Grant (RFC 8628) - PR #189
- ✓ CLI session management (list, revoke) - PR #189
- ✓ Bearer token authentication - PR #189
- ✓ Instance list/get/start/stop/reboot operations - existing

### Active

- [ ] Device codes use 12-character alphanumeric format (A3K9-M7P2-X4Q8) for brute-force protection
- [ ] CLI sessions track `last_used` datetime for audit trail
- [ ] Session revocation uses soft-delete (`revoked_at` field) for audit compliance
- [ ] Instance commands accept hostname arguments (e.g., `hostodo get myserver`)
- [ ] Hostname display in list and get commands
- [ ] Default scope to instances (e.g., `hostodo list` instead of `hostodo instances list`)
- [ ] Instance deployment via CLI with interactive and flag-based modes
- [ ] Config loading refactored to eliminate duplication

### Out of Scope

- Multiple resource types (volumes, snapshots) - defer to future CLI versions
- SSH integration (direct terminal access) - separate feature
- Bulk operations (stop all, deploy multiple) - v2.1+
- CLI plugins/extensions - not needed yet

## Context

### Current State
- Backend: Django REST API with OAuth device flow complete (28 requirements, 5 phases shipped)
- CLI: Go application using Cobra framework, OAuth login/logout working
- Authentication: Dual auth (CLI Bearer tokens + Web JWT cookies)
- User feedback: Need hostname support (remembering IDs is painful)

### Code Review Findings (Nate)
**Security:**
- 8-digit device codes vulnerable to brute force (10^8 combinations exhaustible in hours)
- No audit trail for token usage (can't track suspicious activity)
- Hard-deleted sessions lose audit history

**UX:**
- Command structure verbose (`hostodo instances list`)
- No hostname support (users work with numeric IDs)

**Dev:**
- Config loading logic scattered across multiple files

## Constraints

- **Backend compatibility**: Must work with existing odopanel API (no breaking backend changes for security fixes)
- **Go version**: 1.21+ (current project version)
- **Cobra framework**: Keep using Cobra for CLI structure
- **Backward compatibility**: Existing `hostodo instances list` commands must still work
- **Migration path**: Users with existing 8-digit codes must be handled gracefully

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| 12-char alphanumeric device codes | Balance security (36^12 combinations) with UX (dash-separated, readable) | — Pending |
| Soft-delete sessions | Audit compliance requires retention of revocation history | — Pending |
| Default instance scope | Instances are 90%+ of CLI usage, reduce cognitive load | — Pending |
| Interactive + flags for deploy | Interactive for humans, flags for automation/scripts | — Pending |
| Hostname as primary identifier | Users think in hostnames, not IDs (matches web UI) | — Pending |

---
*Last updated: 2026-02-15 after initialization*
