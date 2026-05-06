package handlers

import (
	"github.com/deliveroo-clone/internal/config"
	"github.com/deliveroo-clone/internal/websocket"
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

// BaseHandler holds shared dependencies
type BaseHandler struct {
	db       *pgxpool.Pool
	rdb      *redis.Client
	hub      *websocket.Hub
	cfg      *config.Config
	validate *validator.Validate
}

// Handler aggregates all domain handlers
type Handler struct {
	Auth     *AuthHandler
	Customer *CustomerHandler
	Restaurant *RestaurantHandler
	Owner    *OwnerHandler
	Order    *OrderHandler
	Payment  *PaymentHandler
	Driver   *DriverHandler
	Admin    *AdminHandler
}

func NewHandler(db *pgxpool.Pool, rdb *redis.Client, hub *websocket.Hub, cfg *config.Config) *Handler {
	base := &BaseHandler{
		db:       db,
		rdb:      rdb,
		hub:      hub,
		cfg:      cfg,
		validate: validator.New(),
	}
	return &Handler{
		Auth:       &AuthHandler{base},
		Customer:   &CustomerHandler{base},
		Restaurant: &RestaurantHandler{base},
		Owner:      &OwnerHandler{base},
		Order:      &OrderHandler{base},
		Payment:    &PaymentHandler{base},
		Driver:     &DriverHandler{base},
		Admin:      &AdminHandler{base},
	}
}
