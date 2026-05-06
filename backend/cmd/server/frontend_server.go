package main

import (
	"fmt"
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	app := fiber.New(fiber.Config{
		AppName: "Deliveroo Clone Frontend Server",
	})

	app.Use(logger.New())

	// API endpoints for testing
	api := app.Group("/api/v1")

	api.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":    "ok",
			"timestamp": "2026-05-07T00:54:00Z",
			"version":   "1.0.0",
		})
	})

	// Mock auth
	api.Post("/auth/login", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"token": "mock-jwt-token",
			"user": fiber.Map{
				"id":         1,
				"email":      "admin@deliveroo.clone",
				"first_name": "Admin",
				"last_name":  "User",
				"role":       "admin",
			},
		})
	})

	// Mock restaurants
	api.Get("/restaurants", func(c *fiber.Ctx) error {
		restaurants := []fiber.Map{
			{
				"id":          "r1",
				"name":        "Burger Palace",
				"emoji":       "🍔",
				"category":    "burgers",
				"rating":      4.9,
				"time":        "18-28",
				"fee":         0.0,
				"min":         10.0,
				"description": "Gourmet smash burgers",
				"open":        true,
				"tags":        []string{"Popular", "New"},
			},
			{
				"id":          "r2",
				"name":        "Pizza Roma",
				"emoji":       "🍕",
				"category":    "pizza",
				"rating":      4.7,
				"time":        "22-35",
				"fee":         1.99,
				"min":         12.0,
				"description": "Authentic Neapolitan pizza",
				"open":        true,
				"tags":        []string{"Vegan options"},
			},
		}
		return c.JSON(fiber.Map{
			"restaurants": restaurants,
			"total":       len(restaurants),
		})
	})

	// Mock orders
	api.Get("/orders", func(c *fiber.Ctx) error {
		orders := []fiber.Map{
			{
				"id":     "FD8K2P",
				"status": "preparing",
				"total":  25.90,
				"time":   "25-35 min",
			},
		}
		return c.JSON(orders)
	})

	// Route handler for different frontend apps
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendFile("../../frontend/customer/index.html")
	})

	app.Get("/customer", func(c *fiber.Ctx) error {
		return c.SendFile("../../frontend/customer/index.html")
	})

	app.Get("/restaurant", func(c *fiber.Ctx) error {
		return c.SendFile("../../frontend/restaurant/index.html")
	})

	app.Get("/driver", func(c *fiber.Ctx) error {
		return c.SendFile("../../frontend/driver/index.html")
	})

	app.Get("/admin", func(c *fiber.Ctx) error {
		return c.SendFile("../../frontend/admin/index.html")
	})

	// Static files for each app
	app.Static("/customer", "../../frontend/customer")
	app.Static("/restaurant", "../../frontend/restaurant")
	app.Static("/driver", "../../frontend/driver")
	app.Static("/admin", "../../frontend/admin")

	// Main navigation page
	app.Get("/apps", func(c *fiber.Ctx) error {
		html := `
<!DOCTYPE html>
<html>
<head>
    <title>Deliveroo Clone - All Apps</title>
    <style>
        body { font-family: Arial, sans-serif; max-width: 800px; margin: 50px auto; padding: 20px; }
        .app-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(300px, 1fr)); gap: 20px; margin-top: 30px; }
        .app-card { border: 2px solid #00ccb4; border-radius: 12px; padding: 20px; text-decoration: none; color: #333; transition: all 0.3s; }
        .app-card:hover { transform: translateY(-5px); box-shadow: 0 10px 25px rgba(0,204,180,0.2); }
        .app-title { font-size: 24px; font-weight: bold; margin-bottom: 10px; color: #00ccb4; }
        .app-desc { color: #666; line-height: 1.5; }
        .app-emoji { font-size: 48px; margin-bottom: 15px; }
        h1 { text-align: center; color: #333; margin-bottom: 10px; }
        .subtitle { text-align: center; color: #666; margin-bottom: 30px; }
    </style>
</head>
<body>
    <h1>🍃 Foodie - Deliveroo Clone</h1>
    <p class="subtitle">Choose an interface to test:</p>
    
    <div class="app-grid">
        <a href="/customer" class="app-card">
            <div class="app-emoji">🛍️</div>
            <div class="app-title">Customer App</div>
            <div class="app-desc">Browse restaurants, place orders, and track delivery. Full food ordering experience.</div>
        </a>
        
        <a href="/restaurant" class="app-card">
            <div class="app-emoji">🍽️</div>
            <div class="app-title">Restaurant Owner</div>
            <div class="app-desc">Manage menu, handle orders, and view analytics dashboard.</div>
        </a>
        
        <a href="/driver" class="app-card">
            <div class="app-emoji">🛵</div>
            <div class="app-title">Driver App</div>
            <div class="app-desc">Accept orders, track location, and manage deliveries.</div>
        </a>
        
        <a href="/admin" class="app-card">
            <div class="app-emoji">⚙️</div>
            <div class="app-title">Admin Panel</div>
            <div class="app-desc">System administration, user management, and analytics.</div>
        </a>
    </div>
    
    <div style="margin-top: 40px; padding: 20px; background: #f5f5f5; border-radius: 8px;">
        <h3>🔗 API Endpoints</h3>
        <ul>
            <li><a href="/api/v1/health" target="_blank">Health Check</a></li>
            <li><a href="/api/v1/restaurants" target="_blank">Restaurants API</a></li>
            <li><a href="/api/v1/orders" target="_blank">Orders API</a></li>
        </ul>
    </div>
</body>
</html>`
		return c.Type("html").SendString(html)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8084"
	}

	fmt.Printf("🚀 Frontend Server running on http://localhost:%s\n", port)
	fmt.Printf("📱 All Apps: http://localhost:%s/apps\n", port)
	fmt.Printf("🛍️ Customer: http://localhost:%s/customer\n", port)
	fmt.Printf("🍽️ Restaurant: http://localhost:%s/restaurant\n", port)
	fmt.Printf("🛵 Driver: http://localhost:%s/driver\n", port)
	fmt.Printf("⚙️ Admin: http://localhost:%s/admin\n", port)
	fmt.Printf("🔌 API: http://localhost:%s/api/v1/\n", port)

	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
