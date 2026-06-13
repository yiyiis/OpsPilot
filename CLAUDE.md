# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**OpsPilot** is an AI-driven intelligent operations assistant (OnCall duty support system) built in Go. It combines RAG knowledge Q&A with Plan-Execute-Replan alert analysis to help ops teams diagnose and resolve production incidents.

## Build & Run Commands

```bash
# Build
go build main.go

# Run the backend (requires HTTPS_PROXY for Gemini API)
HTTPS_PROXY=http://<proxy>:<port> go run main.go

# Run tests — no test suite exists currently (no *_test.go files)

# Lint — no linter configured (no .golangci.yml)

# Run a specific CLI command
go run internal/ai/cmd/<cmd>/main.go
```

### Infrastructure Dependencies

Start Milvus vector database before the backend:

```bash
cd manifest/docker && docker-compose up -d
# Milvus :19530, Attu UI :8000, MinIO :9000, etcd (internal)
```

### Startup Order (Critical)

1. **Milvus** (`docker-compose up`)
2. **Knowledge indexing** — must run BEFORE backend, otherwise the backend auto-creates the collection with wrong schema:
   ```bash
   HTTPS_PROXY=http://<proxy>:<port> go run internal/ai/cmd/knowledge_cmd/main.go
   ```
3. **Log MCP Server**: `go run internal/ai/mcp/log_reader_server/main.go` (SSE on `:3001`)
4. **Backend**: `HTTPS_PROXY=http://<proxy>:<port> go run main.go` (Gin on `:6872`)
5. **Frontend**: `cd OpsPilotFrontend && bash start.sh` (serves on `:8080`)

## Architecture

### HTTP Layer

- **Framework**: Gin (not GoFrame despite README claiming it) — see `main.go`
- **4 endpoints**: `/api/chat` (sync), `/api/chat_stream` (SSE), `/api/upload` (multipart), `/api/ai_ops` (alert analysis)
- **SSE infrastructure**: `internal/logic/sse/` manages per-client streaming with UUID-based session IDs
- **Controllers**: `internal/controller/chat/`

### Three AI Agent Pipelines (CloudWeGo Eino DAG Orchestration)

**(A) RAG Chat Agent** — `internal/ai/agent/chat_pipeline/`
- Parallel DAG: Milvus retrieval path + conversation context path → ChatTemplate → ReActAgent
- ReActAgent: max 25 reasoning-action loops, 5 tool types
- Key files: `orchestration.go` (graph topology), `flow.go` (ReAct agent), `prompt.go`, `lambda_func.go` (data transformers), `retriever.go`, `tools_node.go`

**(B) Plan-Execute-Replan Agent** — `internal/ai/agent/plan_execute_replan/`
- Planner (Think model) → Executor (Quick model + tools) → Replanner (Think model) → loop or final answer
- Uses Eino ADK's prebuilt `planexecute` framework, max 20 iterations
- Key files: `plan_execute_replan.go`, `planner.go`, `executor.go`, `replan.go`

**(C) Knowledge Index Pipeline** — `internal/ai/agent/knowledge_index_pipeline/`
- Linear DAG: FileSource → FileLoader → MarkdownSplitter → MilvusIndexer
- Key files: `orchestration.go`, `loader.go`, `transformer.go`, `indexer.go`

### AI Infrastructure Layer

- **Models** (`internal/ai/models/`): LLM factory — GLM "Think" model for planning/replanning, "Quick" model for chat/execution, both via OpenAI-compatible API
- **Embedder** (`internal/ai/embedder/`): Custom Gemini Embedder (3072-dim float vectors via `google.golang.org/genai`)
- **Retriever** (`internal/ai/retriever/`): Milvus vector search wrapper
- **Indexer** (`internal/ai/indexer/`): Milvus vector write wrapper
- **Tools** (`internal/ai/tools/`): Agent tool implementations — MCP multi-server (log-reader + Zhipu web search + web reader), Prometheus alerts, MySQL CRUD (GORM), time utility, internal docs RAG
- **MCP** (`internal/ai/mcp/log_reader_server/`): Standalone MCP server for local log file querying (SSE on `:3001`)

### Shared Utilities

- `utility/config/`: Viper-based YAML config loading (`manifest/config/config.yaml`)
- `utility/client/`: Milvus client init with auto-provisioning (creates `agent` DB + `biz` collection)
- `utility/mem/`: In-memory sliding-window conversation memory (per-session, max 6 messages)
- `utility/common/`: Shared constants (`MilvusDBName="agent"`, `MilvusCollectionName="biz"`)

### Frontend

`OpsPilotFrontend/` — vanilla JS + SSE, zero build step, Gemini-style chat UI. Served via `start.sh` (Python `http.server` or Node `serve`).

## Key Technical Decisions

- **Eino framework** (not LangChain) for AI Agent orchestration — CloudWeGo's DAG graph-based approach
- **Dual LLM strategy**: "Think" model for complex reasoning (planning/replanning), "Quick" model for fast responses (chat/execution)
- **MCP protocol** for tool integration — 3 MCP servers configured in `config.yaml`
- **Milvus schema**: FloatVector(3072) with COSINE metric (defined in `indexer.go` and `retriever.go`)
- **Known inconsistency**: `utility/client/client.go` defines the vector field as `BinaryVector(65536)` but the actual indexer/retriever use `FloatVector(3072)` — the indexer/retriever are authoritative

## Configuration

All config in `manifest/config/config.yaml` (Viper):
- LLM model configs (GLM Think + Quick via `open.bigmodel.cn`)
- Gemini embedding model config
- File directory path for knowledge docs
- MCP server connection configs (3 servers)
- API keys (set directly in YAML)

## Codebase Conventions

- Code comments are in Chinese — maintain this convention
- Module READMEs exist under each agent pipeline directory with concept explanations
- `docs/ai-agent-architecture-guide.md` provides a 7-concept learning guide with data flow diagrams
- No tests exist yet — adding `*_test.go` files is welcome
- Go module name is simply `OpsPilot` (no domain prefix)
