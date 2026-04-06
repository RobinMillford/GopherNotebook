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
for req in docker go npm; do
    if ! command -v $req &> /dev/null; then
        echo -e "${RED}Error: '$req' is not installed or not in PATH.${NC}"
        exit 1
    fi
done
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

# 3. Start Core Infrastructure (Docker Compose)
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

# 4. Start Go Backend
echo -e "${YELLOW}Starting Go Backend (http://localhost:8090)...${NC}"
cd backend
go mod tidy
go run ./cmd/server > server.log 2>&1 &
BACKEND_PID=$!
cd ..

# 5. Start Next.js Frontend
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
