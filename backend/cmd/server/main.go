package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/deliveroo-clone/internal/config"
	"github.com/deliveroo-clone/internal/database"
	"github.com/deliveroo-clone/internal/handlers"
	"github.com/deliveroo-clone/internal/middleware"
	"github.com/deliveroo-clone/internal/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/helmet"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	fiberws "github.com/gofiber/websocket/v2"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Connect to PostgreSQL
	db, err := database.NewPostgres(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to PostgreSQL: %v", err)
	}
	defer db.Close()

	// Connect to Redis
	rdb, err := database.NewRedis(cfg.RedisURL)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer rdb.Close()

	// Init WebSocket hub
	hub := websocket.NewHub(rdb)
	go hub.Run()

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:      "Deliveroo Clone API v1.0",
		ErrorHandler: middleware.ErrorHandler,
		BodyLimit:    10 * 1024 * 1024, // 10MB
	})

	// Global middleware
	app.Use(recover.New())
	app.Use(helmet.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path} (${latency})\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOriginsFunc: func(origin string) bool {
			return cfg.AppEnv == "development"
		},
		AllowOrigins:     cfg.AllowedOrigins,
		AllowMethods:     "GET,POST,PUT,PATCH,DELETE,OPTIONS",
		AllowHeaders:     "Origin,Content-Type,Accept,Authorization,X-Request-ID",
		AllowCredentials: true,
		MaxAge:           86400,
	}))

	// Rate limiting
	app.Use("/api/", limiter.New(limiter.Config{
		Max:        100,
		Expiration: time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
	}))

	// Initialize handlers
	h := handlers.NewHandler(db, rdb, hub, cfg)

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":    "ok",
			"timestamp": time.Now().UTC(),
			"version":   "1.0.0",
		})
	})

	// WebSocket endpoint
	app.Use("/ws", func(c *fiber.Ctx) error {
		if fiberws.IsWebSocketUpgrade(c) {
			c.Locals("allowed", true)
			return c.Next()
		}
		return fiber.ErrUpgradeRequired
	})
	app.Get("/ws/:token", fiberws.New(hub.HandleWebSocket))

	// API v1 routes
	api := app.Group("/api/v1")

	// ---- AUTH ----
	auth := api.Group("/auth")
	auth.Post("/register", h.Auth.Register)
	auth.Post("/login", h.Auth.Login)
	auth.Post("/logout", middleware.Auth(cfg), h.Auth.Logout)
	auth.Post("/refresh", h.Auth.RefreshToken)
	auth.Post("/forgot-password", h.Auth.ForgotPassword)
	auth.Post("/reset-password", h.Auth.ResetPassword)
	auth.Post("/verify-email", h.Auth.VerifyEmail)
	auth.Get("/me", middleware.Auth(cfg), h.Auth.Me)

	// ---- CUSTOMERS ----
	customers := api.Group("/customers", middleware.Auth(cfg), middleware.RequireRole("customer", "admin"))
	customers.Get("/profile", h.Customer.GetProfile)
	customers.Put("/profile", h.Customer.UpdateProfile)
	customers.Get("/addresses", h.Customer.ListAddresses)
	customers.Post("/addresses", h.Customer.CreateAddress)
	customers.Put("/addresses/:id", h.Customer.UpdateAddress)
	customers.Delete("/addresses/:id", h.Customer.DeleteAddress)
	customers.Get("/orders", h.Customer.ListOrders)
	customers.Get("/orders/:id", h.Customer.GetOrder)
	customers.Post("/orders/:id/review", h.Customer.ReviewOrder)
	customers.Get("/wallet", h.Customer.GetWallet)
	customers.Get("/wallet/transactions", h.Customer.ListWalletTransactions)
	customers.Get("/notifications", h.Customer.ListNotifications)
	customers.Put("/notifications/:id/read", h.Customer.MarkNotificationRead)
	customers.Put("/notifications/read-all", h.Customer.MarkAllNotificationsRead)
	customers.Get("/payment-cards", h.Customer.ListPaymentCards)
	customers.Post("/payment-cards", h.Customer.AddPaymentCard)
	customers.Delete("/payment-cards/:id", h.Customer.DeletePaymentCard)
	customers.Post("/payment-cards/:id/default", h.Customer.SetDefaultCard)

	// ---- RESTAURANTS (Public) ----
	restaurants := api.Group("/restaurants")
	restaurants.Get("/", h.Restaurant.List)
	restaurants.Get("/search", h.Restaurant.Search)
	restaurants.Get("/categories", h.Restaurant.ListCategories)
	restaurants.Get("/featured", h.Restaurant.ListFeatured)
	restaurants.Get("/:slug", h.Restaurant.GetBySlug)
	restaurants.Get("/:id/menu", h.Restaurant.GetMenu)
	restaurants.Get("/:id/reviews", h.Restaurant.GetReviews)

	// ---- RESTAURANT OWNER ----
	owner := api.Group("/owner", middleware.Auth(cfg), middleware.RequireRole("restaurant_owner", "admin"))
	owner.Get("/restaurant", h.Owner.GetMyRestaurant)
	owner.Post("/restaurant", h.Owner.CreateRestaurant)
	owner.Put("/restaurant", h.Owner.UpdateRestaurant)
	owner.Put("/restaurant/hours", h.Owner.UpdateHours)
	owner.Post("/restaurant/toggle-open", h.Owner.ToggleOpen)
	owner.Get("/dashboard", h.Owner.GetDashboard)
	owner.Get("/orders", h.Owner.ListOrders)
	owner.Get("/orders/:id", h.Owner.GetOrder)
	owner.Put("/orders/:id/status", h.Owner.UpdateOrderStatus)
	owner.Get("/menu/sections", h.Owner.ListMenuSections)
	owner.Post("/menu/sections", h.Owner.CreateMenuSection)
	owner.Put("/menu/sections/:id", h.Owner.UpdateMenuSection)
	owner.Delete("/menu/sections/:id", h.Owner.DeleteMenuSection)
	owner.Get("/menu/items", h.Owner.ListMenuItems)
	owner.Post("/menu/items", h.Owner.CreateMenuItem)
	owner.Put("/menu/items/:id", h.Owner.UpdateMenuItem)
	owner.Delete("/menu/items/:id", h.Owner.DeleteMenuItem)
	owner.Put("/menu/items/:id/availability", h.Owner.ToggleItemAvailability)
	owner.Post("/menu/items/:id/options", h.Owner.AddItemOptions)
	owner.Get("/analytics", h.Owner.GetAnalytics)
	owner.Get("/reviews", h.Owner.GetReviews)
	owner.Post("/reviews/:id/reply", h.Owner.ReplyReview)

	// ---- ORDERS (Shared) ----
	orders := api.Group("/orders", middleware.Auth(cfg))
	orders.Post("/", middleware.RequireRole("customer"), h.Order.Create)
	orders.Get("/:id", h.Order.GetByID)
	orders.Post("/:id/cancel", h.Order.Cancel)
	orders.Get("/:id/track", h.Order.Track)
	orders.Post("/promo/validate", h.Order.ValidatePromo)

	// ---- PAYMENTS ----
	payments := api.Group("/payments", middleware.Auth(cfg))
	payments.Post("/intent", h.Payment.CreateIntent)
	payments.Post("/confirm", h.Payment.ConfirmPayment)
	payments.Post("/webhook", h.Payment.StripeWebhook)

	// ---- DRIVERS ----
	driver := api.Group("/driver", middleware.Auth(cfg), middleware.RequireRole("driver", "admin"))
	driver.Get("/profile", h.Driver.GetProfile)
	driver.Post("/profile", h.Driver.CreateProfile)
	driver.Put("/profile", h.Driver.UpdateProfile)
	driver.Post("/toggle-status", h.Driver.ToggleStatus)
	driver.Put("/location", h.Driver.UpdateLocation)
	driver.Get("/orders/available", h.Driver.GetAvailableOrders)
	driver.Get("/orders/active", h.Driver.GetActiveOrder)
	driver.Get("/orders/history", h.Driver.GetOrderHistory)
	driver.Post("/orders/:id/accept", h.Driver.AcceptOrder)
	driver.Post("/orders/:id/reject", h.Driver.RejectOrder)
	driver.Put("/orders/:id/status", h.Driver.UpdateOrderStatus)
	driver.Get("/earnings", h.Driver.GetEarnings)
	driver.Get("/dashboard", h.Driver.GetDashboard)

	// ---- ADMIN ----
	admin := api.Group("/admin", middleware.Auth(cfg), middleware.RequireRole("admin"))
	admin.Get("/dashboard", h.Admin.GetDashboard)
	admin.Get("/users", h.Admin.ListUsers)
	admin.Get("/users/:id", h.Admin.GetUser)
	admin.Put("/users/:id/status", h.Admin.UpdateUserStatus)
	admin.Get("/restaurants", h.Admin.ListRestaurants)
	admin.Put("/restaurants/:id/status", h.Admin.UpdateRestaurantStatus)
	admin.Get("/orders", h.Admin.ListOrders)
	admin.Get("/orders/:id", h.Admin.GetOrder)
	admin.Get("/drivers", h.Admin.ListDrivers)
	admin.Put("/drivers/:id/approve", h.Admin.ApproveDriver)
	admin.Get("/analytics", h.Admin.GetAnalytics)
	admin.Get("/analytics/revenue", h.Admin.GetRevenueAnalytics)
	admin.Post("/promotions", h.Admin.CreatePromotion)
	admin.Get("/promotions", h.Admin.ListPromotions)
	admin.Put("/promotions/:id", h.Admin.UpdatePromotion)
	admin.Delete("/promotions/:id", h.Admin.DeletePromotion)
	admin.Get("/categories", h.Admin.ListCategories)
	admin.Post("/categories", h.Admin.CreateCategory)
	admin.Put("/categories/:id", h.Admin.UpdateCategory)
	admin.Delete("/categories/:id", h.Admin.DeleteCategory)

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		port := cfg.Port
		if port == "" {
			port = "8080"
		}
		log.Printf("🚀 Server running on :%s", port)
		if err := app.Listen(":" + port); err != nil {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-quit
	log.Println("Shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := app.ShutdownWithContext(ctx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}
	log.Println("Server exited gracefully")
}
