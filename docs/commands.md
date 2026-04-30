# Command Reference

`wazuh-cli` provides comprehensive coverage of the Wazuh API. Commands are grouped by resource type.

## Global Flags

These flags apply to all commands:
- `-u, --url`: API URL
- `--user`: Username
- `--password`: Password (use `-` for stdin)
- `-k, --insecure`: Skip TLS verification
- `-o, --output`: Format (`json`, `markdown`, `raw`)
- `--debug`: Verbose logging

## Core Commands

### Infrastructure
- `auth`: Interactive login and token management.
- `init`: Setup wizard for initial configuration.
- `config`: Manage the local `config.json` file.
- `manager`: Server information, logs, and configuration.
- `cluster`: Cluster status and node management.

### Agents
- `agent`: List, get, add, delete, and upgrade agents.
- `group`: Manage agent groups and assignments.

### Security Monitoring
- `syscheck`: File Integrity Monitoring (FIM) scans and results.
- `rootcheck`: Policy and anomaly detection.
- `sca`: Security Configuration Assessment results.
- `vulnerability`: Query vulnerability detection data.
- `syscollector`: Hardware and software inventory.

### Rules & Decoders
- `rule`: List, get, and test rules.
- `decoder`: List, get, and test decoders.
- `cdb-list`: Manage Constant Database (CDB) lists.
- `logtest`: Interactive or file-based log analysis testing.

### Threat Intelligence & Frameworks
- `mitre`: Query MITRE ATT&CK framework mappings.
- `ciscat`: CIS-CAT assessment results.

### AI Agent Integration
- `mcp`: Start an MCP (Model Context Protocol) server over stdio. Exposes Wazuh operations as tools for AI agents (Claude Desktop, Cursor, Cline). Available tools: `agent_list`, `agent_get`, `agent_summary`, `alert_list`, `alert_get`, `alert_stats`, `vulnerability_list`, `vulnerability_summary`. Resources: `wazuh://agent/{id}`, `wazuh://alert/{id}`. Alert tools require indexer configuration (`WAZUH_INDEXER_URL`); they are hidden when not configured. Set `WAZUH_MCP_READONLY=true` for read-only mode. Audit logs go to stderr as JSON lines (suppress with `--quiet`).

## Output Formats

### JSON (Default)
Optimized for AI agents and `jq`.
```bash
wazuh-cli agent list --output json
```

### Markdown
Generates clean tables for reports.
```bash
wazuh-cli vulnerability list 001 --output markdown
```

### Raw
Returns the exact response from the Wazuh API without processing.
```bash
wazuh-cli manager info --output raw
```
