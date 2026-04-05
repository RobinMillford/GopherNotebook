#!/bin/bash

# GopherNotebook - Download Open-Source Local Models
# This script downloads the exact Qwen3 embedding and reranker models used by the system.
# Run this script after cloning the repository if the models are missing.

set -e

MODELS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)/models"
mkdir -p "$MODELS_DIR"

echo "Downloading Qwen3 Embedding Model..."
wget -q --show-progress "https://huggingface.co/enacimie/Qwen3-Embedding-0.6B-Q4_K_M-GGUF/resolve/main/qwen3-embedding-0.6b-q4_k_m.gguf" -O "$MODELS_DIR/qwen3-embedding-0.6b-q4_k_m.gguf"

echo "Downloading Qwen3 Reranker Model..."
wget -q --show-progress "https://huggingface.co/mradermacher/Qwen3-Reranker-0.6B-GGUF/resolve/main/Qwen3-Reranker-0.6B.Q4_K_M.gguf" -O "$MODELS_DIR/Qwen3-Reranker-0.6B.Q4_K_M.gguf"

echo "Download complete! Models are ready in $MODELS_DIR/"
