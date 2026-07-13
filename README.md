# dbgenius

**Local-first AI database browser** — a terminal user interface (TUI) for browsing databases with built-in AI assistance via local LLMs.

Explore schemas, run queries, and get AI-powered explanations, suggestions, and optimizations — all from your terminal.

## Features

- **Multi-database support**: PostgreSQL, SQLite, MySQL
- **Schema browser**: Navigate databases, schemas, tables, columns, foreign keys, and indexes with keyboard controls
- **SQL editor**: Multi-line editor with syntax highlighting, command/insert mode, and query history
- **Data viewer**: Paginated, scrollable result grid with null-value handling and clipboard copy
- **AI-powered commands** (via Ollama):
  - `/explain` — Get a plain-English explanation of any SQL query
  - `/suggest` — Generate SQL from natural language descriptions
  - `/optimize` — Get performance optimization suggestions for your queries
- **Connection management**: Configurable connections with automatic retry and reconnection
- **Async queries**: Non-blocking query execution with cancellation and timeout support
- **Graceful degradation**: All TUI features work even without Ollama running

## Quick Start

### Prerequisites

- [Go](https://go.dev/dl/) 1.23 or later
- [Ollama](https://ollama.ai/) (optional, for AI features)

### Install

```bash
# Clone the repository
git clone https://github.com/jefhei/dbgenius.git
cd dbgenius

# Build from source
make build

# Or build directly
go build -o bin/dbgenius ./cmd/dbgenius/
```

### Configure

dbgenius reads configuration from `~/.config/dbgenius/config.toml`. A sample config is created automatically on first run if none exists.

```toml
# ~/.config/dbgenius/config.toml

[database]
# Supported types: postgres, sqlite, mysql
type = "postgres"
host = "localhost"
port = 5432
user = "postgres"
password = ""
dbname = "postgres"
sslmode = "disable"

[ollama]
url = "http://localhost:11434"
model = "llama3.2"
timeout = "30s"
```

### Run

```bash
make run
```

Or directly:

```bash
./bin/dbgenius
```

### Environment Variables

All config options can be overridden with `DBGENIUS_` prefixed environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `DBGENIUS_DATABASE_TYPE` | Database type (`postgres`, `sqlite`, `mysql`) | `postgres` |
| `DBGENIUS_DATABASE_HOST` | Database host | `localhost` |
| `DBGENIUS_DATABASE_PORT` | Database port | `5432` |
| `DBGENIUS_DATABASE_USER` | Database user | `postgres` |
| `DBGENIUS_DATABASE_PASSWORD` | Database password | `` |
| `DBGENIUS_DATABASE_DBNAME` | Database name | `postgres` |
| `DBGENIUS_DATABASE_SSLMODE` | SSL mode | `disable` |
| `DBGENIUS_OLLAMA_URL` | Ollama server URL | `http://localhost:11434` |
| `DBGENIUS_OLLAMA_MODEL` | Ollama model name | `llama3.2` |
| `DBGENIUS_OLLAMA_TIMEOUT` | Ollama request timeout | `30s` |
| `DBGENIUS_CONFIG_DIR` | Config directory | `~/.config/dbgenius` |

### SQLite Quick Start

For SQLite, just specify the database file path:

```toml
[database]
type = "sqlite"
dbname = "/path/to/your/database.db"
```

Or with environment variables:

```bash
DBGENIUS_DATABASE_TYPE=sqlite DBGENIUS_DATABASE_DBNAME=/path/to/db.sqlite ./bin/dbgenius
```

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| `Tab` / `Ctrl+W` | Cycle focus between panels |
| `↑` `↓` | Navigate up/down |
| `←` `→` | Expand/collapse tree, scroll results |
| `Enter` | Expand tree node, select table |
| `PgUp` `PgDn` | Page up/down in results |
| `i` | Enter insert mode (editor) |
| `Esc` | Enter command mode (editor) |
| `Ctrl+Enter` | Execute query |
| `Ctrl+C` | Cancel running query |
| `?` | Toggle help bar |
| `q` / `Ctrl+C` (in help) | Quit / Close help |

### Slash Commands

Type these at the start of a line in the editor:

| Command | Description |
|---------|-------------|
| `/explain` | Explain the current SQL query |
| `/suggest <request>` | Generate SQL from a natural language request |
| `/optimize` | Optimize the current SQL query |
| `/help` | Show available commands |

## AI Integration

dbgenius connects to a local [Ollama](https://ollama.ai/) instance for AI features. Ensure Ollama is running:

```bash
# Start Ollama
ollama serve

# Pull a model (e.g., llama3.2)
ollama pull llama3.2
```

> **Note**: AI features are purely optional. The TUI works fully without Ollama — AI commands simply show a friendly "Ollama not available" message.

## Project Structure

```
dbgenius/
├── cmd/
│   └── dbgenius/          # Main entry point
│       └── main.go
├── internal/
│   ├── ai/                # AI integration (Ollama client, prompts)
│   │   ├── ollama.go          # Ollama HTTP client
│   │   ├── prompt_templates.go # AI prompt templates
│   │   └── schema_context.go   # Schema context builder for LLMs
│   ├── config/            # Configuration loading and validation
│   │   └── config.go
│   ├── db/                # Database abstraction layer
│   │   ├── database.go        # Database interface & types
│   │   ├── factory.go         # Database backend factory
│   │   ├── postgres.go        # PostgreSQL backend (pgx)
│   │   ├── sqlite.go          # SQLite backend (modernc.org/sqlite)
│   │   ├── schema.go          # Schema metadata types & cache
│   │   ├── reconnect.go       # Retry and reconnection logic
│   │   ├── introspect_postgres.go
│   │   └── introspect_sqlite.go
│   └── tui/               # Bubble Tea TUI components
│       ├── model.go           # Root TUI model
│       ├── update.go          # Update delegation
│       ├── view.go            # View rendering
│       ├── tree.go            # Schema tree component
│       ├── editor.go          # SQL editor component
│       ├── dataviewer.go      # Query results viewer
│       ├── commands.go        # Slash command parser
│       ├── sql_highlight.go   # SQL syntax highlighting
│       └── error_handling.go  # Error display and logging
├── bin/                   # Compiled binaries
├── Makefile               # Build automation
├── README.md              # This file
├── PRD.md                 # Product requirements
└── BUILD_PLAN.md          # Task list and progress
```

## Building from Source

### Prerequisites

- Go 1.23 or later

### Build

```bash
# Build for current platform
make build

# Run tests
make test

# Lint
make lint

# Clean build artifacts
make clean
```

### Cross-platform Build

Build for all supported platforms:

```bash
make build-all
```

Or build for a specific platform:

```bash
make build-linux        # Linux amd64
make build-linux-arm    # Linux arm64
make build-darwin       # macOS amd64 (Intel)
make build-darwin-arm   # macOS arm64 (Apple Silicon)
```

## Development

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out -o coverage.html
```

### Adding a New Database Backend

1. Create a new file in `internal/db/` (e.g., `mysql.go`)
2. Implement the `Database` interface (Connect, Close, Ping, GetType, GetSchemas, GetTables, GetTableInfo, ExecuteQuery, GetTableRowCount)
3. Optionally implement `SchemaIntrospector` for schema introspection
4. Register the backend in `internal/db/factory.go`
5. Add test cases following the existing patterns

## Tech Stack

- **TUI Framework**: [Bubble Tea](https://github.com/charmbracelet/bubbletea) (Go)
- **Components**: [Bubbles](https://github.com/charmbracelet/bubbles), [BubbleZone](https://github.com/lrstanley/bubblezone)
- **PostgreSQL**: [pgx/v5](https://github.com/jackc/pgx)
- **SQLite**: [modernc.org/sqlite](https://modernc.org/sqlite) (pure Go, no CGO)
- **Config**: [Viper](https://github.com/spf13/viper)
- **AI**: [Ollama](https://ollama.ai/) API (local LLMs)

## License

MIT

---

*Built with Hermes Agent*
