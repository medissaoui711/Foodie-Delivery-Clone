package models

import (
	"time"
)

// ============================================
// USER MODELS
// ============================================

type UserRole string
type AuthProvider string

const (
	RoleCustomer    UserRole = "customer"
	RoleOwner       UserRole = "restaurant_owner"
	RoleDriver      UserRole = "driver"
	RoleAdmin       UserRole = "admin"
)

type User struct {
	ID                string      `json:"id" db:"id"`
	Email             string      `json:"email" db:"email"`
	Phone             *string     `json:"phone,omitempty" db:"phone"`
	PasswordHash      string      `json:"-" db:"password_hash"`
	Role              UserRole    `json:"role" db:"role"`
	AuthProvider      AuthProvider `json:"auth_provider" db:"auth_provider"`
	FirstName         string      `json:"first_name" db:"first_name"`
	LastName          string      `json:"last_name" db:"last_name"`
	AvatarURL         *string     `json:"avatar_url,omitempty" db:"avatar_url"`
	IsEmailVerified   bool        `json:"is_email_verified" db:"is_email_verified"`
	IsPhoneVerified   bool        `json:"is_phone_verified" db:"is_phone_verified"`
	IsActive          bool        `json:"is_active" db:"is_active"`
	FCMToken          *string     `json:"-" db:"fcm_token"`
	LastLoginAt       *time.Time  `json:"last_login_at,omitempty" db:"last_login_at"`
	CreatedAt         time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at" db:"updated_at"`
}

type UserPublic struct {
	ID        string  `json:"id"`
	FirstName string  `json:"first_name"`
	LastName  string  `json:"last_name"`
	AvatarURL *string `json:"avatar_url,omitempty"`
	Rating    float64 `json:"rating,omitempty"`
}

// ============================================
// AUTH MODELS
// ============================================

type RegisterRequest struct {
	Email     string   `json:"email" validate:"required,email"`
	Password  string   `json:"password" validate:"required,min=8"`
	FirstName string   `json:"first_name" validate:"required,min=2"`
	LastName  string   `json:"last_name" validate:"required,min=2"`
	Phone     string   `json:"phone" validate:"omitempty,e164"`
	Role      UserRole `json:"role" validate:"omitempty,oneof=customer restaurant_owner driver"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type AuthResponse struct {
	User         *User  `json:"user"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// ============================================
// RESTAURANT MODELS
// ============================================

type RestaurantStatus string

const (
	StatusPending   RestaurantStatus = "pending"
	StatusActive    RestaurantStatus = "active"
	StatusSuspended RestaurantStatus = "suspended"
	StatusClosed    RestaurantStatus = "closed"
)

type Restaurant struct {
	ID                        string           `json:"id" db:"id"`
	OwnerID                   string           `json:"owner_id" db:"owner_id"`
	Name                      string           `json:"name" db:"name"`
	Slug                      string           `json:"slug" db:"slug"`
	Description               *string          `json:"description,omitempty" db:"description"`
	LogoURL                   *string          `json:"logo_url,omitempty" db:"logo_url"`
	CoverImageURL             *string          `json:"cover_image_url,omitempty" db:"cover_image_url"`
	Phone                     *string          `json:"phone,omitempty" db:"phone"`
	Email                     *string          `json:"email,omitempty" db:"email"`
	Status                    RestaurantStatus `json:"status" db:"status"`
	AddressLine1              string           `json:"address_line1" db:"address_line1"`
	AddressLine2              *string          `json:"address_line2,omitempty" db:"address_line2"`
	City                      string           `json:"city" db:"city"`
	PostalCode                string           `json:"postal_code" db:"postal_code"`
	Country                   string           `json:"country" db:"country"`
	Latitude                  float64          `json:"latitude" db:"latitude"`
	Longitude                 float64          `json:"longitude" db:"longitude"`
	DeliveryRadiusKm          float64          `json:"delivery_radius_km" db:"delivery_radius_km"`
	MinOrderAmount            float64          `json:"min_order_amount" db:"min_order_amount"`
	DeliveryFee               float64          `json:"delivery_fee" db:"delivery_fee"`
	EstimatedDeliveryTimeMin  int              `json:"estimated_delivery_time_min" db:"estimated_delivery_time_min"`
	EstimatedDeliveryTimeMax  int              `json:"estimated_delivery_time_max" db:"estimated_delivery_time_max"`
	Rating                    float64          `json:"rating" db:"rating"`
	TotalReviews              int              `json:"total_reviews" db:"total_reviews"`
	TotalOrders               int              `json:"total_orders" db:"total_orders"`
	CommissionRate            float64          `json:"commission_rate" db:"commission_rate"`
	IsFeatured                bool             `json:"is_featured" db:"is_featured"`
	IsOpen                    bool             `json:"is_open" db:"is_open"`
	Tags                      []string         `json:"tags" db:"tags"`
	Categories                []Category       `json:"categories,omitempty"`
	Hours                     []RestaurantHour `json:"hours,omitempty"`
	DistanceKm                float64          `json:"distance_km,omitempty"`
	CreatedAt                 time.Time        `json:"created_at" db:"created_at"`
	UpdatedAt                 time.Time        `json:"updated_at" db:"updated_at"`
}

type Category struct {
	ID       string `json:"id" db:"id"`
	Name     string `json:"name" db:"name"`
	Slug     string `json:"slug" db:"slug"`
	IconURL  string `json:"icon_url" db:"icon_url"`
	SortOrder int   `json:"sort_order" db:"sort_order"`
}

type RestaurantHour struct {
	ID           string `json:"id" db:"id"`
	RestaurantID string `json:"restaurant_id" db:"restaurant_id"`
	DayOfWeek    int    `json:"day_of_week" db:"day_of_week"`
	OpenTime     string `json:"open_time" db:"open_time"`
	CloseTime    string `json:"close_time" db:"close_time"`
	IsClosed     bool   `json:"is_closed" db:"is_closed"`
}

type CreateRestaurantRequest struct {
	Name             string   `json:"name" validate:"required,min=2"`
	Description      string   `json:"description"`
	Phone            string   `json:"phone" validate:"required"`
	Email            string   `json:"email" validate:"required,email"`
	AddressLine1     string   `json:"address_line1" validate:"required"`
	AddressLine2     string   `json:"address_line2"`
	City             string   `json:"city" validate:"required"`
	PostalCode       string   `json:"postal_code" validate:"required"`
	Latitude         float64  `json:"latitude" validate:"required"`
	Longitude        float64  `json:"longitude" validate:"required"`
	DeliveryRadiusKm float64  `json:"delivery_radius_km"`
	MinOrderAmount   float64  `json:"min_order_amount"`
	DeliveryFee      float64  `json:"delivery_fee"`
	CategoryIDs      []string `json:"category_ids"`
	Tags             []string `json:"tags"`
}

// ============================================
// MENU MODELS
// ============================================

type MenuSection struct {
	ID             string     `json:"id" db:"id"`
	RestaurantID   string     `json:"restaurant_id" db:"restaurant_id"`
	Name           string     `json:"name" db:"name"`
	Description    *string    `json:"description,omitempty" db:"description"`
	SortOrder      int        `json:"sort_order" db:"sort_order"`
	IsActive       bool       `json:"is_active" db:"is_active"`
	Items          []MenuItem `json:"items,omitempty"`
}

type MenuItem struct {
	ID                string           `json:"id" db:"id"`
	RestaurantID      string           `json:"restaurant_id" db:"restaurant_id"`
	SectionID         *string          `json:"section_id,omitempty" db:"section_id"`
	Name              string           `json:"name" db:"name"`
	Description       *string          `json:"description,omitempty" db:"description"`
	Price             float64          `json:"price" db:"price"`
	ImageURL          *string          `json:"image_url,omitempty" db:"image_url"`
	Status            string           `json:"status" db:"status"`
	IsFeatured        bool             `json:"is_featured" db:"is_featured"`
	Calories          *int             `json:"calories,omitempty" db:"calories"`
	Allergens         []string         `json:"allergens,omitempty" db:"allergens"`
	DietaryTags       []string         `json:"dietary_tags,omitempty" db:"dietary_tags"`
	PreparationTime   int              `json:"preparation_time_min" db:"preparation_time_min"`
	SortOrder         int              `json:"sort_order" db:"sort_order"`
	TotalOrders       int              `json:"total_orders" db:"total_orders"`
	Rating            float64          `json:"rating" db:"rating"`
	OptionGroups      []OptionGroup    `json:"option_groups,omitempty"`
	CreatedAt         time.Time        `json:"created_at" db:"created_at"`
}

type OptionGroup struct {
	ID            string   `json:"id" db:"id"`
	ItemID        string   `json:"item_id" db:"item_id"`
	Name          string   `json:"name" db:"name"`
	Description   *string  `json:"description,omitempty" db:"description"`
	MinSelections int      `json:"min_selections" db:"min_selections"`
	MaxSelections int      `json:"max_selections" db:"max_selections"`
	IsRequired    bool     `json:"is_required" db:"is_required"`
	Options       []Option `json:"options,omitempty"`
}

type Option struct {
	ID            string  `json:"id" db:"id"`
	GroupID       string  `json:"group_id" db:"group_id"`
	Name          string  `json:"name" db:"name"`
	PriceAddition float64 `json:"price_addition" db:"price_addition"`
	IsDefault     bool    `json:"is_default" db:"is_default"`
	IsAvailable   bool    `json:"is_available" db:"is_available"`
}

// ============================================
// ORDER MODELS
// ============================================

type OrderStatus string

const (
	OrderPending        OrderStatus = "pending"
	OrderConfirmed      OrderStatus = "confirmed"
	OrderPreparing      OrderStatus = "preparing"
	OrderReadyPickup    OrderStatus = "ready_for_pickup"
	OrderDriverAssigned OrderStatus = "driver_assigned"
	OrderPickedUp       OrderStatus = "picked_up"
	OrderOnTheWay       OrderStatus = "on_the_way"
	OrderDelivered      OrderStatus = "delivered"
	OrderCancelled      OrderStatus = "cancelled"
	OrderRefunded       OrderStatus = "refunded"
)

type Order struct {
	ID                    string       `json:"id" db:"id"`
	OrderNumber           string       `json:"order_number" db:"order_number"`
	CustomerID            string       `json:"customer_id" db:"customer_id"`
	RestaurantID          string       `json:"restaurant_id" db:"restaurant_id"`
	DriverID              *string      `json:"driver_id,omitempty" db:"driver_id"`
	Status                OrderStatus  `json:"status" db:"status"`
	PaymentMethod         string       `json:"payment_method" db:"payment_method"`
	PaymentStatus         string       `json:"payment_status" db:"payment_status"`
	StripePaymentIntentID *string      `json:"-" db:"stripe_payment_intent_id"`
	Subtotal              float64      `json:"subtotal" db:"subtotal"`
	DeliveryFee           float64      `json:"delivery_fee" db:"delivery_fee"`
	ServiceFee            float64      `json:"service_fee" db:"service_fee"`
	DiscountAmount        float64      `json:"discount_amount" db:"discount_amount"`
	TipAmount             float64      `json:"tip_amount" db:"tip_amount"`
	TotalAmount           float64      `json:"total_amount" db:"total_amount"`
	PromoCode             *string      `json:"promo_code,omitempty" db:"promo_code"`
	DeliveryAddressID     *string      `json:"delivery_address_id,omitempty" db:"delivery_address_id"`
	DeliveryAddressSnapshot interface{} `json:"delivery_address" db:"delivery_address_snapshot"`
	DeliveryInstructions  *string      `json:"delivery_instructions,omitempty" db:"delivery_instructions"`
	EstimatedDeliveryAt   *time.Time   `json:"estimated_delivery_at,omitempty" db:"estimated_delivery_at"`
	ActualDeliveryAt      *time.Time   `json:"actual_delivery_at,omitempty" db:"actual_delivery_at"`
	CancelledAt           *time.Time   `json:"cancelled_at,omitempty" db:"cancelled_at"`
	CancellationReason    *string      `json:"cancellation_reason,omitempty" db:"cancellation_reason"`
	RestaurantNote        *string      `json:"restaurant_note,omitempty" db:"restaurant_note"`
	Items                 []OrderItem  `json:"items,omitempty"`
	Restaurant            *Restaurant  `json:"restaurant,omitempty"`
	Customer              *UserPublic  `json:"customer,omitempty"`
	Driver                *UserPublic  `json:"driver,omitempty"`
	CreatedAt             time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt             time.Time    `json:"updated_at" db:"updated_at"`
}

type OrderItem struct {
	ID                   string      `json:"id" db:"id"`
	OrderID              string      `json:"order_id" db:"order_id"`
	MenuItemID           *string     `json:"menu_item_id,omitempty" db:"menu_item_id"`
	Name                 string      `json:"name" db:"name"`
	Description          *string     `json:"description,omitempty" db:"description"`
	ImageURL             *string     `json:"image_url,omitempty" db:"image_url"`
	UnitPrice            float64     `json:"unit_price" db:"unit_price"`
	Quantity             int         `json:"quantity" db:"quantity"`
	SelectedOptions      interface{} `json:"selected_options,omitempty" db:"selected_options"`
	OptionsPrice         float64     `json:"options_price" db:"options_price"`
	Subtotal             float64     `json:"subtotal" db:"subtotal"`
	SpecialInstructions  *string     `json:"special_instructions,omitempty" db:"special_instructions"`
}

type CreateOrderRequest struct {
	RestaurantID         string            `json:"restaurant_id" validate:"required,uuid"`
	Items                []CreateOrderItem `json:"items" validate:"required,min=1"`
	DeliveryAddressID    string            `json:"delivery_address_id" validate:"required,uuid"`
	PaymentMethod        string            `json:"payment_method" validate:"required,oneof=card cash wallet"`
	PaymentCardID        string            `json:"payment_card_id"`
	PromoCode            string            `json:"promo_code"`
	TipAmount            float64           `json:"tip_amount"`
	DeliveryInstructions string            `json:"delivery_instructions"`
	RestaurantNote       string            `json:"restaurant_note"`
}

type CreateOrderItem struct {
	MenuItemID          string        `json:"menu_item_id" validate:"required,uuid"`
	Quantity            int           `json:"quantity" validate:"required,min=1"`
	SelectedOptions     []string      `json:"selected_options"`
	SpecialInstructions string        `json:"special_instructions"`
}

// ============================================
// DRIVER MODELS
// ============================================

type DriverStatus string
type VehicleType string

const (
	DriverOffline DriverStatus = "offline"
	DriverOnline  DriverStatus = "online"
	DriverBusy    DriverStatus = "busy"
)

type DriverProfile struct {
	ID                  string       `json:"id" db:"id"`
	UserID              string       `json:"user_id" db:"user_id"`
	Status              DriverStatus `json:"status" db:"status"`
	VehicleType         VehicleType  `json:"vehicle_type" db:"vehicle_type"`
	VehicleMake         *string      `json:"vehicle_make,omitempty" db:"vehicle_make"`
	VehicleModel        *string      `json:"vehicle_model,omitempty" db:"vehicle_model"`
	VehicleYear         *int         `json:"vehicle_year,omitempty" db:"vehicle_year"`
	VehiclePlate        *string      `json:"vehicle_plate,omitempty" db:"vehicle_plate"`
	LicenseNumber       *string      `json:"license_number,omitempty" db:"license_number"`
	Rating              float64      `json:"rating" db:"rating"`
	TotalDeliveries     int          `json:"total_deliveries" db:"total_deliveries"`
	TotalEarnings       float64      `json:"total_earnings" db:"total_earnings"`
	CurrentLatitude     *float64     `json:"current_latitude,omitempty" db:"current_latitude"`
	CurrentLongitude    *float64     `json:"current_longitude,omitempty" db:"current_longitude"`
	LastLocationUpdate  *time.Time   `json:"last_location_update,omitempty" db:"last_location_update"`
	IsApproved          bool         `json:"is_approved" db:"is_approved"`
	CreatedAt           time.Time    `json:"created_at" db:"created_at"`
	UpdatedAt           time.Time    `json:"updated_at" db:"updated_at"`
}

type UpdateLocationRequest struct {
	Latitude  float64  `json:"latitude" validate:"required"`
	Longitude float64  `json:"longitude" validate:"required"`
	Accuracy  float64  `json:"accuracy"`
	Speed     float64  `json:"speed"`
	Heading   float64  `json:"heading"`
	OrderID   *string  `json:"order_id"`
}

// ============================================
// PAGINATION
// ============================================

type PaginationQuery struct {
	Page     int    `query:"page"`
	Limit    int    `query:"limit"`
	SortBy   string `query:"sort_by"`
	SortDir  string `query:"sort_dir"`
}

type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	Limit      int         `json:"limit"`
	TotalPages int         `json:"total_pages"`
}

// ============================================
// ANALYTICS
// ============================================

type DashboardStats struct {
	TotalRevenue     float64 `json:"total_revenue"`
	TotalOrders      int64   `json:"total_orders"`
	ActiveRestaurants int64  `json:"active_restaurants"`
	ActiveDrivers    int64   `json:"active_drivers"`
	TotalCustomers   int64   `json:"total_customers"`
	OrdersToday      int64   `json:"orders_today"`
	RevenueToday     float64 `json:"revenue_today"`
	AvgOrderValue    float64 `json:"avg_order_value"`
	AvgDeliveryTime  float64 `json:"avg_delivery_time_min"`
}
