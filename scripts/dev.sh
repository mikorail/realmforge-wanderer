#!/bin/sh
# System node (v21) too old for Vite 8 → force nvm v24
export PATH="/Users/pratitasujatmika/.nvm/versions/node/v24.14.0/bin:$PATH"
cd "$(dirname "$0")/.." && exec npm run dev -- --port 5195 --strictPort
