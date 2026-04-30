<div align="center">
  <img src="https://unavatar.io/github/wazuh" alt="Wazuh Logo" width="200" />
  <h1>wazuh-cli</h1>
  <p><b>The AI-agent-first CLI for the Wazuh Server API</b></p>

  <div>
    <a href="https://golang.org"><img src="https://img.shields.io/badge/Go-1.26+-00ADD8?style=for-the-badge&logo=go" alt="Go"></a>
    <a href="https://github.com/ba0f3/wazuh-cli/releases"><img src="https://img.shields.io/github/v/release/ba0f3/wazuh-cli?style=for-the-badge&color=blue" alt="Release"></a>
    <a href="LICENSE"><img src="https://img.shields.io/badge/License-MIT-green?style=for-the-badge" alt="License"></a>
  </div>

  <br />

  <p>
    <code>wazuh-cli</code> is a high-performance, single-binary wrapper for the entire Wazuh Server REST API. 
    Designed specifically for <b>AI agents</b> (Claude Code, Gemini CLI, Cline) and <b>Security Engineers</b> who value speed, structure, and security.
  </p>
</div>

---

## ⚡ Key Features

*   🛡️ **Total Parity** — Complete coverage of all Wazuh API modules: Agents, FIM, SCA, RBAC, and more.
*   🤖 **AI Native** — Built-in [Agent Skill](skill/SKILL.md), deterministic exit codes, and machine-first JSON output.
*   🔒 **Hardened Security** — 0600 file permissions, secure interactive login, and credential masking.
*   📊 **Rich Output** — Toggle between structured JSON for scripts and clean Markdown tables for reports.
*   🚀 **Zero Friction** — Single binary, no heavy dependencies, and a 4-tier configuration system.

---

## 📂 Documentation

Quick links to deep-dive guides:

- 🏗️ [**Architecture**](docs/architecture.md) — Design philosophy and component structure.
- 🔑 [**Authentication**](docs/authentication.md) — Multi-method auth, token caching, and secure password input.
- ⚙️ [**Configuration**](docs/configuration.md) — Priority resolution, environment variables, and `config set` flags.
- 📜 [**Command Reference**](docs/commands.md) — Comprehensive list of supported modules.
- 👥 [**User Management**](docs/user-management.md) — How to create and configure API users.
- 🛠️ [**Implementation**](docs/implementation.md) — Technical details for developers.
- 🤖 [**AGENTS.md**](AGENTS.md) — Guidance for AI coding agents (Gemini CLI, Codex, etc.).

---

## MCP Server (AI Agent Integration)

`wazuh-cli` includes a built-in MCP (Model Context Protocol) server, letting AI agents call Wazuh operations directly as tools.

```bash
# Start the MCP server (stdio transport)
wazuh-cli mcp
```

**Claude Desktop / Cursor config:**
```json
{
  "mcpServers": {
    "wazuh": {
      "command": "wazuh-cli",
      "args": ["mcp"],
      "env": {
        "WAZUH_URL": "https://your-wazuh:55000",
        "WAZUH_USER": "wazuh-wui",
        "WAZUH_PASSWORD": "your-password",
        "WAZUH_INSECURE": "true"
      }
    }
  }
}
```

**Available tools:** `agent_list`, `agent_get`, `agent_summary`, `alert_list`, `alert_get`, `alert_stats`, `vulnerability_list`, `vulnerability_summary`

**Resources:** `wazuh://agent/{id}`, `wazuh://alert/{id}`

**Note:** Alert tools and `wazuh://alert/{id}` resource require indexer configuration (`WAZUH_INDEXER_URL`). They are automatically hidden when indexer is not configured.

**Security:** Set `WAZUH_MCP_READONLY=true` for read-only mode (restricts write tools in future phases). Audit logs (JSON lines) go to stderr; suppress with `--quiet`.

---

## 🛠️ Installation

### ⚡ One-liner (Linux & macOS)
```bash
curl -fsSL https://raw.githubusercontent.com/ba0f3/wazuh-cli/main/install.sh | sh
```
Detects your OS and architecture, downloads the correct release binary from GitHub, verifies its checksum, and installs to `/usr/local/bin` (or `~/bin` if not writable). Pass optional flags after `sh -s --`:
```bash
# install a specific version
curl -fsSL https://raw.githubusercontent.com/ba0f3/wazuh-cli/main/install.sh | sh -s -- --version v1.2.3

# install to a custom directory
curl -fsSL https://raw.githubusercontent.com/ba0f3/wazuh-cli/main/install.sh | sh -s -- --dir ~/.local/bin
```

### 🍺 Homebrew (macOS & Linux)
```bash
brew install ba0f3/tap/wazuh-cli
```

### 🚀 Go Install
```bash
go install github.com/ba0f3/wazuh-cli@latest
```

### 📦 Alternative: Pre-built Binaries
Download the latest binary for your platform from the [GitHub Releases](https://github.com/ba0f3/wazuh-cli/releases) page.

### 🏗️ From Source
```bash
git clone https://github.com/ba0f3/wazuh-cli
cd wazuh-cli
make build
sudo mv bin/wazuh-cli /usr/local/bin/
```

---

## 🚦 Quick Start

### 1. Secure Login
Use the interactive login to cache your JWT token without leaking passwords to your shell history:
```bash
wazuh-cli auth login --url https://wazuh:55000 --user admin --insecure
```

### 2. Verify Connectivity
```bash
wazuh-cli manager info
```

### 3. Practical Examples
```bash
# List active agents in Markdown format
wazuh-cli agent list --status active --output markdown

# Find critical vulnerabilities for a specific agent
wazuh-cli vulnerability list 001 --severity critical

# Check cluster and manager status
wazuh-cli cluster health
wazuh-cli manager status
```

---

## ⚙️ Configuration Priority

Settings are merged in the following order (highest wins):

1.  **Flags**: `--url`, `--user`, `--password`, etc.
2.  **Env Vars**: `WAZUH_URL`, `WAZUH_USER`, `WAZUH_PASSWORD`, `WAZUH_TOKEN`, `WAZUH_INDEXER_URL`, etc.
3.  **Local**: `.env` file in the current working directory.
4.  **Global**: `~/.config/wazuh/config.json`.

> [!NOTE]
> **Alerts & OpenSearch**: To query alerts using `wazuh-cli alert`, you must configure `indexer_url` (e.g. `wazuh-cli config set indexer_url https://indexer:9200`). If `indexer_user` and `indexer_password` are not explicitly set, the CLI will automatically fall back to using the Wazuh Manager `user` and `password`.

> [!IMPORTANT]
> Both the config file and the token cache (`~/.config/wazuh/token`) must have **0600 permissions**. The CLI will refuse to load them if they are too open.

---

## 🤖 AI Agent Integration

`wazuh-cli` is optimized to be used as a tool by LLM-based agents. 

### Claude Code Setup
1. Copy the skill definition: `cp skill/SKILL.md ~/.claude/skills/wazuh-cli.md`
2. Or simply reference it in your project's `CLAUDE.md`.

The skill file provides the agent with a compressed reference of all commands, investigation patterns, and error recovery strategies.

---

## 🛡️ Security Policy

Please refer to [SECURITY.md](SECURITY.md) for supported versions and instructions on how to report a vulnerability.

---

## 📜 License

Distributed under the **MIT License**. See `LICENSE` for more information.

<div align="center">
  <sub>Made with ❤️ in Vietnam 🇻🇳</sub>
</div>
