# 🍃 Foodie - Full-Stack Food Delivery Platform

[![Go Version](https://img.shields.io/badge/Go-1.23-blue.svg)](https://golang.org)
[![Fiber](https://img.shields.io/badge/Fiber-v2.52.5-00b0ff.svg)](https://gofiber.io)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-336791.svg)](https://postgresql.org)
[![Redis](https://img.shields.io/badge/Redis-7-dc382d.svg)](https://redis.io)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

> 🚀 **A production-ready food delivery platform inspired by Deliveroo**

[Live Demo](https://foodie-demo.pages.dev) • [API Docs](#api-documentation) • [Deployment Guide](#-deployment)

![Architecture Diagram](docs/architecture.png)

---

## ✨ Features

### 🛍️ For Customers
- 🔍 Browse restaurants with search & filters
- 📋 View detailed menus with images
- 🛒 Shopping cart with real-time updates
- 📍 Multiple delivery addresses
- 💳 Multiple payment methods (Cards, Wallet, Cash)
- 📱 Real-time order tracking
- ⭐ Rate & review orders

### 🍽️ For Restaurant Owners
- 📊 Dashboard with analytics
- 📝 Menu management
- 📦 Order management system
- ⏰ Opening hours control
- 📈 Sales reports
- 💬 Customer reviews

### 🛵 For Drivers
- 📍 Real-time location tracking
- 🔔 Instant order notifications
- 🗺️ Route optimization
- 💰 Earnings dashboard
- 📊 Performance metrics

### ⚙️ For Admin
- 👥 User management
- 🏪 Restaurant approval
- 📊 System analytics
- 🎟️ Promo codes management
- 📈 Revenue reports

---

## 🏗️ Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                     Cloudflare Pages                        │
│                   (Static Frontend Hosting)                   │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    Cloudflare Workers                       │
│                     (API Gateway / Edge)                      │
└─────────────────────────────────────────────────────────────┘
                              │
            ┌─────────────────┼─────────────────┐
            ▼                 ▼                 ▼
    ┌───────────┐    ┌─────────────┐    ┌───────────┐
    │   Go API  │    │  WebSocket  │    │  Redis    │
    │  (Docker) │    │    Server   │    │   Cache   │
    └─────┬─────┘    └─────────────┘    └───────────┘
          │
          ▼
    ┌─────────────┐
    │  PostgreSQL │
    │   (Docker)  │
    └─────────────┘
```

---

## 🚀 Quick Start

### Prerequisites
- [Go 1.23+](https://golang.org/dl/)
- [Docker & Docker Compose](https://docs.docker.com/get-docker/)
- [Node.js 18+](https://nodejs.org/) (for Cloudflare deployment)

### 1. Clone Repository
```bash
git clone https://github.com/yourusername/foodie.git
cd foodie
```

### 2. Setup Environment
```bash
# Copy environment template
cp .env.example .env

# Edit .env with your values
nano .env
```

### 3. Start Services
```bash
# Start all services with Docker Compose
docker-compose up -d

# Or start individually
docker-compose up -d postgres redis
cd backend && go run ./cmd/server
```

### 4. Access Applications
| App | URL |
|-----|-----|
| 🛍️ Customer | http://localhost:8080 |
| 🍽️ Restaurant | http://localhost:8080/restaurant |
| 🛵 Driver | http://localhost:8080/driver |
| ⚙️ Admin | http://localhost:8080/admin |
| 🔌 API | http://localhost:8080/api/v1/ |

---

## 📁 Project Structure

```
foodie/
├── 📁 backend/                    # Go Backend API
│   ├── 📁 cmd/server/             # Application entry points
│   ├── 📁 internal/
│   │   ├── 📁 config/             # Configuration management
│   │   ├── 📁 database/           # PostgreSQL & Redis
│   │   ├── 📁 handlers/           # HTTP handlers
│   │   ├── 📁 middleware/         # Auth, CORS, Rate limiting
│   │   ├── 📁 models/             # Data models
│   │   └── 📁 websocket/          # Real-time communication
│   ├── 📄 go.mod                  # Go dependencies
│   └── 📄 Dockerfile              # Backend container
│
├── 📁 frontend/                   # Static Frontend Applications
│   ├── 📁 customer/             # Customer ordering app
│   ├── 📁 restaurant/             # Restaurant dashboard
│   ├── 📁 driver/                 # Driver mobile app
│   └── 📁 admin/                  # Admin panel
│
├── 📁 nginx/                      # Nginx configuration
├── 📁 scripts/                    # Database migration scripts
├── 📁 docs/                       # Documentation
├── 📁 .github/                    # GitHub Actions workflows
├── 📄 docker-compose.yml          # Docker orchestration
├── 📄 wrangler.toml               # Cloudflare configuration
└── 📄 README.md                   # This file
```

---

## 🔌 API Documentation

### Authentication
```http
POST /api/v1/auth/register
POST /api/v1/auth/login
POST /api/v1/auth/refresh
GET  /api/v1/auth/me
```

### Restaurants
```http
GET /api/v1/restaurants              # List restaurants
GET /api/v1/restaurants/search       # Search restaurants
GET /api/v1/restaurants/featured     # Featured restaurants
GET /api/v1/restaurants/:slug        # Restaurant details
GET /api/v1/restaurants/:id/menu     # Restaurant menu
```

### Orders
```http
POST /api/v1/orders/                 # Create order
GET  /api/v1/orders/:id              # Get order details
POST /api/v1/orders/:id/cancel       # Cancel order
GET  /api/v1/orders/:id/track        # Track order
```

### Drivers
```http
GET  /api/v1/driver/orders/available # Available orders
POST /api/v1/driver/orders/:id/accept
PUT  /api/v1/driver/location         # Update location
GET  /api/v1/driver/earnings         # Earnings report
```

---

## ☁️ Deployment

### 🌐 Deploy to Cloudflare Pages (Frontend)

```bash
# Install Wrangler
npm install -g wrangler

# Login to Cloudflare
wrangler login

# Deploy frontend
cd frontend/customer
wrangler pages deploy . --project-name=foodie-customer

cd ../restaurant
wrangler pages deploy . --project-name=foodie-restaurant
```

### 🐳 Deploy Backend to Cloudflare Workers

```bash
# Build Go binary for Workers
GOOS=js GOARCH=wasm go build -o main.wasm ./cmd/server

# Deploy
wrangler deploy
```

### 🚀 Deploy to VPS (Production)

```bash
# Clone on server
git clone https://github.com/yourusername/foodie.git
cd foodie

# Setup environment
cp .env.example .env
# Edit .env with production values

# Start with Docker Compose
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

---

## 🧪 Testing

```bash
# Run unit tests
cd backend
go test ./... -v

# Run tests with coverage
go test -cover ./...

# Run specific test
go test -run TestAuth -v
```

---

## 🛣️ Roadmap

### ✅ Completed
- [x] Core API with Go & Fiber
- [x] PostgreSQL + Redis integration
- [x] JWT Authentication
- [x] WebSocket real-time updates
- [x] Customer, Restaurant, Driver, Admin interfaces
- [x] Docker containerization
- [x] CI/CD with GitHub Actions

### 🚧 In Progress
- [ ] Flutter mobile app
- [ ] Stripe payment integration
- [ ] Firebase push notifications
- [ ] Google Maps integration

### 📅 Planned
- [ ] AI-powered recommendations
- [ ] Multi-language support
- [ ] Advanced analytics dashboard
- [ ] White-label solution

---

## 🤝 Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

1. Fork the repository
2. Create your feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

---

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

---

## 👨‍💻 Authors

- **Your Name** - *Initial work* - [YourGithub](https://github.com/yourusername)

---

## 🙏 Acknowledgments

- Inspired by [Deliveroo](https://deliveroo.com)
- Built with [Go Fiber](https://gofiber.io)
- Hosted on [Cloudflare](https://cloudflare.com)

---

<p align="center">
  Made with ❤️ and ☕
</p>
