#!/bin/bash

# GopherNotebook Quickstart Script
# This script configures, installs dependencies, and launches the entire stack.

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${BLUE}=======================================${NC}"
echo -e "${BLUE} 🚀 Starting GopherNotebook...${NC}"
echo -e "${BLUE}=======================================${NC}"

# 1. Check prerequisites
echo -e "${YELLOW}Checking prerequisites...${NC}"

OS="$(uname -s)"
case "${OS}" in
    Linux*)     MACHINE=Linux;;
    Darwin*)    MACHINE=Mac;;
    CYGWIN*|MINGW*|MSYS*) MACHINE=Windows;;
    *)          MACHINE=Other;;
esac

# Check and install Go if needed
if ! command -v go &> /dev/null; then
    echo -e "${YELLOW}'go' is not installed. Attempting to install Go for ${MACHINE}...${NC}"
    if [ "$MACHINE" = "Linux" ]; then
        GO_VERSION="1.22.2"
        echo "Downloading Go ${GO_VERSION}..."
        curl -LO "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz" || wget "https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz"
        rm -rf "$HOME/.local/go"
        mkdir -p "$HOME/.local"
        tar -C "$HOME/.local" -xzf "go${GO_VERSION}.linux-amd64.tar.gz"
        rm "go${GO_VERSION}.linux-amd64.tar.gz"
        export PATH="$PATH:$HOME/.local/go/bin"
        if [ -f "$HOME/.bashrc" ] && ! grep -q "\.local/go/bin" "$HOME/.bashrc"; then
            echo 'export PATH=$PATH:$HOME/.local/go/bin' >> "$HOME/.bashrc"
        fi
        echo -e "${GREEN}✓ Go installed successfully to ~/.local/go.${NC}"
    elif [ "$MACHINE" = "Mac" ]; then
        if ! command -v brew &> /dev/null; then echo -e "${RED}Homebrew not found. Please install brew or Go manually.${NC}"; exit 1; fi
        brew install go
    elif [ "$MACHINE" = "Windows" ]; then
        winget install GoLang.Go
        echo -e "${YELLOW}Please restart your terminal to apply the new PATH variables after Winget installations.${NC}"
    else
        echo -e "${RED}Error: 'go' is not installed. Auto-install is only supported on Linux, Mac, and Windows.${NC}"
        exit 1
    fi
fi

# Check and install Node.js/npm if needed
if ! command -v npm &> /dev/null; then
    echo -e "${YELLOW}'npm' is not installed. Attempting to install Node.js for ${MACHINE}...${NC}"
    if [ "$MACHINE" = "Linux" ]; then
        curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.7/install.sh | bash
        export NVM_DIR="$HOME/.nvm"
        [ -s "$NVM_DIR/nvm.sh" ] && \. "$NVM_DIR/nvm.sh"
        nvm install 20
        nvm use 20
        echo -e "${GREEN}✓ Node.js and npm installed successfully.${NC}"
    elif [ "$MACHINE" = "Mac" ]; then
        if ! command -v brew &> /dev/null; then echo -e "${RED}Homebrew not found. Please install Node manually.${NC}"; exit 1; fi
        brew install node
    elif [ "$MACHINE" = "Windows" ]; then
        winget install OpenJS.NodeJS
    else
        echo -e "${RED}Error: 'npm' is not installed. Auto-install is only supported on Linux, Mac, and Windows.${NC}"
        exit 1
    fi
fi

# Check and install Docker if needed
if ! command -v docker &> /dev/null; then
    echo -e "${YELLOW}'docker' is not installed. Attempting to install Docker for ${MACHINE}...${NC}"
    if [ "$MACHINE" = "Linux" ]; then
        echo -e "${YELLOW}Note: Docker installation may require your sudo password.${NC}"
        curl -fsSL https://get.docker.com -o get-docker.sh
        sudo sh get-docker.sh
        rm get-docker.sh
        sudo usermod -aG docker $USER
        echo -e "${GREEN}✓ Docker installed successfully.${NC}"
        echo -e "${YELLOW}IMPORTANT: You might need to restart your terminal or run 'su - $USER' for Docker group permissions to apply.${NC}"
    elif [ "$MACHINE" = "Mac" ]; then
        if ! command -v brew &> /dev/null; then echo -e "${RED}Homebrew not found. Please install Docker manually.${NC}"; exit 1; fi
        brew install --cask docker
        echo -e "${YELLOW}IMPORTANT: Open Docker Desktop from your Applications folder to finish setup.${NC}"
    elif [ "$MACHINE" = "Windows" ]; then
        winget install Docker.DockerDesktop
        echo -e "${YELLOW}IMPORTANT: You must restart Windows mapping and open Docker Desktop to finish setup.${NC}"
    else
        echo -e "${RED}Error: 'docker' is not installed. Auto-install is only supported on Linux, Mac, and Windows.${NC}"
        exit 1
    fi
fi

echo -e "${GREEN}✓ All prerequisites met.${NC}"

# 2. Check and Download Models
MODELS_DIR="./models"
MODEL_1="$MODELS_DIR/qwen3-embedding-0.6b-q4_k_m.gguf"
MODEL_2="$MODELS_DIR/Qwen3-Reranker-0.6B.Q4_K_M.gguf"

if [[ ! -f "$MODEL_1" ]] || [[ ! -f "$MODEL_2" ]]; then
    echo -e "${YELLOW}Local AI Models are missing. Initiating download...${NC}"
    if [[ -f "./scripts/download_models.sh" ]]; then
        bash ./scripts/download_models.sh
    else
        echo -e "${RED}Error: scripts/download_models.sh not found.${NC}"
        exit 1
    fi
else
    echo -e "${GREEN}✓ Local AI Models found.${NC}"
fi

# Determine the correct docker compose command
if docker compose version &> /dev/null; then
    DOCKER_COMPOSE_CMD="docker compose"
elif docker-compose --version &> /dev/null; then
    DOCKER_COMPOSE_CMD="docker-compose"
else
    echo -e "${RED}Error: Docker Compose is not installed or not in PATH.${NC}"
    exit 1
fi

# 3. Free ports before starting (kill any stale processes)
echo -e "${YELLOW}Freeing ports 8090 and 3000...${NC}"
fuser -k 8090/tcp 2>/dev/null || true
fuser -k 3000/tcp 2>/dev/null || true
sleep 1

# 4. Start Core Infrastructure (Docker Compose)
echo -e "${YELLOW}Starting core infrastructure (Weaviate & LocalAI)...${NC}"
$DOCKER_COMPOSE_CMD up -d

# Process management for cleanup
cleanup() {
    echo -e "\n${YELLOW}Shutting down GopherNotebook Engine...${NC}"
    
    if [ -n "$FRONTEND_PID" ]; then
        kill $FRONTEND_PID 2>/dev/null || true
    fi
    if [ -n "$BACKEND_PID" ]; then
        kill $BACKEND_PID 2>/dev/null || true
    fi
    
    echo -e "${YELLOW}Stopping docker containers...${NC}"
    $DOCKER_COMPOSE_CMD stop
    echo -e "${GREEN}✓ Shutdown complete. Goodbye!${NC}"
    exit
}

trap cleanup SIGINT SIGTERM EXIT

# 5. Start Go Backend
echo -e "${YELLOW}Starting Go Backend (http://localhost:8090)...${NC}"
cd backend
go mod tidy
go run ./cmd/server > server.log 2>&1 &
BACKEND_PID=$!
cd ..

# 6. Start Next.js Frontend
echo -e "${YELLOW}Starting Next.js Frontend (http://localhost:3000)...${NC}"
cd frontend
if [ ! -d "node_modules" ]; then
    echo -e "${YELLOW}Installing frontend dependencies...${NC}"
    npm install
fi
npm run dev > frontend.log 2>&1 &
FRONTEND_PID=$!
cd ..

echo -e "${BLUE}=======================================${NC}"
echo -e "${GREEN}🌟 GopherNotebook is LIVE 🌟${NC}"
echo -e "Frontend: ${GREEN}http://localhost:3000${NC}"
echo -e "Backend:  ${GREEN}http://localhost:8090${NC}"
echo -e ""
echo -e "Logs are being written to:"
echo -e " - backend/server.log"
echo -e " - frontend/frontend.log"
echo -e ""
echo -e "${YELLOW}Press Ctrl+C to stop all services gracefully.${NC}"
echo -e "${BLUE}=======================================${NC}"

# Wait indefinitely for signals
wait
