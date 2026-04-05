# Installation and Configuration

This guide covers local build, runtime modes, environment variables, and MCP integration.

Back to overview: [README.md](README.md)

## Prerequisites

- Go 1.25.0 or newer
- Reachable MySQL and/or PostgreSQL server
- Valid database credentials

## Clone and Build

```bash
git clone https://github.com/vcrobe/databases-mcp-server
cd databases-mcp-server
go build -o databases-mcp-server
```

## Runtime Modes

The server supports these transports via `MCP_TRANSPORT`:

- `stdio` (default when unset)
- `streamable-http` (alias: `http`)
- `sse` (legacy)

### Run in stdio mode (default)

```bash
./databases-mcp-server
```

### Run in HTTP mode

```bash
MCP_TRANSPORT=http MCP_HTTP_ADDR=:8080 ./databases-mcp-server
```

### Run in SSE mode

```bash
MCP_TRANSPORT=sse MCP_HTTP_ADDR=:8080 ./databases-mcp-server
```

### Health check mode

When the server runs in HTTP or SSE mode, a health endpoint is available at `GET /health`.

You can also probe using the binary:

```bash
./databases-mcp-server -healthcheck
```

Exit code `0` means healthy; `1` means unhealthy or unreachable.

## Environment Variables

Database servers are configured in indexed blocks (`_0`, `_1`, `_2`, ...).

Required keys per server index:

- `DATABASE_SERVER_NAME_{i}`
- `DATABASE_SERVER_ENGINE_{i}`: `mysql` or `postgres`
- `DATABASE_HOST_{i}`
- `DATABASE_PORT_{i}`
- `DATABASE_USER_{i}`
- `DATABASE_PASSWORD_{i}`

Conditionally required:

- `DATABASE_NAME_{i}` is required when `DATABASE_SERVER_ENGINE_{i}=postgres`

Global runtime keys:

- `MCP_TRANSPORT`: `stdio`, `streamable-http`, `http`, or `sse`
- `MCP_HTTP_ADDR`: bind address for HTTP/SSE mode (example: `:8080`)

## Example .env File (MySQL + Postgres)

```dotenv
# Transport (optional; defaults to stdio)
MCP_TRANSPORT=stdio

# HTTP/SSE bind address (used when MCP_TRANSPORT is http/streamable-http/sse)
MCP_HTTP_ADDR=:8080

# Server 0: MySQL
DATABASE_SERVER_NAME_0=mysql-local
DATABASE_SERVER_ENGINE_0=mysql
DATABASE_HOST_0=127.0.0.1
DATABASE_PORT_0=3306
DATABASE_USER_0=root
DATABASE_PASSWORD_0=change-me
# DATABASE_NAME_0 is optional for mysql

# Server 1: PostgreSQL
DATABASE_SERVER_NAME_1=postgres-local
DATABASE_SERVER_ENGINE_1=postgres
DATABASE_HOST_1=127.0.0.1
DATABASE_PORT_1=5432
DATABASE_USER_1=postgres
DATABASE_PASSWORD_1=change-me
DATABASE_NAME_1=appdb
```

## MCP Client Setup (VS Code)

Add this to your MCP config:

```json
{
  "servers": {
    "database-server": {
      "command": "${workspaceFolder}/databases-mcp-server",
      "args": [],
      "envFile": "${workspaceFolder}/.env"
    }
  }
}
```

## Tool Inputs and Usage Notes

### list_available_servers

- Input: none
- Output: configured server names

### list_all_databases

- Input: `server_name`
- Output: list of database names

### inspect_single_table_schema

- Input: `server_name`, `database_name`, `table_name`
- `database_name` means MySQL database or PostgreSQL schema
- Table and database/schema names are validated as safe identifiers

### execute_select_query

- Input: `server_name`, `explanation`, `sql`
- Intended for read-only `SELECT` queries
- Include fully-qualified references:
  - MySQL: `database_name.table_name`
  - PostgreSQL: `schema_name.table_name`

### execute_write_statement

- Input: `server_name`, `explanation`, `sql`
- For `INSERT`, `UPDATE`, `DELETE`, `CREATE`, `ALTER`, and similar writes
- `DROP TABLE` and `DROP DATABASE` are explicitly blocked

## Troubleshooting

### "no database servers configured"

Cause: no valid indexed `DATABASE_SERVER_NAME_{i}` entries found.

Fix: define at least one complete server block in `.env`.

### "invalid database configuration"

Cause: missing required keys or unsupported engine.

Fix: confirm engine is exactly `mysql` or `postgres` and required fields are present.

### Server listed but queries fail

Cause: server connection could not be opened or pinged at startup.

Fix:

1. Confirm host, port, user, and password are correct.
2. Confirm database is reachable from this machine.
3. Restart the server after fixing env vars.

### Health endpoint returns degraded/503

Cause: one or more configured pooled connections cannot be pinged.

Fix: inspect server logs (`mcp-audit.log`) and verify database connectivity.

## Security Notes

- Use least-privilege database users.
- Treat `.env` as sensitive; do not commit credentials.
- Review SQL intent before write operations.
- Keep this server on trusted networks or protected runtime environments.

## Verify Build

```bash
go build -o databases-mcp-server
```

If build succeeds, the binary is ready to run with your `.env` configuration.
