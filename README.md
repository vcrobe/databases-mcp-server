# Databases MCP Server

A Model Context Protocol (MCP) server that gives AI agents safe, structured access to MySQL and PostgreSQL for schema inspection, read queries, and controlled write operations.

Detailed setup guide: [INSTALLATION.md](INSTALLATION.md)

## Why This Project

This server is designed for Copilot/MCP workflows where an agent needs database access with explicit tool boundaries.

- Supports multiple configured database servers (MySQL and PostgreSQL)
- Exposes typed MCP tools with clear purpose and input contracts
- Requires human-readable intent (`explanation`) for query/write operations
- Blocks dangerous destructive operations (`DROP TABLE`, `DROP DATABASE`)
- Provides HTTP health endpoint when running in HTTP/SSE mode

## Features

Registered MCP tools:

1. `list_available_servers`
2. `list_all_databases`
3. `inspect_single_table_schema`
4. `execute_select_query`
5. `execute_write_statement`

## Quick Start

### 1. Build

```bash
go build -o databases-mcp-server
```

### 2. Configure

Create a `.env` file with at least one database server definition.

A full example is available in [INSTALLATION.md](INSTALLATION.md#example-env-file-mysql--postgres).

### 3. Run (default stdio)

```bash
./databases-mcp-server
```

## MCP Client Integration (VS Code)

Example MCP client configuration:

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

## Typical Tool Workflow

1. Call `list_available_servers` to discover configured server names.
2. Call `list_all_databases` with `server_name`.
3. Call `inspect_single_table_schema` with `server_name`, `database_name` (or schema for PostgreSQL), and `table_name`.
4. Call `execute_select_query` for reads (requires `server_name`, `explanation`, `sql`).
5. Call `execute_write_statement` for writes (requires `server_name`, `explanation`, `sql`).

## Safety Model

- `execute_select_query` is intended for read-only `SELECT` statements.
- `execute_write_statement` rejects `DROP TABLE` and `DROP DATABASE`.
- `inspect_single_table_schema` validates table/database identifiers.
- Query and write tools require non-empty `explanation` for auditability.

## Requirements

- Go 1.25.0+
- Access to at least one MySQL or PostgreSQL instance

## Documentation

- Full installation and configuration: [INSTALLATION.md](INSTALLATION.md)
- License: [LICENSE](LICENSE)

## Contributing

Issues and pull requests are welcome at:

https://github.com/vcrobe/databases-mcp-server

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE).
