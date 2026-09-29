# AGENTS.md

## Project Purpose

This repository contains a Go-based MCP server for the Lexware Office public API. The project goal is to provide a maintainable, extensible bridge between MCP clients and Lexware resources such as contacts, articles, invoices, vouchers, files, and related accounting workflows.

Primary external reference:

- Lexware API documentation: https://developers.lexware.io/docs/

## Mandatory Repository Language

All repository-facing content must be written in English.

This rule applies to:

- source code comments
- commit messages
- pull request titles and descriptions
- README content
- AGENTS/CLAUDE guidance files
- issue text added by agents
- generated docs and examples
- test names when readable prose is involved

User prompts may be in German or another language, but the repository output must remain English unless the user explicitly requests a non-English artifact for product reasons.

## Working Agreement For Agents

- Preserve the existing Go codebase and extend it incrementally.
- Prefer small, composable changes over broad rewrites.
- Keep the server usable as an MVP even while dedicated endpoint wrappers are still incomplete.
- Favor typed MCP tools where the schema is stable and valuable.
- Do not add generic raw passthrough MCP tools. This repository should expose typed requests and typed responses only.
- Respect Lexware API limits and keep rate limiting or retry behavior explicit in code.
- Prefer official Lexware API behavior over assumptions; verify unclear details against the official documentation.

## Documentation Requirements

Before every commit, ensure both of these files are up to date with the current implementation:

- `README.md`
- `AGENTS.md`

If the code changes behavior, setup, scope, supported tools, constraints, or conventions, update the documentation in the same change before committing.

## Project Layout

The repository follows the [golang-standards/project-layout](https://github.com/golang-standards/project-layout) convention:

- `cmd/lexware-office-mcp/main.go`: application entrypoint
- `internal/lexware/config.go`: environment-based configuration loading
- `internal/lexware/client.go`: authenticated Lexware HTTP client built on resty with request pacing (2 requests per second) and 429 retry handling
- `internal/lexware/types.go`: typed request and response models
- `internal/lexware/workflows.go`: typed Lexware operations, including follow-up voucher workflows
- `internal/lexware/validation.go`: local validation and normalization before requests are sent
- `internal/lexware/dates.go`: date normalization to the Lexware timestamp format (Europe/Berlin)
- `internal/server/server.go`: MCP server setup and tool registration
- `build/goreleaser/.goreleaser.yml`: GoReleaser configuration for multi-platform releases
- `build/package/docker/`: Dockerfiles for container image builds
- `example/`: ready-to-use MCP client configuration files for Claude, Cursor, Codex, and Windsurf
- `.github/workflows/release.yml`: GitHub Actions workflow triggered by version tags

## Current MVP Tool Surface

The repository currently exposes these MCP tools:

- `lexware_get_profile`
- `lexware_create_simple_contact`
- `lexware_create_contact`
- `lexware_search_contacts`
- `lexware_get_contact`
- `lexware_search_articles`
- `lexware_list_vouchers`
- `lexware_list_payment_conditions`
- `lexware_create_quotation`
- `lexware_get_quotation`
- `lexware_create_order_confirmation`
- `lexware_create_order_confirmation_from_quotation`
- `lexware_get_order_confirmation`
- `lexware_create_invoice`
- `lexware_create_invoice_from_voucher`
- `lexware_get_invoice`
- `lexware_get_down_payment_invoice`

Conventions shared by the voucher tools:

- Dates accept `YYYY-MM-DD` or RFC 3339 and are normalized by `NormalizeDate`.
- Write payloads are validated locally (`prepare()` methods) before any request is sent; keep new validation there.
- Follow-up vouchers send only the leading unit price (gross for `taxType` gross, net otherwise).
- `internal/server/server_test.go` asserts the exact tool list; update it when adding or removing tools.

When adding or removing tools, update `README.md` and this file before committing.

## Preferred Next Steps

- Expose rendered voucher PDFs (`/v1/{resource}/{id}/file`) and file uploads
- Add contact updates and article management
- Add credit notes, delivery notes, and dunnings
- Add tests for config loading
- Consider better error mapping for Lexware validation errors (HTTP 400/406)

## Release Process

Releases are managed by [GoReleaser](https://goreleaser.com/) and triggered by pushing a git tag matching `v*`:

1. Tag a commit: `git tag v0.1.0 && git push origin v0.1.0`
2. GitHub Actions builds multi-platform binaries (linux/darwin/windows, amd64/arm64)
3. Multi-arch Docker images are pushed to `ghcr.io/hermsi1337/lexware-office-mcp`
4. A GitHub Release is created with binaries and checksums

Local testing via Docker (no GoReleaser install needed):

- `make release-check` -- validate config
- `make release-snapshot` -- build without publishing

## Commit Hygiene

- Keep commits focused and readable.
- Use English commit messages.
- Do not commit local workspace artifacts such as `.codex`, `.gocache`, `.gomodcache`, or IDE metadata.
- Keep `.gitignore` current when new local-only files appear.

## CLAUDE.md

`CLAUDE.md` should remain a symlink to `AGENTS.md` so tools and coding agents that look for either file receive the same repository guidance.
