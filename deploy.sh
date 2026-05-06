#!/bin/bash
# ============================================
# DELIVEROO CLONE — DEPLOYMENT SCRIPT
# Run: chmod +x deploy.sh && ./deploy.sh
# ============================================
set -e

echo "🚀 Deploying Deliveroo Clone..."

# Check .env exists
if [ ! -f .env ]; then
  echo "❌ .env not found. Copy .env.example to .env and fill in values."
  exit 1
fi

# Load env
export $(grep -v '^#' .env | xargs)

echo "📦 Building containers..."
docker-compose build --no-cache

echo "🗄️ Starting database services..."
docker-compose up -d postgres redis

echo "⏳ Waiting for database to be ready..."
sleep 10

echo "🚀 Starting all services..."
docker-compose up -d

echo "✅ All services started!"
echo ""
echo "🌐 Customer App:    http://localhost/"
echo "🍽️  Restaurant App: http://localhost/restaurant/"
echo "🛵  Driver App:     http://localhost/driver/"
echo "⚙️  Admin Panel:    http://localhost/admin/"
echo "🔌 API:            http://localhost/api/v1/"
echo "❤️  Health:         http://localhost/health"
echo ""
echo "📊 Container status:"
docker-compose ps
