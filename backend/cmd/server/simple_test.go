package main

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
)

func TestHealthEndpointSimple(t *testing.T) {
	app := fiber.New()

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"version": "1.0.0",
		})
	})

	// Create HTTP request
	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Parse response
	var result map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&result)
	assert.NoError(t, err)
	assert.Equal(t, "ok", result["status"])
	assert.Equal(t, "1.0.0", result["version"])
}

func TestBasicAPI(t *testing.T) {
	app := fiber.New()

	api := app.Group("/api/v1")

	api.Get("/restaurants", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"restaurants": []fiber.Map{},
			"total":       0,
		})
	})

	api.Get("/restaurants/featured", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"featured": []fiber.Map{},
		})
	})

	// Test restaurants endpoint
	req := httptest.NewRequest("GET", "/api/v1/restaurants", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Test featured endpoint
	req = httptest.NewRequest("GET", "/api/v1/restaurants/featured", nil)
	resp, err = app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestAuthEndpoints(t *testing.T) {
	app := fiber.New()

	api := app.Group("/api/v1")
	auth := api.Group("/auth")

	auth.Post("/register", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"message": "User registered successfully",
		})
	})

	auth.Post("/login", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"token": "mock-jwt-token",
			"user":  fiber.Map{"id": 1, "email": "test@example.com"},
		})
	})

	// Test register
	registerData := map[string]string{
		"email":    "test@example.com",
		"password": "password123",
		"name":     "Test User",
	}
	jsonData, _ := json.Marshal(registerData)
	req := httptest.NewRequest("POST", "/api/v1/auth/register", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)

	// Test login
	loginData := map[string]string{
		"email":    "test@example.com",
		"password": "password123",
	}
	jsonData, _ = json.Marshal(loginData)
	req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")
	resp, err = app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
}

func TestErrorHandling(t *testing.T) {
	app := fiber.New()

	app.Get("/error", func(c *fiber.Ctx) error {
		return fiber.NewError(fiber.StatusNotFound, "Not found")
	})

	app.Get("/panic", func(c *fiber.Ctx) error {
		panic("test panic")
	})

	// Test error response
	req := httptest.NewRequest("GET", "/error", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)
}

func TestMiddleware(t *testing.T) {
	app := fiber.New()

	// Add CORS middleware simulation
	app.Use(func(c *fiber.Ctx) error {
		c.Set("Access-Control-Allow-Origin", "*")
		return c.Next()
	})

	app.Get("/test", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"message": "test"})
	})

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	assert.NoError(t, err)
	assert.Equal(t, 200, resp.StatusCode)
	assert.Equal(t, "*", resp.Header.Get("Access-Control-Allow-Origin"))
}
