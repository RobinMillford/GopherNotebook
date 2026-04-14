<div align="center">

# 🐀 GopherNotebook

**Source-grounded RAG workspaces — 100% local embeddings, BYOK or fully offline generation.**

[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://go.dev/)
[![Next.js](https://img.shields.io/badge/Next.js-16.2-black?style=flat&logo=next.js)](https://nextjs.org/)
[![Weaviate](https://img.shields.io/badge/Weaviate-1.28-green?style=flat)](https://weaviate.io/)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](CONTRIBUTING.md)

</div>

---

GopherNotebook is an open-source document intelligence platform. Upload PDFs, Word docs, spreadsheets, and web pages into isolated workspaces called *Notebooks*, then chat with them using any LLM you choose — cloud or fully local. Your files never leave your machine during ingestion; embeddings, chunking, and reranking all run locally via Weaviate and Qwen3 GGUF models.

---

## Table of Contents

- [Features](#-features)
- [Architecture](#-architecture)
- [Tech Stack](#-tech-stack)
- [Quick Start](#-quick-start)
- [Configuration](#-configuration)
- [Usage Guide](#-usage-guide)
- [Supported LLM Providers](#-supported-llm-providers)
- [Supported File Types](#-supported-file-types)
- [Security & Privacy](#-security--privacy)
- [Roadmap](#-roadmap)
- [Project Structure](#-project-structure)
- [Contributing](#-contributing)
- [License](#-license)

---

## ✨ Features

### RAG Core
| Feature | Description |
|---|---|
| **Local embeddings** | Files are chunked and vectorized entirely on your machine via Qwen3 (GGUF Q4_K_M) and LocalAI — no data leaves your network |
| **Hybrid search** | Combines dense vector similarity (HNSW cosine) with BM25 keyword matching (α = 0.5) for best-of-both retrieval |
| **Cross-encoder reranking** | A second Qwen3 model re-scores the top candidates using full pairwise attention before they reach the LLM |
| **HyDE retrieval** | Hypothetical Document Embedding — generates a fake answer, embeds it, and uses *that* as the query vector for better recall on abstract questions |
| **Semantic deduplication** | Skips near-duplicate chunks at ingest time (cosine distance < 0.03 threshold, configurable) to keep the index clean |
| **Source filtering** | Pin retrieval scope to specific uploaded files per chat turn |
| **Adjustable retrieval params** | Per-query control over retrieval limit (1–50), reranker top-N, and LLM temperature via the Settings panel |

### Notebook Management
| Feature | Description |
|---|---|
| **Isolated workspaces** | Every notebook is a self-contained project with its own documents, chat history, and system prompt |
| **Notebook tags** | Label notebooks with filterable tags; filter the dashboard by any tag with one click |
| **Custom system prompts** | Override the default assistant persona per notebook |
| **Multi-notebook search** | Search across all notebooks from the dashboard — pure retrieval, no API key needed |

### Ingestion
| Feature | Description |
|---|---|
| **Multi-file upload** | Drag-and-drop or browse; up to 50 MB per file, processed concurrently via a Go worker pool |
| **URL ingestion** | Paste any `http/https` URL; the backend fetches, strips HTML, and ingests it as a source |
| **Re-ingest** | Re-process any previously uploaded file from disk without re-uploading (after config changes or model swaps) |
| **Format support** | PDF, DOCX, XLSX, PPTX, TXT, HTML — parsed and semantically chunked (~800 tokens, 800-char overlap) |

### Chat & UX
| Feature | Description |
|---|---|
| **Streaming responses** | Tokens stream token-by-token via Server-Sent Events — no waiting for the full reply |
| **Granular citations** | Every assistant response links claims to the exact source file and page number |
| **Message edit & regenerate** | Hover any user message and click the pencil icon to truncate history and re-ask from that point |
| **Export chat** | Download the full conversation as a Markdown file |
| **Clear chat history** | Wipe the conversation without deleting your documents |

### LLM Flexibility
| Feature | Description |
|---|---|
| **7 providers** | OpenAI, Anthropic, Google Gemini, Groq, OpenRouter, Ollama, LM Studio |
| **BYOK** | API keys live exclusively in browser `localStorage` — the backend never sees them |
| **Live model discovery** | Queries the provider API on key entry to enumerate real available models — no stale hardcoded lists |
| **Model search** | Fuzzy-search through 300+ OpenRouter models instantly |
| **Ollama auto-start** | `start.sh` detects a local Ollama installation and starts the server automatically |

---

## 🏗 Architecture

GopherNotebook uses a three-stage RAG pipeline: **Ingest → Retrieve → Generate**.

```
┌──────────────────────────────────────────────────────────────────┐
│                         INGEST PIPELINE                          │
│                                                                  │
│  File/URL ──► Go Worker Pool ──► Semantic Chunker (~800 tok)     │
│                                        │                         │
│                                  LocalAI (Qwen3 embed)           │
│                                        │                         │
│                      [Semantic dedup check via nearVector query]  │
│                                        │                         │
│                                  Weaviate (HNSW + BM25 index)    │
└──────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────┐
│                        RETRIEVE PIPELINE                         │
│                                                                  │
│  User query ──► [optional HyDE: generate fake answer, embed it]  │
│              ──► Hybrid search (vector α=0.5 + BM25)             │
│              ──► Top 20 candidates                               │
│              ──► Cross-encoder reranker (Qwen3-Reranker-0.6B)    │
│              ──► Top N ranked chunks                             │
└──────────────────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────────────────┐
│                        GENERATE PIPELINE                         │
│                                                                  │
│  Ranked chunks + system prompt + history ──► langchaingo         │
│  ──► OpenAI / Anthropic / Gemini / Groq / OpenRouter             │
│       / Ollama / LM Studio                                       │
│  ──► Streaming SSE ──► Next.js frontend                          │
└──────────────────────────────────────────────────────────────────┘
```

### Service Map

| Service | Port | Technology | Purpose |
|---|---|---|---|
| **Frontend** | 3000 | Next.js 16 + React 19 | Chat UI, dashboard, settings |
| **Backend** | 8090 | Go + Gin | REST API, SSE streaming, RAG orchestration |
| **Weaviate** | 8080 | Weaviate 1.28 | Vector DB — HNSW + BM25 |
| **LocalAI** | 8081 | LocalAI (llama.cpp) | Qwen3 embedding model |
| **Reranker** | 8082 | llama.cpp server | Qwen3-Reranker-0.6B cross-encoder |

---

## 🛠 Tech Stack

**Backend**
- [Go 1.22+](https://go.dev/) — concurrent worker pool, streaming handlers
- [Gin](https://github.com/gin-gonic/gin) — HTTP router
- [langchaingo](https://github.com/tmc/langchaingo) — LLM provider abstraction
- [tabula](https://github.com/tsawler/tabula) — document parsing (PDF, DOCX, XLSX, PPTX)
- [Weaviate Go client](https://github.com/weaviate/weaviate-go-client) — vector DB operations
- [google/uuid](https://github.com/google/uuid) — notebook/message IDs

**Frontend**
- [Next.js 16.2](https://nextjs.org/) + [React 19](https://react.dev/)
- [Tailwind CSS v4](https://tailwindcss.com/) + [shadcn/ui](https://ui.shadcn.com/) (base-nova style)
- [Framer Motion](https://www.framer.com/motion/) — animations
- [date-fns](https://date-fns.org/), [react-markdown](https://github.com/remarkjs/react-markdown), [remark-gfm](https://github.com/remarkjs/remark-gfm)

**Infrastructure**
- [Weaviate](https://weaviate.io/) — vector database with HNSW + inverted index
- [LocalAI](https://localai.io/) — self-hosted inference for the embedding model
- [llama.cpp server](https://github.com/ggerganov/llama.cpp) — self-hosted cross-encoder reranker
- [Docker Compose](https://docs.docker.com/compose/) — service orchestration

**Models (GGUF, auto-downloaded on first run)**
- `qwen3-embedding-0.6b-q4_k_m.gguf` — 1024-dim text embeddings
- `Qwen3-Reranker-0.6B.Q4_K_M.gguf` — cross-encoder reranker

---

## 🚀 Quick Start

### Prerequisites

| Tool | Minimum Version | Install |
|---|---|---|
| Docker + Docker Compose | Latest | [docs.docker.com](https://docs.docker.com/get-docker/) |
| Go | 1.22+ | [go.dev/dl](https://go.dev/dl/) |
| Node.js / npm | 20+ | [nodejs.org](https://nodejs.org/) |

> **Ollama** is optional — install from [ollama.com](https://ollama.com) for fully local, offline generation.

### Install & Run

```bash
# 1. Clone
git clone https://github.com/RobinMillford/GopherNotebook.git
cd GopherNotebook

# 2. Launch (downloads ~1.5 GB of models on first run)
chmod +x start.sh
./start.sh
```

The script handles everything:
1. Checks and auto-installs missing dependencies (Go, Node, Docker) where possible
2. Downloads Qwen3 embedding + reranker GGUF models if absent
3. Starts Weaviate, LocalAI, and the reranker via Docker Compose and waits for health checks
4. Starts the Go backend on `http://localhost:8090`
5. Starts the Next.js frontend on `http://localhost:3000`
6. Auto-starts Ollama if it is installed

**Press `Ctrl+C`** to gracefully shut down all services.

### Development (when Docker is already running)

```bash
# Backend
cd backend && go run ./cmd/server

# Frontend (separate terminal)
cd frontend && npm run dev

# Docker infra only
docker compose up -d
```

---

## ⚙️ Configuration

All backend settings are driven by environment variables. Defaults work out of the box for local development.

| Variable | Default | Description |
|---|---|---|
| `SERVER_PORT` | `8090` | Go backend HTTP port |
| `LOCALAI_URL` | `http://localhost:8081` | LocalAI endpoint for embeddings |
| `EMBEDDING_MODEL` | `qwen3-embed` | Model name registered in LocalAI |
| `EMBEDDING_DIM` | `1024` | Output dimension of the embedding model |
| `RERANKER_URL` | `http://localhost:8082` | llama.cpp reranker endpoint |
| `RERANKER_MODEL` | `Qwen3-Reranker-0.6B` | Reranker model identifier |
| `RERANKER_TOP_N` | `5` | Default number of chunks after reranking |
| `WEAVIATE_HOST` | `localhost:8080` | Weaviate host:port |
| `WEAVIATE_SCHEME` | `http` | `http` or `https` |
| `NOTEBOOK_DATA_DIR` | `./data/notebooks` | Notebook JSON storage directory |
| `UPLOAD_DIR` | `./data/uploads` | Uploaded files directory |
| `SEMANTIC_DEDUP_THRESHOLD` | `0.03` | Cosine distance dedup threshold (set `0` to disable) |

> API keys are browser-only — stored in `localStorage`, sent per-request via `X-API-Key` header. The backend never stores or logs them.

---

## 📖 Usage Guide

### 1 — Create a Notebook
Click **New Notebook** on the dashboard. Give it a name, an optional description, and tags. Tags let you filter the dashboard grid and keep projects organized.

### 2 — Upload Documents
Open a notebook and drag files onto the upload area, or click to browse. Supported formats: **PDF, DOCX, XLSX, PPTX, TXT, HTML** (up to 50 MB per file). The backend chunks and embeds them locally — a progress bar streams updates in real time.

You can also **ingest a URL** (click the link icon in the sidebar) to fetch and process any public web page.

To **re-process** a file (e.g. after changing the chunk size or embedding model), hover the file in the sidebar and click the refresh icon.

### 3 — Configure Your LLM
Click **Settings** in the left sidebar:

- **Provider** — pick cloud (OpenAI, Anthropic, Gemini, Groq, OpenRouter) or local (Ollama, LM Studio)
- **API Key** — paste your key; available models are fetched live. Keys never leave your browser.
- **Model** — search and select from the live list
- **System Prompt** — override the assistant persona for this notebook
- **HyDE** — toggle Hypothetical Document Embedding for better recall on vague queries
- **Retrieval Limit / Reranker Top N / Temperature** — fine-tune chunk retrieval and response creativity

### 4 — Chat
Ask your question and hit **Send**. Responses stream token-by-token. Citations link each claim back to the exact source file and page.

- **Edit a message** — hover any user turn and click the pencil icon to truncate history and re-ask
- **Export** — download the full conversation as Markdown via the icon at the top of the chat area
- **Filter sources** — check/uncheck files in the sidebar to restrict retrieval scope (shown when 2+ sources exist)

### 5 — Multi-Notebook Search
From the dashboard, use the search bar at the top to find relevant chunks across *all* notebooks at once. No API key required — pure local retrieval. Click any result to navigate to that notebook.

---

## 🤖 Supported LLM Providers

| Provider | Local? | Key Source | Notes |
|---|---|---|---|
| **Ollama** | Yes | None | Auto-started by `start.sh`. Run `ollama pull <model>` to add models. |
| **LM Studio** | Yes | None | Start the Local Server inside LM Studio first. |
| **OpenAI** | No | [platform.openai.com](https://platform.openai.com/api-keys) | GPT-4o, o1, GPT-4 Turbo. Live model fetch. |
| **Anthropic** | No | [console.anthropic.com](https://console.anthropic.com/) | Claude 3.7, 3.5 Sonnet, Haiku, Opus. Live model fetch. |
| **Google Gemini** | No | [aistudio.google.com](https://aistudio.google.com/app/apikey) | Gemini 2.5 Flash, 2.0 Flash, 1.5 Pro. Live model fetch. |
| **Groq** | No | [console.groq.com](https://console.groq.com/keys) | Llama 3.3, Mixtral, Gemma 2. Ultra-fast inference. Live model fetch. |
| **OpenRouter** | No | [openrouter.ai/keys](https://openrouter.ai/keys) | 300+ models. Live model fetch + in-app search. |

---

## 📄 Supported File Types

| Format | Extension | Parser |
|---|---|---|
| PDF | `.pdf` | tabula |
| Word | `.docx` | tabula |
| Excel | `.xlsx` | tabula |
| PowerPoint | `.pptx` | tabula |
| Plain text | `.txt` | native |
| HTML / Web pages | `.html` | native (also via URL ingest) |

---

## 🛡 Security & Privacy

- **Embeddings are local.** Files are chunked and vectorized on your machine. LocalAI and Weaviate have no internet access by default.
- **API keys are browser-only.** Stored in `localStorage`. Sent directly to the LLM provider per-request. The Go backend never receives, stores, or logs them.
- **SSRF prevention.** The URL ingestion endpoint validates `http`/`https` scheme before fetching.
- **Path traversal prevention.** File names are sanitized with `filepath.Base` before any disk access.
- **No telemetry.** GopherNotebook sends no analytics or usage data anywhere.

---

## 🗺 Roadmap

Known gaps and improvement areas. Contributions are welcome on any of these.

### Short-term
- [ ] **GPU acceleration** — expose `--gpu-layers` to LocalAI and the reranker container via env var (currently CPU-only)
- [ ] **Document summarization** — auto-generate a 1-paragraph summary per file after ingestion
- [ ] **Notebook export/import** — archive a notebook (metadata + uploads) as a zip; import on another machine
- [ ] **Conversation branching** — fork chat history at any message to explore multiple response paths
- [ ] **Dark / light theme toggle** — UI is currently dark-only

### Medium-term
- [ ] **Optional auth** — single-user password gate for LAN-exposed deployments
- [ ] **Streaming rerank indicator** — show a "Reranking…" status in the UI while the cross-encoder runs
- [ ] **Weaviate multi-tenancy** — isolate each notebook into a separate Weaviate tenant for stronger data boundaries
- [ ] **Pluggable embedding models** — swap Qwen3-Embed for any GGUF embedding model without rebuilding Docker images
- [ ] **Auto conversation titles** — generate a short title from the first user message
- [ ] **Keyboard shortcuts** — `/` focus search, `Ctrl+Enter` send, `Esc` close dialogs

### Long-term
- [ ] **Multi-modal ingestion** — image OCR and audio transcription via Whisper
- [ ] **Agent mode** — let the LLM decide which notebooks to search and when to stop retrieving
- [ ] **REST API SDK** — typed Python/TypeScript client for programmatic notebook and chat access
- [ ] **Collaborative notebooks** — real-time multi-user chat on a shared notebook
- [ ] **OIDC / SSO** — enterprise-ready auth for team deployments

---

## 🏗 Project Structure

```
GopherNotebook/
├── backend/
│   ├── cmd/server/          # Entry point (main.go)
│   └── internal/
│       ├── api/             # Gin handlers and router
│       ├── config/          # Env-based configuration
│       ├── db/              # Weaviate schema + query helpers
│       ├── generate/        # langchaingo LLM streaming + HyDE
│       ├── ingest/          # Parsing, chunking, embedding, worker pool
│       ├── notebook/        # CRUD + message history (JSON on disk)
│       └── retrieve/        # Hybrid search + reranking
├── frontend/
│   └── src/
│       ├── app/
│       │   ├── dashboard/   # Notebook grid, tag filter, global search
│       │   ├── notebook/    # Chat UI, file upload, settings panel
│       │   └── api/models/  # Route handler for live model enumeration
│       ├── components/ui/   # shadcn/ui components
│       └── lib/api.ts       # Typed API client
├── models/                  # GGUF model files (git-ignored, auto-downloaded)
├── data/                    # Weaviate volumes + notebook JSON (git-ignored)
├── scripts/
│   └── download_models.sh   # Model downloader
├── docker-compose.yml
└── start.sh                 # Full-stack launcher
```

---

## 🤝 Contributing

Contributions are welcome! Please read [CONTRIBUTING.md](CONTRIBUTING.md) first.

```bash
# Backend tests
cd backend && go test ./...

# Backend lint
cd backend && go vet ./...

# Frontend type-check + build
cd frontend && npm run build
```

**Branch naming:** `feature/<name>`, `fix/<name>`, `docs/<name>`  
**Commit style:** [Conventional Commits](https://www.conventionalcommits.org/) — `feat:`, `fix:`, `docs:`, `chore:`, etc.

Please open an issue before starting large PRs to align on approach.

---

## 📄 License

Distributed under the **MIT License**. See [LICENSE](LICENSE) for details.

---

<div align="center">
Made with Go and Next.js &nbsp;·&nbsp;
<a href="https://github.com/RobinMillford/GopherNotebook/issues">Report a bug</a> &nbsp;·&nbsp;
<a href="https://github.com/RobinMillford/GopherNotebook/issues">Request a feature</a>
</div>
