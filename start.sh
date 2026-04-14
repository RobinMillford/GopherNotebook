#!/bin/bash

# GopherNotebook Quickstart Script
# Installs missing dependencies, downloads models, starts all services.
#
# Usage:
#   ./start.sh              # full stack (Docker + backend + frontend)
#   ./start.sh --no-docker  # skip Docker (infra already running)

set -e

# ─── Colors ────────────────────────────────────────────────────────────────────
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

# ─── Args ──────────────────────────────────────────────────────────────────────
SKIP_DOCKER=false
for arg in "$@"; do
    case $arg in
        --no-docker) SKIP_DOCKER=true ;;
        --help|-h)
            echo "Usage: $0 [--no-docker]"
            echo "  --no-docker   Skip Docker Compose (use when infra is already running)"
            exit 0
            ;;
    esac
done

# ─── Banner ────────────────────────────────────────────────────────────────────
echo ""
echo -e "${BLUE}${BOLD}╔══════════════════════════════════════════╗${NC}"
echo -e "${BLUE}${BOLD}║        🐀  GopherNotebook  🐀           ║${NC}"
echo -e "${BLUE}${BOLD}╚══════════════════════════════════════════╝${NC}"
echo ""

# ─── Detect OS ─────────────────────────────────────────────────────────────────
OS="$(uname -s)"
case "${OS}" in
    Linux*)   MACHINE=Linux ;;
    Darwin*)  MACHINE=Mac ;;
    CYGWIN*|MINGW*|MSYS*) MACHINE=Windows ;;
    *)        MACHINE=Other ;;
esac

# ─── Helper: wait for HTTP endpoint ────────────────────────────────────────────
wait_for() {
    local url="$1"
    local label="$2"
    local max_wait="${3:-120}"
    local elapsed=0
    printf "${YELLOW}  Waiting for ${label}...${NC}"
    while ! curl -sf "$url" > /dev/null 2>&1; do
        sleep 2
        elapsed=$((elapsed + 2))
        printf "."
        if [ "$elapsed" -ge "$max_wait" ]; then
            echo ""
            echo -e "${RED}✗ Timed out waiting for ${label} (${max_wait}s). Check Docker logs.${NC}"
            exit 1
        fi
    done
    echo -e " ${GREEN}ready${NC}"
}

# ─── 1. Prerequisites ──────────────────────────────────────────────────────────
echo -e "${CYAN}[1/6]${NC} ${BOLD}Checking prerequisites...${NC}"

check_or_install_go() {
    if command -v go &> /dev/null; then
        echo -e "  ${GREEN}✓${NC} Go $(go version | awk '{print $3}')"
        return
    fi
    echo -e "  ${YELLOW}Go not found — installing...${NC}"
    if [ "$MACHINE" = "Linux" ]; then
        local ver="1.22.2"
        curl -sLO "https://go.dev/dl/go${ver}.linux-amd64.tar.gz"
        rm -rf "$HOME/.local/go"
        mkdir -p "$HOME/.local"
        tar -C "$HOME/.local" -xzf "go${ver}.linux-amd64.tar.gz"
        rm "go${ver}.linux-amd64.tar.gz"
        export PATH="$PATH:$HOME/.local/go/bin"
        grep -q '\.local/go/bin' "$HOME/.bashrc" 2>/dev/null || \
            echo 'export PATH=$PATH:$HOME/.local/go/bin' >> "$HOME/.bashrc"
        echo -e "  ${GREEN}✓${NC} Go installed to ~/.local/go"
    elif [ "$MACHINE" = "Mac" ]; then
        command -v brew &>/dev/null || { echo -e "${RED}Homebrew required. Install from https://brew.sh${NC}"; exit 1; }
        brew install go
    elif [ "$MACHINE" = "Windows" ]; then
        winget install GoLang.Go
    else
        echo -e "${RED}Please install Go manually: https://go.dev/dl${NC}"; exit 1
    fi
}

check_or_install_node() {
    if command -v npm &> /dev/null; then
        echo -e "  ${GREEN}✓${NC} Node $(node --version) / npm $(npm --version)"
        return
    fi
    echo -e "  ${YELLOW}npm not found — installing Node.js via nvm...${NC}"
    if [ "$MACHINE" = "Linux" ]; then
        curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh | bash
        export NVM_DIR="$HOME/.nvm"
        # shellcheck source=/dev/null
        [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
        nvm install 20 && nvm use 20
        echo -e "  ${GREEN}✓${NC} Node.js installed"
    elif [ "$MACHINE" = "Mac" ]; then
        command -v brew &>/dev/null || { echo -e "${RED}Homebrew required.${NC}"; exit 1; }
        brew install node
    elif [ "$MACHINE" = "Windows" ]; then
        winget install OpenJS.NodeJS
    else
        echo -e "${RED}Please install Node.js manually: https://nodejs.org${NC}"; exit 1
    fi
}

check_or_install_docker() {
    if command -v docker &> /dev/null; then
        echo -e "  ${GREEN}✓${NC} Docker $(docker --version | awk '{print $3}' | tr -d ',')"
        return
    fi
    echo -e "  ${YELLOW}Docker not found — installing...${NC}"
    if [ "$MACHINE" = "Linux" ]; then
        curl -fsSL https://get.docker.com -o get-docker.sh
        sudo sh get-docker.sh && rm get-docker.sh
        sudo usermod -aG docker "$USER"
        echo -e "  ${GREEN}✓${NC} Docker installed. You may need to restart your session for group permissions."
    elif [ "$MACHINE" = "Mac" ]; then
        command -v brew &>/dev/null || { echo -e "${RED}Homebrew required.${NC}"; exit 1; }
        brew install --cask docker
        echo -e "  ${YELLOW}Open Docker Desktop from Applications to finish setup, then re-run this script.${NC}"
        exit 0
    elif [ "$MACHINE" = "Windows" ]; then
        winget install Docker.DockerDesktop
        echo -e "  ${YELLOW}Restart Windows and open Docker Desktop, then re-run this script.${NC}"
        exit 0
    else
        echo -e "${RED}Please install Docker manually: https://docs.docker.com/get-docker${NC}"; exit 1
    fi
}

check_or_install_go
check_or_install_node
if [ "$SKIP_DOCKER" = false ]; then
    check_or_install_docker
fi

# ─── 2. Models ─────────────────────────────────────────────────────────────────
echo ""
echo -e "${CYAN}[2/6]${NC} ${BOLD}Checking local AI models...${NC}"

MODEL_1="./models/qwen3-embedding-0.6b-q4_k_m.gguf"
MODEL_2="./models/Qwen3-Reranker-0.6B.Q4_K_M.gguf"

if [ -f "$MODEL_1" ] && [ -f "$MODEL_2" ]; then
    echo -e "  ${GREEN}✓${NC} Models present"
else
    echo -e "  ${YELLOW}Models missing — downloading (~1.5 GB)...${NC}"
    if [ -f "./scripts/download_models.sh" ]; then
        bash ./scripts/download_models.sh
    else
        echo -e "  ${RED}scripts/download_models.sh not found. Please download models manually.${NC}"
        exit 1
    fi
fi

# ─── 3. Docker Compose ─────────────────────────────────────────────────────────
echo ""
echo -e "${CYAN}[3/6]${NC} ${BOLD}Starting infrastructure (Weaviate + LocalAI + Reranker)...${NC}"

if [ "$SKIP_DOCKER" = true ]; then
    echo -e "  ${YELLOW}--no-docker set. Skipping Docker Compose.${NC}"
else
    # Pick compose command
    if docker compose version &> /dev/null 2>&1; then
        DC="docker compose"
    elif docker-compose --version &> /dev/null 2>&1; then
        DC="docker-compose"
    else
        echo -e "  ${RED}Docker Compose not found. Install it and try again.${NC}"; exit 1
    fi

    $DC up -d

    # Wait for services
    wait_for "http://localhost:8080/v1/.well-known/ready" "Weaviate"     120
    wait_for "http://localhost:8081/readyz"                "LocalAI"      180
    wait_for "http://localhost:8082/health"                "Reranker"     120
fi

# ─── 4. Ollama (optional) ──────────────────────────────────────────────────────
echo ""
echo -e "${CYAN}[4/6]${NC} ${BOLD}Checking Ollama (optional)...${NC}"

OLLAMA_PID=""
if command -v ollama &> /dev/null; then
    if curl -sf http://localhost:11434/api/tags > /dev/null 2>&1; then
        echo -e "  ${GREEN}✓${NC} Ollama already running"
    else
        echo -e "  ${YELLOW}Starting ollama serve...${NC}"
        ollama serve > /dev/null 2>&1 &
        OLLAMA_PID=$!
        for i in $(seq 1 10); do
            sleep 1
            if curl -sf http://localhost:11434/api/tags > /dev/null 2>&1; then
                echo -e "  ${GREEN}✓${NC} Ollama started (PID $OLLAMA_PID)"
                break
            fi
            if [ "$i" -eq 10 ]; then
                echo -e "  ${YELLOW}Ollama did not respond in time — continuing without it.${NC}"
            fi
        done
    fi
else
    echo -e "  ${YELLOW}Ollama not installed (optional). Get it at https://ollama.com${NC}"
fi

# ─── 5. Free ports ─────────────────────────────────────────────────────────────
echo ""
echo -e "${CYAN}[5/6]${NC} ${BOLD}Releasing ports 8090 and 3000...${NC}"
fuser -k 8090/tcp 2>/dev/null || true
fuser -k 3000/tcp 2>/dev/null || true
sleep 1
echo -e "  ${GREEN}✓${NC} Ports free"

# ─── Cleanup trap ──────────────────────────────────────────────────────────────
cleanup() {
    echo ""
    echo -e "${YELLOW}Shutting down GopherNotebook...${NC}"
    [ -n "$FRONTEND_PID" ] && kill "$FRONTEND_PID" 2>/dev/null || true
    [ -n "$BACKEND_PID"  ] && kill "$BACKEND_PID"  2>/dev/null || true
    if [ -n "$OLLAMA_PID" ]; then
        echo -e "  ${YELLOW}Stopping Ollama (started by this script)...${NC}"
        kill "$OLLAMA_PID" 2>/dev/null || true
    fi
    if [ "$SKIP_DOCKER" = false ] && [ -n "$DC" ]; then
        echo -e "  ${YELLOW}Stopping Docker services...${NC}"
        $DC stop
    fi
    echo -e "${GREEN}Shutdown complete. Goodbye!${NC}"
    exit 0
}
trap cleanup SIGINT SIGTERM

# ─── 6. App servers ────────────────────────────────────────────────────────────
echo ""
echo -e "${CYAN}[6/6]${NC} ${BOLD}Starting application servers...${NC}"

# Backend
echo -e "  Starting Go backend..."
cd backend
go mod tidy -e > /dev/null 2>&1 || true
go run ./cmd/server > server.log 2>&1 &
BACKEND_PID=$!
cd ..

# Wait for backend health
wait_for "http://localhost:8090/health" "Backend" 30

# Frontend
echo -e "  Starting Next.js frontend..."
cd frontend
if [ ! -d "node_modules" ]; then
    echo -e "  ${YELLOW}Installing npm dependencies (first run)...${NC}"
    npm install --silent
fi
npm run dev > frontend.log 2>&1 &
FRONTEND_PID=$!
cd ..

# ─── Ready banner ──────────────────────────────────────────────────────────────
echo ""
echo -e "${BLUE}${BOLD}╔══════════════════════════════════════════╗${NC}"
echo -e "${BLUE}${BOLD}║         🌟  Stack is LIVE  🌟            ║${NC}"
echo -e "${BLUE}${BOLD}╠══════════════════════════════════════════╣${NC}"
echo -e "${BLUE}${BOLD}║${NC}  App       ${GREEN}http://localhost:3000${NC}       ${BLUE}${BOLD}║${NC}"
echo -e "${BLUE}${BOLD}║${NC}  API       ${GREEN}http://localhost:8090${NC}       ${BLUE}${BOLD}║${NC}"
echo -e "${BLUE}${BOLD}║${NC}  Weaviate  ${GREEN}http://localhost:8080${NC}       ${BLUE}${BOLD}║${NC}"
echo -e "${BLUE}${BOLD}╠══════════════════════════════════════════╣${NC}"
echo -e "${BLUE}${BOLD}║${NC}  Logs: backend/server.log              ${BLUE}${BOLD}║${NC}"
echo -e "${BLUE}${BOLD}║${NC}        frontend/frontend.log           ${BLUE}${BOLD}║${NC}"
echo -e "${BLUE}${BOLD}╠══════════════════════════════════════════╣${NC}"
echo -e "${BLUE}${BOLD}║${NC}  Press ${YELLOW}Ctrl+C${NC} to stop all services    ${BLUE}${BOLD}║${NC}"
echo -e "${BLUE}${BOLD}╚══════════════════════════════════════════╝${NC}"
echo ""

wait
