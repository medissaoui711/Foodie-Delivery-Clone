package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
)

func main() {
	app := fiber.New(fiber.Config{
		AppName: "Deliveroo Clone API v1.0",
	})

	app.Use(logger.New())

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":    "ok",
			"timestamp": "2026-05-06T02:39:00Z",
			"version":   "1.0.0",
		})
	})

	// API v1 routes
	api := app.Group("/api/v1")

	// Mock auth endpoints
	auth := api.Group("/auth")
	auth.Post("/login", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"token": "mock-jwt-token",
			"user": fiber.Map{
				"id":         1,
				"email":      "test@example.com",
				"first_name": "Test",
				"last_name":  "User",
			},
		})
	})

	auth.Post("/register", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "User registered successfully",
			"user": fiber.Map{
				"id":         2,
				"email":      "new@example.com",
				"first_name": "New",
				"last_name":  "User",
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

	api.Get("/restaurants/featured", func(c *fiber.Ctx) error {
		featured := []fiber.Map{
			{
				"id":          "r1",
				"name":        "Burger Palace",
				"emoji":       "🍔",
				"rating":      4.9,
				"time":        "18-28",
				"fee":         0.0,
				"min":         10.0,
				"description": "Gourmet smash burgers",
				"open":        true,
				"tags":        []string{"Popular", "New"},
			},
		}
		return c.JSON(fiber.Map{
			"featured": featured,
		})
	})

	// Serve frontend files
	app.Static("/", "../frontend/customer")
	app.Get("/", func(c *fiber.Ctx) error {
		return c.SendFile("../frontend/customer/index.html")
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("🚀 Server running on http://localhost:%s", port)
	log.Printf("🍔 Customer app: http://localhost:%s/", port)
	log.Printf("🔌 API: http://localhost:%s/api/v1/", port)
	log.Printf("❤️ Health: http://localhost:%s/health", port)

	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
