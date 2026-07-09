# Product Requirements Document: dbgenius

**Version:** 1.0
**Status:** Draft
**Category:** AI/ML Application (Local-First Developer Tool)

---

## 1. Executive Summary

**dbgenius** is a terminal user interface (TUI) database client with built-in, fully local AI assistance powered by [Ollama](https://ollama.com). It sits between two extremes on the current tooling spectrum: heavyweight GUI clients (DataGrip, TablePlus, DBeaver) that are resource-hungry and mouse-driven, and bare-bones CLIs (`psql`, `mysql`) that offer zero intelligence.

dbgenius delivers a fast, keyboard-driven TUI for browsing tables and writing SQL, augmented with AI commands (`/explain`, `/suggest`, `/optimize`) that run entirely offline against a local LLM. No data leaves the machine, no API keys, no cloud dependency — a critical differentiator for developers working with sensitive or regulated data.

The MVP (1–2 weeks) targets PostgreSQL and SQLite, ships a table browser and SQL editor, and integrates three AI commands via Ollama.

---

## 2. Problem Statement

Developers interacting with databases today face a false choice:

**GUI tools are bloated.** DBeaver and DataGrip consume hundreds of MB of RAM, launch slowly, and pull developers out of their terminal-centric workflow. They bury power-user features under menus and dialogs.

**CLIs are dumb.** `psql` and `mysql` are fast and scriptable but offer no assistance. When a developer hits a slow query, an unfamiliar schema, or a cryptic error, they must context-switch to a browser, paste (potentially sensitive) SQL and schema into an online LLM, and manually apply the result.

**Cloud AI has a trust problem.** Copilot-style SQL assistants send schema and query data to remote servers. For teams handling PII, financial, or health data, this is a non-starter — often a compliance violation.

**The gap:** There is no local-first, AI-native database browser. Local LLMs (via Ollama) are now capable enough to explain queries, suggest SQL from natural language, and recommend optimizations — entirely offline. No tool combines a fast TUI, multi-database browsing, and on-device AI.

**Why now:** Quantized models (Llama 3.1, Qwen2.5-Coder, CodeGemma) run acceptably on consumer laptops via Ollama. The infrastructure for offline, private AI assistance is finally mature — but no one has built the database tool around it.

---

## 3. Target Audience

| Segment | Description | Key Need |
|---|---|---|
| **Backend/full-stack developers** | Live in the terminal, work with Postgres/SQLite daily | Fast query iteration without leaving the shell |
| **Data engineers / analysts** | Write complex SQL, debug slow queries | Query explanation and optimization hints |
| **Security-conscious teams** | Handle PII, financial, health data | AI assistance that never sends data off-machine |
| **DBA / platform engineers** | Manage many databases, review query performance | Schema exploration + optimization suggestions |
| **SQL learners** | Building fluency | Plain-English explanations of queries and schemas |

**Primary MVP persona:** The terminal-native backend developer who wants AI help but cannot or will not paste production data into ChatGPT.

---

## 4. Feature Requirements

### Must-Have (MVP)

| ID | Feature | Description |
|---|---|---|
| F1 | **DB Connection Manager** | Connect to PostgreSQL and SQLite via connection string or config file |
| F2 | **Table Browser** | Navigable tree/list of schemas → tables → columns; keyboard navigation |
| F3 | **Data Viewer** | Paginated table data display with column types |
| F4 | **SQL Editor** | Multi-line editor with execution (Ctrl+Enter), syntax highlighting |
| F5 | **Result Grid** | Scrollable, paginated query results in the TUI |
| F6 | **Ollama Integration** | Detect/connect to local Ollama; model selection |
| F7 | **`/explain`** | Explain the current/selected SQL query in plain English |
| F8 | **`/suggest`** | Generate SQL from a natural-language prompt using live schema context |
| F9 | **`/optimize`** | Suggest performance improvements for the current query |
| F10 | **Schema Context Injection** | Automatically feed relevant table/column metadata to the LLM |
| F11 | **Error Handling** | Graceful messages when DB or Ollama is unreachable |

### Nice-to-Have (Post-MVP)

| ID | Feature |
|---|---|
| N1 | MySQL, MariaDB, and DuckDB support |
| N2 | `EXPLAIN ANALYZE` integration feeding real query plans into `/optimize` |
| N3 | Query history and saved queries |
| N4 | Export results (CSV, JSON) |
| N5 | Inline AI autocomplete while typing SQL |
| N6 | `/fix` command for broken queries |
| N7 | Vim/Emacs keybinding modes |
| N8 | Multi-connection tabs |
| N9 | Configurable AI prompt templates |
| N10 | Streaming AI responses (token-by-token) |

### Out of Scope (MVP)

- Cloud/remote AI providers
- Data editing/writes via the browser UI (read-focused MVP; raw SQL still allows writes)
- Migration management, ER diagrams, admin tooling

---

## 5. Technical Architecture

**Language:** Go (recommended for MVP velocity — mature TUI + DB ecosystem). Rust is a viable alternative if maximum performance is prioritized.

**Recommended stack (Go):**
- **TUI framework:** [Bubble Tea](https://github.com/charmbracelet/bubbletea) + Lip Gloss + Bubbles
- **DB drivers:** `pgx` (Postgres), `mattn/go-sqlite3` (SQLite)
- **AI backend:** Ollama HTTP API (`http://localhost:11434`)
- **Config:** TOML/YAML file + env vars

### High-Level Component Diagram

```
┌─────────────────────────────────────────────────┐
│                   TUI Layer                        │
│  (Bubble Tea: table browser, editor, result grid) │
└───────────────┬───────────────────┬───────────────┘
                │                   │
        ┌───────▼────────┐  ┌───────▼─────────┐
        │  DB Engine     │  │  AI Engine       │
        │  - conn pool   │  │  - prompt builder│
        │  - query exec  │  │  - Ollama client │
        │  - schema read │  │  - response parse│
        └───────┬────────┘  └───────┬──────────┘
                │                   │
        ┌───────▼────────┐  ┌───────▼──────────┐
        │  Postgres /    │  │  Ollama daemon   │
        │  SQLite        │  │  (local models)  │
        └────────────────┘  └──────────────────┘
```

### Key Architectural Decisions

1. **Schema Context Service** — A dedicated module extracts table names, columns, types, indexes, and foreign keys, then formats a compact context block for LLM prompts. This is the core of AI quality.
2. **Prompt Templates** — Each AI command has a purpose-built system prompt (`/explain`, `/suggest`, `/optimize`).
3. **Streaming** — Consume Ollama's streaming API to render AI output progressively in the TUI (nice-to-have for MVP polish).
4. **Async execution** — DB queries and AI calls run in goroutines/commands to keep the TUI responsive.
5. **Graceful degradation** — App is fully usable as a DB browser even if Ollama is not running.

---

## 6. Milestones & Timeline

**Total MVP: ~10 working days (2 weeks)**

| Milestone | Focus | Duration |
|---|---|---|
| **M1 — Foundation** | Project scaffolding, DB connectivity, config | Days 1–2 |
| **M2 — Core TUI** | Table browser, SQL editor, result grid | Days 3–5 |
| **M3 — AI Integration** | Ollama client, schema context, `/explain` `/suggest` `/optimize` | Days 6–8 |
| **M4 — Polish & Release** | Error handling, docs, packaging, testing | Days 9–10 |

---

## 7. Success Metrics

### Adoption / Usage
- **Time-to-first-query:** < 60 seconds from launch to running a query
- **AI command usage rate:** ≥ 40% of active sessions invoke at least one AI command
- **Startup time:** TUI renders in < 500ms; memory footprint < 100MB (excluding Ollama)

### Quality
- **AI response latency:** `/explain` returns first token in < 3s on a mid-range laptop (7B model)
- **`/suggest` validity:** ≥ 70% of generated queries are syntactically valid and executable on first attempt
- **Crash-free sessions:** ≥ 99%

### North Star
- **Weekly retained developers** who use dbgenius as their primary terminal DB client.

---

## Task List

```json
[
  {
    "milestone": "M1 - Foundation",
    "tasks": [
      {"id":"M1.1","title":"Initialize Go project, module structure, and Makefile","priority":"high","estimate":"0.5d","dependencies":[]},
      {"id":"M1.2","title":"Set up Bubble Tea app skeleton with root model and event loop","priority":"high","estimate":"0.5d","dependencies":["M1.1"]},
      {"id":"M1.3","title":"Implement config loader (TOML/YAML + env) for connections and Ollama settings","priority":"high","estimate":"0.5d","dependencies":["M1.1"]},
      {"id":"M1.4","title":"Build DB connection layer with pgx (Postgres) and go-sqlite3 (SQLite)","priority":"high","estimate":"1d","dependencies":["M1.3"]},
      {"id":"M1.5","title":"Implement schema introspection (schemas, tables, columns, types, FKs, indexes)","priority":"high","estimate":"1d","dependencies":["M1.4"]},
      {"id":"M1.6","title":"Add connection error handling and reconnect logic","priority":"medium","estimate":"0.5d","dependencies":["M1.4"]}
    ]
  },
  {
    "milestone": "M2 - Core TUI",
    "tasks": [
      {"id":"M2.1","title":"Build table browser panel (schema/table tree with keyboard nav)","priority":"high","estimate":"1d","dependencies":["M1.5","M1.2"]},
      {"id":"M2.2","title":"Build paginated data viewer for selected table","priority":"high","estimate":"1d","dependencies":["M2.1"]},
      {"id":"M2.3","title":"Implement multi-line SQL editor component with input handling","priority":"high","estimate":"1d","dependencies":["M1.2"]},
      {"id":"M2.4","title":"Add SQL syntax highlighting in editor","priority":"medium","estimate":"0.5d","dependencies":["M2.3"]},
      {"id":"M2.5","title":"Implement async query execution (Ctrl+Enter) with goroutine + messages","priority":"high","estimate":"0.5d","dependencies":["M2.3","M1.4"]},
      {"id":"M2.6","title":"Build scrollable, paginated result grid","priority":"high","estimate":"1d","dependencies":["M2.5"]},
      {"id":"M2.7","title":"Implement layout, focus switching, and status/help bar","priority":"medium","estimate":"0.5d","dependencies":["M2.1","M2.3","M2.6"]}
    ]
  },
  {
    "milestone": "M3 - AI Integration",
    "tasks": [
      {"id":"M3.1","title":"Build Ollama HTTP client with health check and model listing","priority":"high","estimate":"0.5d","dependencies":["M1.3"]},
      {"id":"M3.2","title":"Implement schema context builder (compact metadata for prompts)","priority":"high","estimate":"1d","dependencies":["M1.5"]},
      {"id":"M3.3","title":"Design prompt templates for /explain, /suggest, /optimize","priority":"high","estimate":"0.5d","dependencies":["M3.1"]},
      {"id":"M3.4","title":"Implement command parser for slash commands in editor","priority":"high","estimate":"0.5d","dependencies":["M2.3"]},
      {"id":"M3.5","title":"Implement /explain command with response rendering panel","priority":"high","estimate":"0.5d","dependencies":["M3.3","M3.4"]},
      {"id":"M3.6","title":"Implement /suggest command with schema context injection","priority":"high","estimate":"1d","dependencies":["M3.2","M3.3","M3.4"]},
      {"id":"M3.7","title":"Implement /optimize command","priority":"high","estimate":"0.5d","dependencies":["M3.2","M3.3","M3.4"]},
      {"id":"M3.8","title":"Add streaming token rendering for AI responses","priority":"medium","estimate":"0.5d","dependencies":["M3.5"]},
      {"id":"M3.9","title":"Handle Ollama-unavailable graceful degradation","priority":"high","estimate":"0.5d","dependencies":["M3.1"]}
    ]
  },
  {
    "milestone": "M4 - Polish & Release",
    "tasks": [
      {"id":"M4.1","title":"End-to-end error handling and user-facing error messages","priority":"high","estimate":"0.5d","dependencies":["M3.9","M2.5"]},
      {"id":"M4.2","title":"Write unit tests for DB layer and schema context builder","priority":"medium","estimate":"1d","dependencies":["M1.5","M3.2"]},
      {"id":"M4.3","title":"Write README, quickstart, and config documentation","priority":"high","estimate":"0.5d","dependencies":["M4.1"]},
      {"id":"M4.4","title":"Cross-platform build and binary packaging (macOS/Linux)","priority":"high","estimate":"0.5d","dependencies":["M4.1"]},
      {"id":"M4.5","title":"Manual QA pass against Postgres and SQLite test databases","priority":"high","estimate":"0.5d","dependencies":["M4.1"]},
      {"id":"M4.6","title":"Measure and tune startup time, memory, and AI latency vs success metrics","priority":"medium","estimate":"0.5d","dependencies":["M4.5"]}
    ]
  }
]
```