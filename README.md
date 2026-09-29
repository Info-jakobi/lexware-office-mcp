# Lexware Office MCP Server

A [Model Context Protocol (MCP)](https://modelcontextprotocol.io/) server for the [Lexware Office API](https://developers.lexware.io/docs/), written in Go. Connect AI assistants like Claude, Cursor, Codex, or Windsurf directly to your Lexware Office account to manage contacts, create invoices, and more.

## Why?

Lexware Office is one of the most popular cloud accounting platforms in Germany. This MCP server lets AI assistants interact with your Lexware Office data through a safe, typed interface -- no manual API calls, no copy-pasting between tools.

**Key features:**

- **Typed tools only** -- every tool has a strict input/output schema, no raw API passthrough
- **Automatic rate limiting** -- requests are paced to the Lexware limit of 2 per second, and throttled requests (HTTP 429) are retried
- **Quotation to invoice workflow** -- create quotations, turn accepted quotations into linked order confirmations, and prepare linked draft invoices without re-typing line items
- **Validation before sending** -- missing quantities, prices, units, performance dates, or addresses are rejected locally with a clear message
- **Stdio transport** -- works out of the box with Claude, Cursor, Codex, Windsurf, and other MCP clients
- **Minimal footprint** -- single binary or Docker image, no external services required
- **Multi-platform** -- pre-built binaries for Linux, macOS, and Windows (amd64 + arm64)

## Available Tools

| Tool | Description |
|------|-------------|
| `lexware_get_profile` | Fetch the Lexware Office profile for the configured API token |
| `lexware_create_simple_contact` | Create a customer contact with a name and optional reference |
| `lexware_create_contact` | Create a private or business customer with billing address, email, and phone numbers |
| `lexware_search_contacts` | Search contacts by name, email, or customer/vendor number |
| `lexware_get_contact` | Fetch a single contact |
| `lexware_search_articles` | Search articles by article number, GTIN, or type |
| `lexware_list_vouchers` | List vouchers by type and status with totals and open amounts |
| `lexware_list_payment_conditions` | List the configured payment conditions |
| `lexware_create_quotation` | Create a quotation, including optional line items |
| `lexware_get_quotation` | Fetch a quotation |
| `lexware_create_order_confirmation` | Create an order confirmation, optionally linked to a quotation |
| `lexware_create_order_confirmation_from_quotation` | Create a linked order confirmation from an accepted quotation |
| `lexware_get_order_confirmation` | Fetch an order confirmation |
| `lexware_create_invoice` | Create an invoice, optionally linked to a quotation or order confirmation |
| `lexware_create_invoice_from_voucher` | Create a linked invoice from a quotation or order confirmation |
| `lexware_get_invoice` | Fetch an invoice |
| `lexware_get_down_payment_invoice` | Fetch a down payment invoice |

### Conventions

- **Dates** accept `YYYY-MM-DD` (interpreted as midnight in Europe/Berlin) or RFC 3339 timestamps and are sent in the Lexware timestamp format. Voucher list date filters use `YYYY-MM-DD`.
- **Quantities** may have decimals, for example `23.45` square meters.
- **Prices**: set `unitPrice.netAmount` for net and all other tax types, or `unitPrice.grossAmount` when `taxType` is `gross`. The currency defaults to `EUR`.
- **Follow-up vouchers** copy address, line items, prices, discounts, tax, and payment conditions from the preceding voucher. Only the leading price (net or gross) is sent so Lexware recalculates the other one. The response includes the source voucher number and totals for cross-checking.
- **Invoices** require `shippingConditions` (date or period of performance). Created invoices stay drafts unless `finalize` or `LEXWARE_FINALIZE_INVOICES` says otherwise.

### Limitations

- Lexware does not accept follow-up vouchers for quotations that contain optional or alternative line items. Resolve these positions in Lexware first, or create the voucher from a full payload without a link.
- Down payment invoices can only be read through the public API.
- Rendered PDF documents are not exposed yet.

## Prerequisites

You need a **Lexware API token**. Generate one at [app.lexware.de/addons/public-api](https://app.lexware.de/addons/public-api).

## Installation

Choose one of the following methods.

### Pre-built binaries

Download the latest release for your platform from [GitHub Releases](https://github.com/Hermsi1337/lexware-office-mcp/releases):

```bash
# Linux (amd64)
curl -sL https://github.com/Hermsi1337/lexware-office-mcp/releases/latest/download/lexware-office-mcp_linux_amd64.tar.gz | tar xz

# macOS (Apple Silicon)
curl -sL https://github.com/Hermsi1337/lexware-office-mcp/releases/latest/download/lexware-office-mcp_darwin_arm64.tar.gz | tar xz

# Windows (amd64) -- download and extract the zip
```

### Docker

No installation needed. The MCP client starts the container automatically (see [Client Setup](#client-setup) below). To verify manually:

```bash
docker run --rm -e LEXWARE_API_TOKEN=your-token ghcr.io/hermsi1337/lexware-office-mcp
```

Multi-arch image supporting `linux/amd64` and `linux/arm64`.

### Build from source

Requires Go 1.26+:

```bash
git clone https://github.com/Hermsi1337/lexware-office-mcp.git
cd lexware-office-mcp
make build
```

## Configuration

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `LEXWARE_API_TOKEN` | Yes | -- | Your private Lexware API token |
| `LEXWARE_BASE_URL` | No | `https://api.lexware.io` | API base URL |
| `LEXWARE_USER_AGENT` | No | `lexware-office-mcp/0.1.0` | User-Agent header sent to Lexware |
| `LEXWARE_FINALIZE_INVOICES` | No | `false` | Automatically finalize invoices on creation |

All variables are passed as environment variables. When using Docker, the MCP client forwards them to the container automatically.

## Client Setup

Ready-to-use configuration files for all major MCP clients are provided in the [`example/`](example/) directory. Copy the relevant file, insert your API token, and adjust the binary path if needed.

> **Binary or Docker?** Each client below shows both options. The binary approach is simpler; the Docker approach requires no installation and keeps your system clean.

---

### Claude Desktop

<details>
<summary><strong>Binary</strong></summary>

Add to your `claude_desktop_config.json` ([full example](example/claude-desktop.json)):

- macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`
- Windows: `%APPDATA%\Claude\claude_desktop_config.json`

```json
{
  "mcpServers": {
    "lexware-office": {
      "command": "/absolute/path/to/lexware-office-mcp",
      "env": {
        "LEXWARE_API_TOKEN": "your-private-api-key"
      }
    }
  }
}
```
</details>

<details>
<summary><strong>Docker</strong></summary>

Add to your `claude_desktop_config.json` ([full example](example/claude-desktop-docker.json)):

```json
{
  "mcpServers": {
    "lexware-office": {
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "-e", "LEXWARE_API_TOKEN",
        "ghcr.io/hermsi1337/lexware-office-mcp:latest"
      ],
      "env": {
        "LEXWARE_API_TOKEN": "your-private-api-key"
      }
    }
  }
}
```
</details>

---

### Claude Code

<details>
<summary><strong>Binary</strong></summary>

```bash
claude mcp add lexware-office \
  -e LEXWARE_API_TOKEN=your-private-api-key \
  -- /absolute/path/to/lexware-office-mcp
```
</details>

<details>
<summary><strong>Docker</strong></summary>

```bash
claude mcp add lexware-office \
  -e LEXWARE_API_TOKEN=your-private-api-key \
  -- docker run -i --rm -e LEXWARE_API_TOKEN ghcr.io/hermsi1337/lexware-office-mcp:latest
```
</details>

---

### Cursor

<details>
<summary><strong>Binary</strong></summary>

Add to `.cursor/mcp.json` in your project or `~/.cursor/mcp.json` globally ([full example](example/cursor.json)):

```json
{
  "mcpServers": {
    "lexware-office": {
      "command": "/absolute/path/to/lexware-office-mcp",
      "env": {
        "LEXWARE_API_TOKEN": "your-private-api-key"
      }
    }
  }
}
```
</details>

<details>
<summary><strong>Docker</strong></summary>

Same location, using Docker ([full example](example/cursor-docker.json)):

```json
{
  "mcpServers": {
    "lexware-office": {
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "-e", "LEXWARE_API_TOKEN",
        "ghcr.io/hermsi1337/lexware-office-mcp:latest"
      ],
      "env": {
        "LEXWARE_API_TOKEN": "your-private-api-key"
      }
    }
  }
}
```
</details>

---

### OpenAI Codex

<details>
<summary><strong>Binary</strong></summary>

Add to `~/.codex/config.toml` or `.codex/config.toml` in your project ([full example](example/codex.toml)):

```toml
[mcp_servers.lexware-office]
command = "/absolute/path/to/lexware-office-mcp"

[mcp_servers.lexware-office.env]
LEXWARE_API_TOKEN = "your-private-api-key"
```
</details>

<details>
<summary><strong>Docker</strong></summary>

Same location, using Docker ([full example](example/codex-docker.toml)):

```toml
[mcp_servers.lexware-office]
command = "docker"
args = ["run", "-i", "--rm", "-e", "LEXWARE_API_TOKEN", "ghcr.io/hermsi1337/lexware-office-mcp:latest"]

[mcp_servers.lexware-office.env]
LEXWARE_API_TOKEN = "your-private-api-key"
```
</details>

---

### Windsurf

<details>
<summary><strong>Binary</strong></summary>

Add to `~/.codeium/windsurf/mcp_config.json` ([full example](example/windsurf.json)):

```json
{
  "mcpServers": {
    "lexware-office": {
      "command": "/absolute/path/to/lexware-office-mcp",
      "env": {
        "LEXWARE_API_TOKEN": "your-private-api-key"
      }
    }
  }
}
```
</details>

<details>
<summary><strong>Docker</strong></summary>

Same location, using Docker ([full example](example/windsurf-docker.json)):

```json
{
  "mcpServers": {
    "lexware-office": {
      "command": "docker",
      "args": [
        "run", "-i", "--rm",
        "-e", "LEXWARE_API_TOKEN",
        "ghcr.io/hermsi1337/lexware-office-mcp:latest"
      ],
      "env": {
        "LEXWARE_API_TOKEN": "your-private-api-key"
      }
    }
  }
}
```
</details>

---

## Tool Examples

Once connected, your AI assistant can call the following tools.

### Fetch your profile

```json
{
  "name": "lexware_get_profile",
  "arguments": {}
}
```

### Create a contact

```json
{
  "name": "lexware_create_simple_contact",
  "arguments": {
    "name": "Max Mustermann",
    "sourceReference": "Order #12345"
  }
}
```

### Create a quotation

```json
{
  "name": "lexware_create_quotation",
  "arguments": {
    "quotation": {
      "voucherDate": "2026-09-29",
      "expirationDate": "2026-10-29",
      "address": {
        "name": "Max Mustermann",
        "street": "Musterstrasse 1",
        "city": "Ingolstadt",
        "zip": "85049",
        "countryCode": "DE"
      },
      "lineItems": [
        { "type": "text", "name": "Bathroom ground floor" },
        {
          "type": "custom",
          "name": "Wall tiling 30x60",
          "quantity": 23.45,
          "unitName": "m²",
          "unitPrice": { "netAmount": 65.5, "taxRatePercentage": 19 }
        },
        {
          "type": "custom",
          "name": "Tiled niche",
          "quantity": 1,
          "unitName": "Stk",
          "unitPrice": { "netAmount": 180, "taxRatePercentage": 19 },
          "optional": true
        }
      ],
      "taxConditions": { "taxType": "net" }
    },
    "finalize": false
  }
}
```

### Turn an accepted quotation into an order confirmation

```json
{
  "name": "lexware_create_order_confirmation_from_quotation",
  "arguments": {
    "quotationId": "7e1b6e4c-0000-0000-0000-000000000000",
    "shippingConditions": {
      "shippingType": "serviceperiod",
      "shippingDate": "2026-10-12",
      "shippingEndDate": "2026-10-23"
    },
    "deliveryTerms": "Execution in calendar weeks 42 and 43"
  }
}
```

### Prepare a draft invoice from an order confirmation

```json
{
  "name": "lexware_create_invoice_from_voucher",
  "arguments": {
    "precedingVoucherType": "orderconfirmation",
    "precedingVoucherId": "0c5f2a91-0000-0000-0000-000000000000",
    "shippingConditions": {
      "shippingType": "serviceperiod",
      "shippingDate": "2026-10-12",
      "shippingEndDate": "2026-10-23"
    },
    "finalize": false
  }
}
```

### List unpaid invoices

```json
{
  "name": "lexware_list_vouchers",
  "arguments": {
    "voucherTypes": ["invoice"],
    "voucherStatuses": ["open", "overdue"]
  }
}
```

### Create an invoice

```json
{
  "name": "lexware_create_invoice",
  "arguments": {
    "invoice": {
      "voucherDate": "2026-04-07",
      "address": {
        "name": "Max Mustermann",
        "street": "Musterstrasse 1",
        "city": "Freiburg",
        "zip": "79098",
        "countryCode": "DE"
      },
      "lineItems": [
        {
          "type": "custom",
          "name": "Consulting",
          "quantity": 1,
          "unitName": "service",
          "unitPrice": {
            "currency": "EUR",
            "grossAmount": 119.0,
            "taxRatePercentage": 19
          }
        }
      ],
      "totalPrice": { "currency": "EUR" },
      "taxConditions": { "taxType": "gross" },
      "paymentConditions": {
        "paymentTermLabel": "Payable immediately",
        "paymentTermDuration": 0
      },
      "shippingConditions": { "shippingType": "none" }
    },
    "finalize": false
  }
}
```

## Project Structure

Follows the [golang-standards/project-layout](https://github.com/golang-standards/project-layout) convention:

```
cmd/lexware-office-mcp/              # Application entrypoint
internal/lexware/                    # API client, config, types, workflows
internal/server/                     # MCP server and tool registration
build/goreleaser/.goreleaser.yml     # GoReleaser configuration
build/package/docker/                # Dockerfiles
example/                            # Ready-to-use MCP client configs
.github/workflows/                   # CI/CD pipelines
```

## Releasing

Releases are automated via [GoReleaser](https://goreleaser.com/) and GitHub Actions. Push a tag to trigger a release:

```bash
git tag v0.1.0
git push origin v0.1.0
```

This builds multi-platform binaries, creates a GitHub Release, and pushes multi-arch Docker images to `ghcr.io/hermsi1337/lexware-office-mcp`.

To test the release process locally (requires Docker):

```bash
make release-check     # Validate GoReleaser config
make release-snapshot  # Build without publishing
```

## Roadmap

- [x] Contact search, lookup, and creation
- [x] Quotation, order confirmation, and invoice workflows with linked follow-up vouchers
- [x] Voucher list and payment condition lookups
- [x] Unit tests for validation, request building, and MCP tool handlers
- [ ] Contact updates
- [ ] Article management tools
- [ ] Rendered voucher PDFs and file uploads
- [ ] Event subscription support

## Links

- [Lexware Office API Documentation](https://developers.lexware.io/docs/)
- [Model Context Protocol Specification](https://modelcontextprotocol.io/)
- [MCP Go SDK](https://github.com/modelcontextprotocol/go-sdk)

## License

See [LICENSE](LICENSE) for details.
