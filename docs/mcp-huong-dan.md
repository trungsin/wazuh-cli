# Hướng dẫn sử dụng Wazuh MCP Server

## Tổng quan

`wazuh-cli mcp` khởi chạy một MCP (Model Context Protocol) server qua giao thức stdio, cho phép các AI agent (Claude Desktop, Cursor, Cline, VS Code Copilot) gọi trực tiếp các thao tác Wazuh mà không cần copy-paste output từ terminal.

### So sánh CLI vs MCP

| Cách dùng | CLI truyền thống | MCP Server |
|-----------|-----------------|------------|
| Gọi lệnh | `wazuh-cli agent list --status active` | AI agent gọi tool `agent_list` với input `{"status": "active"}` |
| Output | Text/JSON in terminal | JSON trả thẳng cho AI trong conversation |
| Workflow | Copy output → paste vào chat | AI tự gọi, tự đọc, tự phân tích |
| Kết hợp | Pipeline shell phức tạp | AI tự orchestrate nhiều tool calls |

---

## Cài đặt và cấu hình

### Yêu cầu

- `wazuh-cli` đã cài đặt (xem README.md)
- Wazuh Manager API đang chạy (mặc định port 55000)
- (Tùy chọn) Wazuh Indexer/OpenSearch đang chạy (mặc định port 9200) — cần cho alert tools

### Bước 1: Cấu hình Wazuh CLI

```bash
# Cách 1: Interactive login (khuyến nghị — không lưu password trong shell history)
wazuh-cli auth login --url https://wazuh-server:55000 --user wazuh-wui --insecure

# Cách 2: Config file
wazuh-cli config set url https://wazuh-server:55000
wazuh-cli config set user wazuh-wui
wazuh-cli config set password -P   # đọc password từ stdin (an toàn nhất)
wazuh-cli config set insecure true  # nếu dùng self-signed cert

# Cách 3: Environment variables
export WAZUH_URL=https://wazuh-server:55000
export WAZUH_USER=wazuh-wui
export WAZUH_PASSWORD=your-password
export WAZUH_INSECURE=true
```

### Bước 2: (Tùy chọn) Cấu hình Indexer cho Alert tools

```bash
wazuh-cli config set indexer_url https://indexer-node:9200
wazuh-cli config set indexer_user admin
wazuh-cli config set indexer_password -P   # đọc từ stdin

# Hoặc dùng env
export WAZUH_INDEXER_URL=https://indexer-node:9200
export WAZUH_INDEXER_USER=admin
export WAZUH_INDEXER_PASSWORD=secret
```

> **Lưu ý:** Nếu không cấu hình indexer, 3 alert tools (`alert_list`, `alert_get`, `alert_stats`) và resource `wazuh://alert/{id}` sẽ tự động ẩn. Các tool khác vẫn hoạt động bình thường.

### Bước 3: Kiểm tra MCP server

```bash
# Xem help
wazuh-cli mcp --help

# Test khởi chạy (Ctrl+C để thoát)
wazuh-cli mcp
```

---

## Tích hợp với AI Agent

### Claude Desktop

Mở file cấu hình Claude Desktop:
- macOS: `~/Library/Application Support/Claude/claude_desktop_config.json`
- Windows: `%APPDATA%\Claude\claude_desktop_config.json`
- Linux: `~/.config/Claude/claude_desktop_config.json`

Thêm cấu hình:

```json
{
  "mcpServers": {
    "wazuh": {
      "command": "wazuh-cli",
      "args": ["mcp"],
      "env": {
        "WAZUH_URL": "https://wazuh-server:55000",
        "WAZUH_USER": "wazuh-wui",
        "WAZUH_PASSWORD": "your-password",
        "WAZUH_INSECURE": "true",
        "WAZUH_INDEXER_URL": "https://indexer:9200",
        "WAZUH_INDEXER_USER": "admin",
        "WAZUH_INDEXER_PASSWORD": "indexer-password"
      }
    }
  }
}
```

> **Mẹo:** Nếu đã cấu hình qua `wazuh-cli config set` hoặc file `.env`, bạn không cần khai báo env vars ở đây — MCP server tự đọc từ `~/.config/wazuh/config.json`.

### Claude Code (CLI)

Thêm vào `.claude/settings.json` hoặc `.mcp.json` trong project:

```json
{
  "mcpServers": {
    "wazuh": {
      "command": "wazuh-cli",
      "args": ["mcp", "--quiet"]
    }
  }
}
```

### Cursor

Mở Settings → MCP Servers → Add Server:
- Name: `wazuh`
- Command: `wazuh-cli`
- Args: `["mcp"]`

### VS Code (GitHub Copilot)

Thêm vào `.vscode/mcp.json`:

```json
{
  "servers": {
    "wazuh": {
      "command": "wazuh-cli",
      "args": ["mcp"]
    }
  }
}
```

---

## Danh sách Tools

### Agent Tools (Quản lý Agent)

| Tool | Mô tả | Input bắt buộc | Input tùy chọn |
|------|--------|----------------|-----------------|
| `agent_list` | Liệt kê và tìm kiếm các Wazuh agent. Trả về ID, tên, IP, OS, status, group. | — | `status` (active/disconnected/pending/never_connected), `group`, `query` (WQL), `limit` (mặc định 500), `offset` |
| `agent_get` | Lấy thông tin chi tiết của một agent theo ID. | `agent_id` (số, VD: "001") | — |
| `agent_summary` | Tổng quan số lượng agent theo trạng thái (active/disconnected/pending). | — | — |

**Ví dụ prompt cho AI:**
- "Liệt kê tất cả agent đang active"
- "Cho tôi thông tin agent 003"
- "Có bao nhiêu agent đang disconnected?"

### Alert Tools (Truy vấn cảnh báo — cần Indexer)

| Tool | Mô tả | Input bắt buộc | Input tùy chọn |
|------|--------|----------------|-----------------|
| `alert_list` | Truy vấn alerts từ OpenSearch. Lọc theo agent, rule, mức độ, thời gian. | — | `agent_id`, `agent_name`, `rule_id`, `rule_group`, `level` (1-15), `query` (Lucene), `from` (mặc định now-24h), `to` (mặc định now), `limit` (mặc định 50, tối đa 500) |
| `alert_get` | Lấy chi tiết một alert theo document ID. | `doc_id` | `index` (ghi đè index pattern) |
| `alert_stats` | Thống kê alerts theo nhóm: level, agent, hoặc rule. | — | `group_by` (level/agent/rule, mặc định level), `level`, `agent_id`, `from`, `to` |

**Ví dụ prompt cho AI:**
- "Cho tôi 10 alerts nghiêm trọng nhất trong 1 giờ qua"
- "Thống kê alerts theo agent trong 24h qua"
- "Agent 002 có alerts gì hôm nay?"

### Vulnerability Tools (Quét lỗ hổng)

| Tool | Mô tả | Input bắt buộc | Input tùy chọn |
|------|--------|----------------|-----------------|
| `vulnerability_list` | Liệt kê CVE được phát hiện trên một agent. | `agent_id` | `severity` (critical/high/medium/low), `limit` (mặc định 500, tối đa 500), `offset` |
| `vulnerability_summary` | Tổng quan số lượng lỗ hổng theo mức độ nghiêm trọng. | `agent_id` | — |

**Ví dụ prompt cho AI:**
- "Agent 001 có những lỗ hổng critical nào?"
- "Tổng quan vulnerability của agent 003"
- "Tìm tất cả CVE severity high trên agent 005"

---

## Resources (Dữ liệu ngữ cảnh)

Resources cung cấp dữ liệu ngữ cảnh cho AI agent mà không cần gọi tool.

| URI Pattern | Mô tả | Nguồn dữ liệu |
|-------------|--------|----------------|
| `wazuh://agent/{id}` | Profile đầy đủ của agent (tên, IP, OS, status, groups) | Wazuh API |
| `wazuh://alert/{id}` | Chi tiết alert (cần Indexer) | OpenSearch |

---

## Bảo mật

### Chế độ Read-Only

Đặt biến môi trường hoặc config để giới hạn MCP server chỉ đọc:

```bash
export WAZUH_MCP_READONLY=true
# hoặc
wazuh-cli config set mcp_readonly true
```

Khi bật, các tool ghi/xóa (sẽ được thêm trong các phiên bản tương lai) sẽ bị vô hiệu hóa. Phase 1 hiện tại chỉ có tool đọc.

### Audit Logging

Mỗi lần AI agent gọi một tool, MCP server ghi một dòng JSON vào stderr:

```json
{"timestamp":"2026-04-30T12:00:00Z","tool":"agent_list","status":"ok","duration":"45ms"}
{"timestamp":"2026-04-30T12:00:01Z","tool":"alert_list","status":"error","duration":"120ms","error":"indexer error (401): auth failed"}
```

Tắt audit log bằng flag `--quiet` hoặc config `quiet: true`:

```bash
wazuh-cli mcp --quiet
```

### Xác thực

MCP server hỗ trợ tất cả phương thức xác thực của wazuh-cli:

| Phương thức | Cấu hình |
|-------------|----------|
| User/Password | `WAZUH_USER` + `WAZUH_PASSWORD` |
| JWT Token trực tiếp | `WAZUH_TOKEN` |
| mTLS (client cert) | `--client-cert` + `--client-key` |
| Custom CA | `--ca-cert` hoặc `WAZUH_CA_CERT` |

### Kiểm soát bảo mật tại MCP boundary

- **Agent ID**: Chỉ chấp nhận số (regex `^[0-9]{1,5}$`), chặn path traversal
- **Document ID**: Kiểm tra `..`, `/`, `\` để chống path injection
- **GroupBy**: Chỉ cho phép `level`, `agent`, `rule` — không cho truy vấn field tùy ý
- **Limit**: Giới hạn tối đa 500 kết quả, chống DoS qua payload lớn

---

## Ví dụ Workflow thực tế

### 1. Điều tra sự cố (Incident Investigation)

Prompt cho AI:
```
Tôi nghi ngờ agent 003 bị compromise. Hãy:
1. Kiểm tra trạng thái agent
2. Xem 20 alerts gần nhất của agent này
3. Liệt kê các lỗ hổng critical
4. Đưa ra đánh giá ban đầu
```

AI sẽ tự động gọi: `agent_get` → `alert_list` → `vulnerability_list` → phân tích và báo cáo.

### 2. Kiểm tra bảo mật hàng ngày (Daily Security Check)

Prompt cho AI:
```
Cho tôi tổng quan bảo mật:
- Bao nhiêu agent online/offline?
- Top alerts theo severity trong 24h
- Agent nào có nhiều vulnerability critical nhất?
```

AI gọi: `agent_summary` → `alert_stats` → `vulnerability_summary` cho từng agent.

### 3. Tìm CVE cụ thể (CVE Hunt)

Prompt cho AI:
```
Kiểm tra tất cả agent xem có bị ảnh hưởng bởi CVE-2024-XXXX không.
Liệt kê agent list trước, rồi check vulnerability từng agent.
```

AI gọi: `agent_list` → loop `vulnerability_list` per agent → tổng hợp báo cáo.

---

## Mapping tính năng: CLI ↔ MCP Tools

Bảng so sánh giữa lệnh CLI và MCP tool tương ứng:

| Lệnh CLI | MCP Tool | Ghi chú |
|----------|----------|---------|
| `wazuh-cli agent list` | `agent_list` | Hỗ trợ đầy đủ filter: status, group, query, limit, offset |
| `wazuh-cli agent get <ID>` | `agent_get` | Input: `agent_id` |
| `wazuh-cli agent summary` | `agent_summary` | Không cần input |
| `wazuh-cli alert list` | `alert_list` | Cần indexer. Filter: agent_id, rule_id, level, time range, Lucene query |
| `wazuh-cli alert get <ID>` | `alert_get` | Cần indexer. Input: `doc_id` |
| `wazuh-cli alert stats` | `alert_stats` | Cần indexer. Group by: level/agent/rule |
| `wazuh-cli vulnerability list <ID>` | `vulnerability_list` | Input: `agent_id`. Filter: severity |
| `wazuh-cli vulnerability summary <ID>` | `vulnerability_summary` | Input: `agent_id` |
| `wazuh-cli agent delete` | — | Chưa hỗ trợ (Phase 2+) |
| `wazuh-cli agent restart` | — | Chưa hỗ trợ (Phase 2+) |
| `wazuh-cli rule list` | — | Chưa hỗ trợ (Phase 2+) |
| `wazuh-cli decoder list` | — | Chưa hỗ trợ (Phase 2+) |
| `wazuh-cli syscheck list` | — | Chưa hỗ trợ (Phase 2+) |
| `wazuh-cli sca list` | — | Chưa hỗ trợ (Phase 2+) |
| `wazuh-cli cluster status` | — | Chưa hỗ trợ (Phase 3+) |
| `wazuh-cli security user list` | — | Chưa hỗ trợ (Phase 3+) |
| `wazuh-cli active-response run` | — | Chưa hỗ trợ (Phase 3+) |
| `wazuh-cli mitre list` | — | Chưa hỗ trợ (Phase 2+) |

### Roadmap mở rộng

- **Phase 2**: Detection tools (rule, decoder, MITRE), FIM (syscheck, rootcheck, SCA), syscollector
- **Phase 3**: RBAC tools (user/role/policy), ops tools (cluster, manager, active-response)
- **Phase 4**: Security workflow prompts (alert-triage, incident-response, compliance-audit)
- **Phase 5**: Tool scoping (allowlist/denylist), HTTP transport

---

## Xử lý sự cố

### MCP server không khởi động

```bash
# Kiểm tra config
wazuh-cli config list

# Test kết nối API
wazuh-cli manager info

# Chạy với debug
wazuh-cli mcp --debug
```

### Alert tools không hiện

Kiểm tra indexer đã cấu hình chưa:

```bash
wazuh-cli config get indexer_url
# Nếu trống, cần cấu hình:
wazuh-cli config set indexer_url https://indexer:9200
```

### Lỗi xác thực

```bash
# Re-login
wazuh-cli auth login

# Hoặc dùng token trực tiếp
export WAZUH_TOKEN=your-jwt-token
wazuh-cli mcp
```

### Claude Desktop không thấy Wazuh tools

1. Kiểm tra path `wazuh-cli` có trong `$PATH`
2. Restart Claude Desktop sau khi thay đổi config
3. Kiểm tra log lỗi trong Claude Desktop developer console
