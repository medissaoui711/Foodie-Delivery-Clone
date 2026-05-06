package handlers

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/deliveroo-clone/internal/models"
	"github.com/deliveroo-clone/internal/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
)

// ============================================
// CUSTOMER HANDLER
// ============================================

type CustomerHandler struct{ *BaseHandler }

func (h *CustomerHandler) GetProfile(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	ctx := context.Background()
	var user models.User
	h.db.QueryRow(ctx,
		`SELECT id, email, phone, role, first_name, last_name, avatar_url, is_email_verified, created_at
		 FROM users WHERE id = $1`, userID).
		Scan(&user.ID, &user.Email, &user.Phone, &user.Role, &user.FirstName, &user.LastName,
			&user.AvatarURL, &user.IsEmailVerified, &user.CreatedAt)
	return c.JSON(fiber.Map{"success": true, "data": user})
}

func (h *CustomerHandler) UpdateProfile(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	var body struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		Phone     string `json:"phone"`
	}
	c.BodyParser(&body)
	ctx := context.Background()
	h.db.Exec(ctx,
		`UPDATE users SET
		 first_name = COALESCE(NULLIF($1,''), first_name),
		 last_name  = COALESCE(NULLIF($2,''), last_name),
		 phone      = COALESCE(NULLIF($3,''), phone),
		 updated_at = NOW()
		 WHERE id = $4`,
		body.FirstName, body.LastName, body.Phone, userID)
	return c.JSON(fiber.Map{"success": true, "message": "profile updated"})
}

func (h *CustomerHandler) ListAddresses(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	ctx := context.Background()
	rows, _ := h.db.Query(ctx,
		`SELECT id, label, line1, line2, city, postal_code, country,
		 latitude, longitude, is_default, instructions
		 FROM addresses WHERE user_id = $1 ORDER BY is_default DESC, created_at DESC`, userID)
	if rows != nil {
		defer rows.Close()
	}
	var addrs []fiber.Map
	for rows.Next() {
		var id, label, line1, city, postalCode, country string
		var line2, instructions *string
		var lat, lng *float64
		var isDef bool
		rows.Scan(&id, &label, &line1, &line2, &city, &postalCode, &country, &lat, &lng, &isDef, &instructions)
		addrs = append(addrs, fiber.Map{
			"id": id, "label": label, "line1": line1, "line2": line2,
			"city": city, "postal_code": postalCode, "country": country,
			"latitude": lat, "longitude": lng, "is_default": isDef, "instructions": instructions,
		})
	}
	return c.JSON(fiber.Map{"success": true, "data": addrs})
}

func (h *CustomerHandler) CreateAddress(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	var body struct {
		Label        string  `json:"label"`
		Line1        string  `json:"line1"`
		Line2        string  `json:"line2"`
		City         string  `json:"city"`
		PostalCode   string  `json:"postal_code"`
		Country      string  `json:"country"`
		Latitude     float64 `json:"latitude"`
		Longitude    float64 `json:"longitude"`
		IsDefault    bool    `json:"is_default"`
		Instructions string  `json:"instructions"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	ctx := context.Background()
	if body.IsDefault {
		h.db.Exec(ctx, "UPDATE addresses SET is_default = false WHERE user_id = $1", userID)
	}
	if body.Country == "" {
		body.Country = "GB"
	}
	if body.Label == "" {
		body.Label = "Home"
	}
	var id string
	h.db.QueryRow(ctx,
		`INSERT INTO addresses (user_id, label, line1, line2, city, postal_code, country, latitude, longitude, is_default, instructions)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id`,
		userID, body.Label, body.Line1, nullString(body.Line2), body.City, body.PostalCode,
		body.Country, body.Latitude, body.Longitude, body.IsDefault, nullString(body.Instructions)).Scan(&id)
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": fiber.Map{"id": id}})
}

func (h *CustomerHandler) UpdateAddress(c *fiber.Ctx) error {
	addrID := c.Params("id")
	userID := c.Locals("user_id").(string)
	var body struct {
		Label     string `json:"label"`
		Line1     string `json:"line1"`
		City      string `json:"city"`
		IsDefault bool   `json:"is_default"`
	}
	c.BodyParser(&body)
	ctx := context.Background()
	if body.IsDefault {
		h.db.Exec(ctx, "UPDATE addresses SET is_default = false WHERE user_id = $1", userID)
	}
	h.db.Exec(ctx,
		`UPDATE addresses SET
		 label = COALESCE(NULLIF($1,''), label),
		 line1 = COALESCE(NULLIF($2,''), line1),
		 city  = COALESCE(NULLIF($3,''), city),
		 is_default = $4
		 WHERE id = $5 AND user_id = $6`,
		body.Label, body.Line1, body.City, body.IsDefault, addrID, userID)
	return c.JSON(fiber.Map{"success": true, "message": "address updated"})
}

func (h *CustomerHandler) DeleteAddress(c *fiber.Ctx) error {
	h.db.Exec(context.Background(), "DELETE FROM addresses WHERE id = $1 AND user_id = $2",
		c.Params("id"), c.Locals("user_id"))
	return c.JSON(fiber.Map{"success": true, "message": "address deleted"})
}

func (h *CustomerHandler) ListOrders(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	ctx := context.Background()
	rows, _ := h.db.Query(ctx,
		`SELECT o.id, o.order_number, o.status, o.total_amount, o.created_at, r.name, r.logo_url
		 FROM orders o JOIN restaurants r ON r.id = o.restaurant_id
		 WHERE o.customer_id = $1 ORDER BY o.created_at DESC LIMIT 50`, userID)
	if rows != nil {
		defer rows.Close()
	}
	var orders []fiber.Map
	for rows.Next() {
		var id, num, status, rName string
		var total float64
		var createdAt time.Time
		var logoURL *string
		rows.Scan(&id, &num, &status, &total, &createdAt, &rName, &logoURL)
		orders = append(orders, fiber.Map{
			"id": id, "order_number": num, "status": status,
			"total_amount": total, "created_at": createdAt,
			"restaurant_name": rName, "restaurant_logo": logoURL,
		})
	}
	return c.JSON(fiber.Map{"success": true, "data": orders})
}

func (h *CustomerHandler) GetOrder(c *fiber.Ctx) error {
	return (&OrderHandler{h.BaseHandler}).GetByID(c)
}

func (h *CustomerHandler) ReviewOrder(c *fiber.Ctx) error {
	orderID := c.Params("id")
	customerID := c.Locals("user_id").(string)
	var body struct {
		FoodRating     int    `json:"food_rating"`
		DeliveryRating int    `json:"delivery_rating"`
		OverallRating  int    `json:"overall_rating"`
		Comment        string `json:"comment"`
		IsAnonymous    bool   `json:"is_anonymous"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	ctx := context.Background()
	var restaurantID string
	var driverID *string
	h.db.QueryRow(ctx,
		"SELECT restaurant_id, driver_id FROM orders WHERE id = $1 AND customer_id = $2 AND status = 'delivered'",
		orderID, customerID).Scan(&restaurantID, &driverID)
	if restaurantID == "" {
		return fiber.NewError(fiber.StatusBadRequest, "order not eligible for review")
	}
	h.db.Exec(ctx,
		`INSERT INTO reviews (order_id, customer_id, restaurant_id, driver_id, food_rating, delivery_rating, overall_rating, comment, is_anonymous)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT (order_id) DO NOTHING`,
		orderID, customerID, restaurantID, driverID, body.FoodRating, body.DeliveryRating,
		body.OverallRating, nullString(body.Comment), body.IsAnonymous)
	h.db.Exec(ctx,
		`UPDATE restaurants SET
		 rating = (SELECT AVG(overall_rating) FROM reviews WHERE restaurant_id = $1),
		 total_reviews = total_reviews + 1
		 WHERE id = $1`, restaurantID)
	return c.JSON(fiber.Map{"success": true, "message": "review submitted"})
}

func (h *CustomerHandler) GetWallet(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	ctx := context.Background()
	var balance float64
	var currency string
	h.db.QueryRow(ctx, "SELECT balance, currency FROM wallets WHERE user_id = $1", userID).Scan(&balance, &currency)
	return c.JSON(fiber.Map{"success": true, "data": fiber.Map{"balance": balance, "currency": currency}})
}

func (h *CustomerHandler) ListWalletTransactions(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	ctx := context.Background()
	rows, _ := h.db.Query(ctx,
		`SELECT wt.id, wt.type, wt.amount, wt.balance_after, wt.description, wt.created_at
		 FROM wallet_transactions wt JOIN wallets w ON w.id = wt.wallet_id
		 WHERE w.user_id = $1 ORDER BY wt.created_at DESC LIMIT 50`, userID)
	if rows != nil {
		defer rows.Close()
	}
	var txns []fiber.Map
	for rows.Next() {
		var id, txType string
		var amount, balAfter float64
		var desc *string
		var createdAt time.Time
		rows.Scan(&id, &txType, &amount, &balAfter, &desc, &createdAt)
		txns = append(txns, fiber.Map{
			"id": id, "type": txType, "amount": amount,
			"balance_after": balAfter, "description": desc, "created_at": createdAt,
		})
	}
	return c.JSON(fiber.Map{"success": true, "data": txns})
}

func (h *CustomerHandler) ListNotifications(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	ctx := context.Background()
	rows, _ := h.db.Query(ctx,
		`SELECT id, type, title, body, data, is_read, created_at
		 FROM notifications WHERE user_id = $1 ORDER BY created_at DESC LIMIT 50`, userID)
	if rows != nil {
		defer rows.Close()
	}
	var notifs []fiber.Map
	for rows.Next() {
		var id, nType, title, body string
		var data *string
		var isRead bool
		var createdAt time.Time
		rows.Scan(&id, &nType, &title, &body, &data, &isRead, &createdAt)
		notifs = append(notifs, fiber.Map{
			"id": id, "type": nType, "title": title, "body": body,
			"data": data, "is_read": isRead, "created_at": createdAt,
		})
	}
	return c.JSON(fiber.Map{"success": true, "data": notifs})
}

func (h *CustomerHandler) MarkNotificationRead(c *fiber.Ctx) error {
	h.db.Exec(context.Background(),
		"UPDATE notifications SET is_read = true, read_at = NOW() WHERE id = $1 AND user_id = $2",
		c.Params("id"), c.Locals("user_id"))
	return c.JSON(fiber.Map{"success": true})
}

func (h *CustomerHandler) MarkAllNotificationsRead(c *fiber.Ctx) error {
	h.db.Exec(context.Background(),
		"UPDATE notifications SET is_read = true, read_at = NOW() WHERE user_id = $1 AND is_read = false",
		c.Locals("user_id"))
	return c.JSON(fiber.Map{"success": true})
}

func (h *CustomerHandler) ListPaymentCards(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	ctx := context.Background()
	rows, _ := h.db.Query(ctx,
		"SELECT id, last_four, brand, exp_month, exp_year, is_default FROM payment_cards WHERE user_id = $1 ORDER BY is_default DESC",
		userID)
	if rows != nil {
		defer rows.Close()
	}
	var cards []fiber.Map
	for rows.Next() {
		var id, lastFour, brand string
		var expMonth, expYear int
		var isDef bool
		rows.Scan(&id, &lastFour, &brand, &expMonth, &expYear, &isDef)
		cards = append(cards, fiber.Map{
			"id": id, "last_four": lastFour, "brand": brand,
			"exp_month": expMonth, "exp_year": expYear, "is_default": isDef,
		})
	}
	return c.JSON(fiber.Map{"success": true, "data": cards})
}

func (h *CustomerHandler) AddPaymentCard(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "message": "use Stripe Elements on frontend to tokenize card"})
}

func (h *CustomerHandler) DeletePaymentCard(c *fiber.Ctx) error {
	h.db.Exec(context.Background(), "DELETE FROM payment_cards WHERE id = $1 AND user_id = $2",
		c.Params("id"), c.Locals("user_id"))
	return c.JSON(fiber.Map{"success": true})
}

func (h *CustomerHandler) SetDefaultCard(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	ctx := context.Background()
	h.db.Exec(ctx, "UPDATE payment_cards SET is_default = false WHERE user_id = $1", userID)
	h.db.Exec(ctx, "UPDATE payment_cards SET is_default = true WHERE id = $1 AND user_id = $2", c.Params("id"), userID)
	return c.JSON(fiber.Map{"success": true})
}

// ============================================
// DRIVER HANDLER
// ============================================

type DriverHandler struct{ *BaseHandler }

func (h *DriverHandler) GetProfile(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	ctx := context.Background()
	var dp models.DriverProfile
	h.db.QueryRow(ctx,
		`SELECT id, user_id, status, vehicle_type, vehicle_make, vehicle_model, vehicle_plate,
		 rating, total_deliveries, total_earnings, current_latitude, current_longitude, is_approved, created_at
		 FROM driver_profiles WHERE user_id = $1`, userID).
		Scan(&dp.ID, &dp.UserID, &dp.Status, &dp.VehicleType, &dp.VehicleMake, &dp.VehicleModel,
			&dp.VehiclePlate, &dp.Rating, &dp.TotalDeliveries, &dp.TotalEarnings,
			&dp.CurrentLatitude, &dp.CurrentLongitude, &dp.IsApproved, &dp.CreatedAt)
	return c.JSON(fiber.Map{"success": true, "data": dp})
}

func (h *DriverHandler) CreateProfile(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	var body struct {
		VehicleType   string `json:"vehicle_type"`
		VehicleMake   string `json:"vehicle_make"`
		VehicleModel  string `json:"vehicle_model"`
		VehiclePlate  string `json:"vehicle_plate"`
		LicenseNumber string `json:"license_number"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	ctx := context.Background()
	var id string
	h.db.QueryRow(ctx,
		`INSERT INTO driver_profiles (user_id, vehicle_type, vehicle_make, vehicle_model, vehicle_plate, license_number)
		 VALUES ($1,$2,$3,$4,$5,$6)
		 ON CONFLICT (user_id) DO UPDATE SET vehicle_type=$2, vehicle_make=$3, vehicle_model=$4
		 RETURNING id`,
		userID, body.VehicleType, body.VehicleMake, body.VehicleModel, body.VehiclePlate, body.LicenseNumber).Scan(&id)
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": fiber.Map{"id": id}})
}

func (h *DriverHandler) UpdateProfile(c *fiber.Ctx) error { return h.CreateProfile(c) }

func (h *DriverHandler) ToggleStatus(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	ctx := context.Background()
	var current string
	h.db.QueryRow(ctx,
		"SELECT status FROM driver_profiles WHERE user_id = $1 AND is_approved = true", userID).Scan(&current)
	if current == "" {
		return fiber.NewError(fiber.StatusForbidden, "driver profile not approved yet")
	}
	newStatus := "online"
	if current == "online" {
		newStatus = "offline"
	}
	h.db.Exec(ctx, "UPDATE driver_profiles SET status = $1 WHERE user_id = $2", newStatus, userID)
	return c.JSON(fiber.Map{"success": true, "data": fiber.Map{"status": newStatus}})
}

func (h *DriverHandler) UpdateLocation(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	var req models.UpdateLocationRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	ctx := context.Background()
	h.db.Exec(ctx,
		`UPDATE driver_profiles SET current_latitude=$1, current_longitude=$2, last_location_update=NOW()
		 WHERE user_id=$3`, req.Latitude, req.Longitude, userID)
	if req.OrderID != nil {
		h.hub.UpdateDriverLocation(*req.OrderID, req.Latitude, req.Longitude)
		h.db.Exec(ctx,
			`INSERT INTO driver_location_history (driver_id, latitude, longitude, accuracy, speed, heading, order_id)
			 SELECT id, $1, $2, $3, $4, $5, $6 FROM driver_profiles WHERE user_id = $7`,
			req.Latitude, req.Longitude, req.Accuracy, req.Speed, req.Heading, req.OrderID, userID)
	}
	return c.JSON(fiber.Map{"success": true})
}

func (h *DriverHandler) GetAvailableOrders(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	ctx := context.Background()
	var lat, lng float64
	h.db.QueryRow(ctx,
		"SELECT COALESCE(current_latitude,0), COALESCE(current_longitude,0) FROM driver_profiles WHERE user_id = $1",
		userID).Scan(&lat, &lng)
	rows, _ := h.db.Query(ctx,
		`SELECT o.id, o.order_number, o.total_amount, o.delivery_fee,
		 r.name, r.address_line1, r.city, r.latitude, r.longitude
		 FROM orders o JOIN restaurants r ON r.id = o.restaurant_id
		 WHERE o.status = 'ready_for_pickup' AND o.driver_id IS NULL
		 ORDER BY o.created_at ASC LIMIT 10`)
	if rows != nil {
		defer rows.Close()
	}
	var orders []fiber.Map
	for rows.Next() {
		var id, num, rName, addr, city string
		var total, fee, rLat, rLng float64
		rows.Scan(&id, &num, &total, &fee, &rName, &addr, &city, &rLat, &rLng)
		dist := haversine(lat, lng, rLat, rLng)
		orders = append(orders, fiber.Map{
			"id": id, "order_number": num, "total_amount": total, "delivery_fee": fee,
			"restaurant_name": rName, "restaurant_address": addr + ", " + city, "distance_km": dist,
		})
	}
	return c.JSON(fiber.Map{"success": true, "data": orders})
}

func (h *DriverHandler) GetActiveOrder(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	ctx := context.Background()
	var id, num, status string
	var total float64
	h.db.QueryRow(ctx,
		`SELECT id, order_number, status, total_amount FROM orders
		 WHERE driver_id = $1 AND status IN ('driver_assigned','picked_up','on_the_way') LIMIT 1`,
		userID).Scan(&id, &num, &status, &total)
	if id == "" {
		return c.JSON(fiber.Map{"success": true, "data": nil})
	}
	return c.JSON(fiber.Map{"success": true, "data": fiber.Map{
		"id": id, "order_number": num, "status": status, "total_amount": total,
	}})
}

func (h *DriverHandler) GetOrderHistory(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	ctx := context.Background()
	rows, _ := h.db.Query(ctx,
		`SELECT o.id, o.order_number, o.status, o.delivery_fee, o.created_at, r.name
		 FROM orders o JOIN restaurants r ON r.id = o.restaurant_id
		 WHERE o.driver_id = $1 AND o.status IN ('delivered','cancelled')
		 ORDER BY o.created_at DESC LIMIT 50`, userID)
	if rows != nil {
		defer rows.Close()
	}
	var orders []fiber.Map
	for rows.Next() {
		var id, num, status, rName string
		var fee float64
		var createdAt time.Time
		rows.Scan(&id, &num, &status, &fee, &createdAt, &rName)
		orders = append(orders, fiber.Map{
			"id": id, "order_number": num, "status": status,
			"delivery_fee": fee, "created_at": createdAt, "restaurant_name": rName,
		})
	}
	return c.JSON(fiber.Map{"success": true, "data": orders})
}

func (h *DriverHandler) AcceptOrder(c *fiber.Ctx) error {
	orderID := c.Params("id")
	userID := c.Locals("user_id").(string)
	ctx := context.Background()
	res, err := h.db.Exec(ctx,
		`UPDATE orders SET driver_id = $1, status = 'driver_assigned'
		 WHERE id = $2 AND status = 'ready_for_pickup' AND driver_id IS NULL`,
		userID, orderID)
	if err != nil || res.RowsAffected() == 0 {
		return fiber.NewError(fiber.StatusConflict, "order no longer available")
	}
	h.db.Exec(ctx, "UPDATE driver_profiles SET status = 'busy' WHERE user_id = $1", userID)
	h.db.Exec(ctx, "INSERT INTO order_status_history (order_id, status, changed_by) VALUES ($1,'driver_assigned',$2)", orderID, userID)
	h.hub.BroadcastToRoom("order:"+orderID, websocket.MsgOrderAssigned, fiber.Map{
		"order_id": orderID, "status": "driver_assigned",
	})
	return c.JSON(fiber.Map{"success": true, "message": "order accepted"})
}

func (h *DriverHandler) RejectOrder(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "message": "order rejected"})
}

func (h *DriverHandler) UpdateOrderStatus(c *fiber.Ctx) error {
	orderID := c.Params("id")
	driverID := c.Locals("user_id").(string)
	var body struct {
		Status string `json:"status"`
	}
	c.BodyParser(&body)
	ctx := context.Background()

	allowed := map[string]bool{"picked_up": true, "on_the_way": true, "delivered": true}
	if !allowed[body.Status] {
		return fiber.NewError(fiber.StatusBadRequest, "invalid status")
	}
	h.db.Exec(ctx, "UPDATE orders SET status = $1, updated_at = NOW() WHERE id = $2 AND driver_id = $3",
		body.Status, orderID, driverID)

	if body.Status == "delivered" {
		h.db.Exec(ctx, "UPDATE orders SET actual_delivery_at = NOW() WHERE id = $1", orderID)
		h.db.Exec(ctx, "UPDATE driver_profiles SET status='online', total_deliveries=total_deliveries+1 WHERE user_id=$1", driverID)
		var deliveryFee float64
		h.db.QueryRow(ctx, "SELECT delivery_fee FROM orders WHERE id=$1", orderID).Scan(&deliveryFee)
		h.db.Exec(ctx, "UPDATE driver_profiles SET total_earnings=total_earnings+$1 WHERE user_id=$2", deliveryFee*0.85, driverID)
	}

	h.db.Exec(ctx, "INSERT INTO order_status_history (order_id, status, changed_by) VALUES ($1,$2,$3)",
		orderID, body.Status, driverID)
	h.hub.BroadcastToRoom("order:"+orderID, websocket.MsgOrderStatusUpdate, fiber.Map{
		"order_id": orderID, "status": body.Status,
	})
	return c.JSON(fiber.Map{"success": true, "data": fiber.Map{"status": body.Status}})
}

func (h *DriverHandler) GetEarnings(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	ctx := context.Background()
	var total float64
	var deliveries int
	h.db.QueryRow(ctx, "SELECT total_earnings, total_deliveries FROM driver_profiles WHERE user_id=$1",
		userID).Scan(&total, &deliveries)
	var today float64
	h.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(delivery_fee*0.85),0) FROM orders
		 WHERE driver_id=$1 AND status='delivered' AND actual_delivery_at >= CURRENT_DATE`, userID).Scan(&today)
	return c.JSON(fiber.Map{"success": true, "data": fiber.Map{
		"total_earnings": total, "today_earnings": today, "total_deliveries": deliveries,
	}})
}

func (h *DriverHandler) GetDashboard(c *fiber.Ctx) error { return h.GetEarnings(c) }

// ============================================
// OWNER HANDLER
// ============================================

type OwnerHandler struct{ *BaseHandler }

func (h *OwnerHandler) GetMyRestaurant(c *fiber.Ctx) error {
	ownerID := c.Locals("user_id").(string)
	ctx := context.Background()
	var r models.Restaurant
	err := h.db.QueryRow(ctx,
		`SELECT id, name, slug, description, logo_url, cover_image_url, phone, email, status,
		 address_line1, city, postal_code, latitude, longitude,
		 delivery_fee, min_order_amount, estimated_delivery_time_min, estimated_delivery_time_max,
		 rating, total_reviews, total_orders, is_open, tags, created_at
		 FROM restaurants WHERE owner_id = $1`, ownerID).
		Scan(&r.ID, &r.Name, &r.Slug, &r.Description, &r.LogoURL, &r.CoverImageURL,
			&r.Phone, &r.Email, &r.Status, &r.AddressLine1, &r.City, &r.PostalCode,
			&r.Latitude, &r.Longitude, &r.DeliveryFee, &r.MinOrderAmount,
			&r.EstimatedDeliveryTimeMin, &r.EstimatedDeliveryTimeMax,
			&r.Rating, &r.TotalReviews, &r.TotalOrders, &r.IsOpen, &r.Tags, &r.CreatedAt)
	if err == pgx.ErrNoRows {
		return c.JSON(fiber.Map{"success": true, "data": nil})
	}
	return c.JSON(fiber.Map{"success": true, "data": r})
}

func (h *OwnerHandler) CreateRestaurant(c *fiber.Ctx) error {
	ownerID := c.Locals("user_id").(string)
	var req models.CreateRestaurantRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	ctx := context.Background()
	slug := strings.ToLower(strings.ReplaceAll(req.Name, " ", "-")) +
		"-" + fmt.Sprintf("%d", time.Now().Unix())
	var id string
	h.db.QueryRow(ctx,
		`INSERT INTO restaurants (owner_id, name, slug, description, phone, email,
		 address_line1, city, postal_code, latitude, longitude,
		 delivery_radius_km, min_order_amount, delivery_fee, tags)
		 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15) RETURNING id`,
		ownerID, req.Name, slug, req.Description, req.Phone, req.Email,
		req.AddressLine1, req.City, req.PostalCode, req.Latitude, req.Longitude,
		req.DeliveryRadiusKm, req.MinOrderAmount, req.DeliveryFee, req.Tags).Scan(&id)
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": fiber.Map{"id": id, "slug": slug}})
}

func (h *OwnerHandler) UpdateRestaurant(c *fiber.Ctx) error {
	ownerID := c.Locals("user_id").(string)
	ctx := context.Background()
	var body map[string]interface{}
	c.BodyParser(&body)
	h.db.Exec(ctx,
		`UPDATE restaurants SET
		 description = COALESCE($1::text, description),
		 phone = COALESCE($2::text, phone),
		 delivery_fee = COALESCE($3::numeric, delivery_fee),
		 min_order_amount = COALESCE($4::numeric, min_order_amount),
		 estimated_delivery_time_min = COALESCE($5::int, estimated_delivery_time_min),
		 estimated_delivery_time_max = COALESCE($6::int, estimated_delivery_time_max),
		 updated_at = NOW()
		 WHERE owner_id = $7`,
		body["description"], body["phone"], body["delivery_fee"],
		body["min_order_amount"], body["estimated_delivery_time_min"],
		body["estimated_delivery_time_max"], ownerID)
	return c.JSON(fiber.Map{"success": true, "message": "restaurant updated"})
}

func (h *OwnerHandler) UpdateHours(c *fiber.Ctx) error {
	ownerID := c.Locals("user_id").(string)
	ctx := context.Background()
	var restaurantID string
	h.db.QueryRow(ctx, "SELECT id FROM restaurants WHERE owner_id = $1", ownerID).Scan(&restaurantID)
	if restaurantID == "" {
		return fiber.NewError(fiber.StatusNotFound, "restaurant not found")
	}
	var hours []models.RestaurantHour
	c.BodyParser(&hours)
	for _, hour := range hours {
		h.db.Exec(ctx,
			`INSERT INTO restaurant_hours (restaurant_id, day_of_week, open_time, close_time, is_closed)
			 VALUES ($1,$2,$3,$4,$5)`,
			restaurantID, hour.DayOfWeek, hour.OpenTime, hour.CloseTime, hour.IsClosed)
	}
	return c.JSON(fiber.Map{"success": true})
}

func (h *OwnerHandler) ToggleOpen(c *fiber.Ctx) error {
	ownerID := c.Locals("user_id").(string)
	ctx := context.Background()
	var isOpen bool
	h.db.QueryRow(ctx,
		"UPDATE restaurants SET is_open = NOT is_open WHERE owner_id = $1 RETURNING is_open", ownerID).Scan(&isOpen)
	return c.JSON(fiber.Map{"success": true, "data": fiber.Map{"is_open": isOpen}})
}

func (h *OwnerHandler) GetDashboard(c *fiber.Ctx) error {
	ownerID := c.Locals("user_id").(string)
	ctx := context.Background()
	var restaurantID string
	h.db.QueryRow(ctx, "SELECT id FROM restaurants WHERE owner_id = $1", ownerID).Scan(&restaurantID)
	var todayOrders int64
	var todayRevenue float64
	var totalOrders int64
	var rating float64
	var totalReviews int
	h.db.QueryRow(ctx,
		`SELECT COUNT(*), COALESCE(SUM(total_amount),0) FROM orders
		 WHERE restaurant_id=$1 AND status='delivered' AND created_at >= CURRENT_DATE`,
		restaurantID).Scan(&todayOrders, &todayRevenue)
	h.db.QueryRow(ctx,
		"SELECT total_orders, rating, total_reviews FROM restaurants WHERE id=$1",
		restaurantID).Scan(&totalOrders, &rating, &totalReviews)
	return c.JSON(fiber.Map{"success": true, "data": fiber.Map{
		"today_orders": todayOrders, "today_revenue": todayRevenue,
		"total_orders": totalOrders, "rating": rating, "total_reviews": totalReviews,
	}})
}

func (h *OwnerHandler) ListOrders(c *fiber.Ctx) error {
	ownerID := c.Locals("user_id").(string)
	ctx := context.Background()
	status := c.Query("status", "")
	var restaurantID string
	h.db.QueryRow(ctx, "SELECT id FROM restaurants WHERE owner_id=$1", ownerID).Scan(&restaurantID)
	q := `SELECT o.id, o.order_number, o.status, o.total_amount, o.payment_method, o.created_at, u.first_name, u.last_name
		  FROM orders o JOIN users u ON u.id = o.customer_id
		  WHERE o.restaurant_id = $1`
	args := []interface{}{restaurantID}
	if status != "" {
		q += " AND o.status = $2"
		args = append(args, status)
	}
	q += " ORDER BY o.created_at DESC LIMIT 100"
	rows, _ := h.db.Query(ctx, q, args...)
	if rows != nil {
		defer rows.Close()
	}
	var orders []fiber.Map
	for rows.Next() {
		var id, num, st, pm, fn, ln string
		var total float64
		var createdAt time.Time
		rows.Scan(&id, &num, &st, &total, &pm, &createdAt, &fn, &ln)
		orders = append(orders, fiber.Map{
			"id": id, "order_number": num, "status": st, "total_amount": total,
			"payment_method": pm, "created_at": createdAt, "customer_name": fn + " " + ln,
		})
	}
	return c.JSON(fiber.Map{"success": true, "data": orders})
}

func (h *OwnerHandler) GetOrder(c *fiber.Ctx) error {
	return (&OrderHandler{h.BaseHandler}).GetByID(c)
}

func (h *OwnerHandler) UpdateOrderStatus(c *fiber.Ctx) error {
	orderID := c.Params("id")
	ownerID := c.Locals("user_id").(string)
	var body struct {
		Status string `json:"status"`
	}
	c.BodyParser(&body)
	ctx := context.Background()

	allowed := map[string]bool{"confirmed": true, "preparing": true, "ready_for_pickup": true, "cancelled": true}
	if !allowed[body.Status] {
		return fiber.NewError(fiber.StatusBadRequest, "invalid status transition")
	}

	var restaurantID string
	h.db.QueryRow(ctx, "SELECT restaurant_id FROM orders WHERE id=$1", orderID).Scan(&restaurantID)
	var ownership bool
	h.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM restaurants WHERE id=$1 AND owner_id=$2)",
		restaurantID, ownerID).Scan(&ownership)
	if !ownership {
		return fiber.NewError(fiber.StatusForbidden, "access denied")
	}

	h.db.Exec(ctx, "UPDATE orders SET status=$1, updated_at=NOW() WHERE id=$2", body.Status, orderID)
	if body.Status == "preparing" {
		h.db.Exec(ctx, "UPDATE orders SET preparation_started_at=NOW() WHERE id=$1", orderID)
	}
	if body.Status == "ready_for_pickup" {
		h.db.Exec(ctx, "UPDATE orders SET ready_at=NOW() WHERE id=$1", orderID)
	}
	h.db.Exec(ctx, "INSERT INTO order_status_history (order_id, status, changed_by) VALUES ($1,$2,$3)",
		orderID, body.Status, ownerID)
	h.hub.BroadcastToRoom("order:"+orderID, websocket.MsgOrderStatusUpdate, fiber.Map{
		"order_id": orderID, "status": body.Status,
	})
	return c.JSON(fiber.Map{"success": true, "data": fiber.Map{"status": body.Status}})
}

func (h *OwnerHandler) ListMenuSections(c *fiber.Ctx) error {
	ownerID := c.Locals("user_id").(string)
	ctx := context.Background()
	var rID string
	h.db.QueryRow(ctx, "SELECT id FROM restaurants WHERE owner_id=$1", ownerID).Scan(&rID)
	rows, _ := h.db.Query(ctx,
		"SELECT id, name, description, sort_order, is_active FROM menu_sections WHERE restaurant_id=$1 ORDER BY sort_order",
		rID)
	if rows != nil {
		defer rows.Close()
	}
	var sections []fiber.Map
	for rows.Next() {
		var id, name string
		var desc *string
		var sortOrder int
		var isActive bool
		rows.Scan(&id, &name, &desc, &sortOrder, &isActive)
		sections = append(sections, fiber.Map{
			"id": id, "name": name, "description": desc, "sort_order": sortOrder, "is_active": isActive,
		})
	}
	return c.JSON(fiber.Map{"success": true, "data": sections})
}

func (h *OwnerHandler) CreateMenuSection(c *fiber.Ctx) error {
	ownerID := c.Locals("user_id").(string)
	ctx := context.Background()
	var rID string
	h.db.QueryRow(ctx, "SELECT id FROM restaurants WHERE owner_id=$1", ownerID).Scan(&rID)
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
		SortOrder   int    `json:"sort_order"`
	}
	c.BodyParser(&body)
	var id string
	h.db.QueryRow(ctx,
		"INSERT INTO menu_sections (restaurant_id, name, description, sort_order) VALUES ($1,$2,$3,$4) RETURNING id",
		rID, body.Name, nullString(body.Description), body.SortOrder).Scan(&id)
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": fiber.Map{"id": id}})
}

func (h *OwnerHandler) UpdateMenuSection(c *fiber.Ctx) error {
	var body struct {
		Name     string `json:"name"`
		IsActive bool   `json:"is_active"`
	}
	c.BodyParser(&body)
	h.db.Exec(context.Background(),
		"UPDATE menu_sections SET name=COALESCE(NULLIF($1,''),name), is_active=$2 WHERE id=$3",
		body.Name, body.IsActive, c.Params("id"))
	return c.JSON(fiber.Map{"success": true})
}

func (h *OwnerHandler) DeleteMenuSection(c *fiber.Ctx) error {
	h.db.Exec(context.Background(), "DELETE FROM menu_sections WHERE id=$1", c.Params("id"))
	return c.JSON(fiber.Map{"success": true})
}

func (h *OwnerHandler) ListMenuItems(c *fiber.Ctx) error {
	ownerID := c.Locals("user_id").(string)
	ctx := context.Background()
	var rID string
	h.db.QueryRow(ctx, "SELECT id FROM restaurants WHERE owner_id=$1", ownerID).Scan(&rID)
	rows, _ := h.db.Query(ctx,
		`SELECT id, name, description, price, image_url, status, is_featured, section_id, sort_order
		 FROM menu_items WHERE restaurant_id=$1 ORDER BY sort_order`, rID)
	if rows != nil {
		defer rows.Close()
	}
	var items []fiber.Map
	for rows.Next() {
		var id, name, status string
		var desc, imgURL, sectionID *string
		var price float64
		var isFeat bool
		var sortOrder int
		rows.Scan(&id, &name, &desc, &price, &imgURL, &status, &isFeat, &sectionID, &sortOrder)
		items = append(items, fiber.Map{
			"id": id, "name": name, "description": desc, "price": price,
			"image_url": imgURL, "status": status, "is_featured": isFeat,
			"section_id": sectionID, "sort_order": sortOrder,
		})
	}
	return c.JSON(fiber.Map{"success": true, "data": items})
}

func (h *OwnerHandler) CreateMenuItem(c *fiber.Ctx) error {
	ownerID := c.Locals("user_id").(string)
	ctx := context.Background()
	var rID string
	h.db.QueryRow(ctx, "SELECT id FROM restaurants WHERE owner_id=$1", ownerID).Scan(&rID)
	var body struct {
		SectionID       string  `json:"section_id"`
		Name            string  `json:"name"`
		Description     string  `json:"description"`
		Price           float64 `json:"price"`
		Calories        int     `json:"calories"`
		PreparationTime int     `json:"preparation_time_min"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	var id string
	h.db.QueryRow(ctx,
		`INSERT INTO menu_items (restaurant_id, section_id, name, description, price, calories, preparation_time_min)
		 VALUES ($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
		rID, nullString(body.SectionID), body.Name, nullString(body.Description),
		body.Price, body.Calories, body.PreparationTime).Scan(&id)
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": fiber.Map{"id": id}})
}

func (h *OwnerHandler) UpdateMenuItem(c *fiber.Ctx) error {
	var body struct {
		Name        string  `json:"name"`
		Price       float64 `json:"price"`
		Description string  `json:"description"`
	}
	c.BodyParser(&body)
	h.db.Exec(context.Background(),
		`UPDATE menu_items SET
		 name = COALESCE(NULLIF($1,''), name),
		 price = CASE WHEN $2 > 0 THEN $2 ELSE price END,
		 description = COALESCE(NULLIF($3,''), description),
		 updated_at = NOW()
		 WHERE id = $4`,
		body.Name, body.Price, body.Description, c.Params("id"))
	return c.JSON(fiber.Map{"success": true})
}

func (h *OwnerHandler) DeleteMenuItem(c *fiber.Ctx) error {
	h.db.Exec(context.Background(), "DELETE FROM menu_items WHERE id=$1", c.Params("id"))
	return c.JSON(fiber.Map{"success": true})
}

func (h *OwnerHandler) ToggleItemAvailability(c *fiber.Ctx) error {
	var body struct {
		Status string `json:"status"`
	}
	c.BodyParser(&body)
	h.db.Exec(context.Background(), "UPDATE menu_items SET status=$1 WHERE id=$2", body.Status, c.Params("id"))
	return c.JSON(fiber.Map{"success": true})
}

func (h *OwnerHandler) AddItemOptions(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"success": true, "message": "options management via /menu/items/:id/options"})
}

func (h *OwnerHandler) GetAnalytics(c *fiber.Ctx) error { return h.GetDashboard(c) }

func (h *OwnerHandler) GetReviews(c *fiber.Ctx) error {
	ownerID := c.Locals("user_id").(string)
	ctx := context.Background()
	var rID string
	h.db.QueryRow(ctx, "SELECT id FROM restaurants WHERE owner_id=$1", ownerID).Scan(&rID)
	rows, _ := h.db.Query(ctx,
		`SELECT r.id, r.food_rating, r.delivery_rating, r.overall_rating, r.comment, r.created_at,
		 u.first_name, u.last_name
		 FROM reviews r JOIN users u ON u.id = r.customer_id
		 WHERE r.restaurant_id = $1 ORDER BY r.created_at DESC LIMIT 50`, rID)
	if rows != nil {
		defer rows.Close()
	}
	var reviews []fiber.Map
	for rows.Next() {
		var id, fn, ln string
		var foodR, delR, ovR *int
		var comment *string
		var createdAt time.Time
		rows.Scan(&id, &foodR, &delR, &ovR, &comment, &createdAt, &fn, &ln)
		reviews = append(reviews, fiber.Map{
			"id": id, "food_rating": foodR, "delivery_rating": delR, "overall_rating": ovR,
			"comment": comment, "created_at": createdAt, "customer_name": fn + " " + ln,
		})
	}
	return c.JSON(fiber.Map{"success": true, "data": reviews})
}

func (h *OwnerHandler) ReplyReview(c *fiber.Ctx) error {
	var body struct {
		Reply string `json:"reply"`
	}
	c.BodyParser(&body)
	h.db.Exec(context.Background(),
		"UPDATE reviews SET restaurant_reply=$1, restaurant_replied_at=NOW() WHERE id=$2",
		body.Reply, c.Params("id"))
	return c.JSON(fiber.Map{"success": true})
}

// ============================================
// ADMIN HANDLER
// ============================================

type AdminHandler struct{ *BaseHandler }

func (h *AdminHandler) GetDashboard(c *fiber.Ctx) error {
	ctx := context.Background()
	var stats models.DashboardStats
	h.db.QueryRow(ctx, "SELECT COALESCE(SUM(total_amount),0), COUNT(*) FROM orders WHERE status='delivered'").
		Scan(&stats.TotalRevenue, &stats.TotalOrders)
	h.db.QueryRow(ctx, "SELECT COUNT(*) FROM restaurants WHERE status='active'").Scan(&stats.ActiveRestaurants)
	h.db.QueryRow(ctx, "SELECT COUNT(*) FROM driver_profiles WHERE status='online'").Scan(&stats.ActiveDrivers)
	h.db.QueryRow(ctx, "SELECT COUNT(*) FROM users WHERE role='customer'").Scan(&stats.TotalCustomers)
	h.db.QueryRow(ctx,
		"SELECT COUNT(*), COALESCE(SUM(total_amount),0) FROM orders WHERE status='delivered' AND created_at >= CURRENT_DATE").
		Scan(&stats.OrdersToday, &stats.RevenueToday)
	h.db.QueryRow(ctx, "SELECT COALESCE(AVG(total_amount),0) FROM orders WHERE status='delivered'").
		Scan(&stats.AvgOrderValue)
	return c.JSON(fiber.Map{"success": true, "data": stats})
}

func (h *AdminHandler) ListUsers(c *fiber.Ctx) error {
	ctx := context.Background()
	role := c.Query("role", "")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	if page < 1 { page = 1 }
	offset := (page - 1) * limit

	q := "SELECT id, email, phone, role, first_name, last_name, is_active, created_at FROM users WHERE 1=1"
	args := []interface{}{}
	idx := 1
	if role != "" {
		q += fmt.Sprintf(" AND role=$%d", idx)
		args = append(args, role)
		idx++
	}
	q += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", idx, idx+1)
	args = append(args, limit, offset)

	rows, _ := h.db.Query(ctx, q, args...)
	if rows != nil { defer rows.Close() }
	var users []fiber.Map
	for rows.Next() {
		var id, email, userRole, fn, ln string
		var phone *string
		var isActive bool
		var createdAt time.Time
		rows.Scan(&id, &email, &phone, &userRole, &fn, &ln, &isActive, &createdAt)
		users = append(users, fiber.Map{
			"id": id, "email": email, "phone": phone, "role": userRole,
			"name": fn + " " + ln, "is_active": isActive, "created_at": createdAt,
		})
	}
	var total int64
	h.db.QueryRow(ctx, "SELECT COUNT(*) FROM users").Scan(&total)
	return c.JSON(fiber.Map{"success": true, "data": fiber.Map{"users": users, "total": total}})
}

func (h *AdminHandler) GetUser(c *fiber.Ctx) error {
	ctx := context.Background()
	var user models.User
	h.db.QueryRow(ctx,
		"SELECT id, email, phone, role, first_name, last_name, is_active, is_email_verified, created_at FROM users WHERE id=$1",
		c.Params("id")).
		Scan(&user.ID, &user.Email, &user.Phone, &user.Role, &user.FirstName, &user.LastName,
			&user.IsActive, &user.IsEmailVerified, &user.CreatedAt)
	return c.JSON(fiber.Map{"success": true, "data": user})
}

func (h *AdminHandler) UpdateUserStatus(c *fiber.Ctx) error {
	var body struct{ IsActive bool `json:"is_active"` }
	c.BodyParser(&body)
	h.db.Exec(context.Background(), "UPDATE users SET is_active=$1 WHERE id=$2", body.IsActive, c.Params("id"))
	return c.JSON(fiber.Map{"success": true})
}

func (h *AdminHandler) ListRestaurants(c *fiber.Ctx) error {
	ctx := context.Background()
	rows, _ := h.db.Query(ctx,
		"SELECT id, name, slug, city, status, rating, total_orders, is_open, created_at FROM restaurants ORDER BY created_at DESC LIMIT 100")
	if rows != nil { defer rows.Close() }
	var restaurants []fiber.Map
	for rows.Next() {
		var id, name, slug, city, status string
		var rating float64
		var totalOrders int
		var isOpen bool
		var createdAt time.Time
		rows.Scan(&id, &name, &slug, &city, &status, &rating, &totalOrders, &isOpen, &createdAt)
		restaurants = append(restaurants, fiber.Map{
			"id": id, "name": name, "slug": slug, "city": city, "status": status,
			"rating": rating, "total_orders": totalOrders, "is_open": isOpen, "created_at": createdAt,
		})
	}
	return c.JSON(fiber.Map{"success": true, "data": restaurants})
}

func (h *AdminHandler) UpdateRestaurantStatus(c *fiber.Ctx) error {
	var body struct{ Status string `json:"status"` }
	c.BodyParser(&body)
	h.db.Exec(context.Background(), "UPDATE restaurants SET status=$1 WHERE id=$2", body.Status, c.Params("id"))
	return c.JSON(fiber.Map{"success": true})
}

func (h *AdminHandler) ListOrders(c *fiber.Ctx) error {
	ctx := context.Background()
	status := c.Query("status", "")
	q := `SELECT o.id, o.order_number, o.status, o.total_amount, o.created_at, r.name
		  FROM orders o JOIN restaurants r ON r.id=o.restaurant_id WHERE 1=1`
	args := []interface{}{}
	idx := 1
	if status != "" {
		q += fmt.Sprintf(" AND o.status=$%d", idx)
		args = append(args, status)
		idx++
	}
	q += " ORDER BY o.created_at DESC LIMIT 200"
	rows, _ := h.db.Query(ctx, q, args...)
	if rows != nil { defer rows.Close() }
	var orders []fiber.Map
	for rows.Next() {
		var id, num, st, rName string
		var total float64
		var createdAt time.Time
		rows.Scan(&id, &num, &st, &total, &createdAt, &rName)
		orders = append(orders, fiber.Map{
			"id": id, "order_number": num, "status": st,
			"total_amount": total, "created_at": createdAt, "restaurant_name": rName,
		})
	}
	return c.JSON(fiber.Map{"success": true, "data": orders})
}

func (h *AdminHandler) GetOrder(c *fiber.Ctx) error {
	return (&OrderHandler{h.BaseHandler}).GetByID(c)
}

func (h *AdminHandler) ListDrivers(c *fiber.Ctx) error {
	ctx := context.Background()
	rows, _ := h.db.Query(ctx,
		`SELECT dp.id, u.first_name, u.last_name, u.email, dp.status, dp.vehicle_type,
		 dp.rating, dp.total_deliveries, dp.is_approved
		 FROM driver_profiles dp JOIN users u ON u.id=dp.user_id ORDER BY dp.created_at DESC`)
	if rows != nil { defer rows.Close() }
	var drivers []fiber.Map
	for rows.Next() {
		var id, fn, ln, email, status, vType string
		var rating float64
		var deliveries int
		var approved bool
		rows.Scan(&id, &fn, &ln, &email, &status, &vType, &rating, &deliveries, &approved)
		drivers = append(drivers, fiber.Map{
			"id": id, "name": fn + " " + ln, "email": email, "status": status,
			"vehicle_type": vType, "rating": rating, "total_deliveries": deliveries, "is_approved": approved,
		})
	}
	return c.JSON(fiber.Map{"success": true, "data": drivers})
}

func (h *AdminHandler) ApproveDriver(c *fiber.Ctx) error {
	h.db.Exec(context.Background(), "UPDATE driver_profiles SET is_approved=true WHERE id=$1", c.Params("id"))
	return c.JSON(fiber.Map{"success": true, "message": "driver approved"})
}

func (h *AdminHandler) GetAnalytics(c *fiber.Ctx) error { return h.GetDashboard(c) }

func (h *AdminHandler) GetRevenueAnalytics(c *fiber.Ctx) error {
	ctx := context.Background()
	rows, _ := h.db.Query(ctx,
		`SELECT DATE(created_at) as date, COUNT(*) as orders, COALESCE(SUM(total_amount),0) as revenue
		 FROM orders WHERE status='delivered' AND created_at >= NOW()-INTERVAL '30 days'
		 GROUP BY DATE(created_at) ORDER BY date`)
	if rows != nil { defer rows.Close() }
	var data []fiber.Map
	for rows.Next() {
		var date string
		var orderCount int64
		var revenue float64
		rows.Scan(&date, &orderCount, &revenue)
		data = append(data, fiber.Map{"date": date, "orders": orderCount, "revenue": revenue})
	}
	return c.JSON(fiber.Map{"success": true, "data": data})
}

func (h *AdminHandler) CreatePromotion(c *fiber.Ctx) error {
	var body struct {
		Code       string    `json:"code"`
		Type       string    `json:"type"`
		Value      float64   `json:"value"`
		StartsAt   time.Time `json:"starts_at"`
		ExpiresAt  time.Time `json:"expires_at"`
		UsageLimit *int      `json:"usage_limit"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	ctx := context.Background()
	var id string
	h.db.QueryRow(ctx,
		"INSERT INTO promotions (code, type, value, starts_at, expires_at, usage_limit) VALUES ($1,$2,$3,$4,$5,$6) RETURNING id",
		strings.ToUpper(body.Code), body.Type, body.Value, body.StartsAt, body.ExpiresAt, body.UsageLimit).Scan(&id)
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": fiber.Map{"id": id}})
}

func (h *AdminHandler) ListPromotions(c *fiber.Ctx) error {
	ctx := context.Background()
	rows, _ := h.db.Query(ctx,
		"SELECT id, code, type, value, usage_count, usage_limit, starts_at, expires_at, is_active FROM promotions ORDER BY created_at DESC")
	if rows != nil { defer rows.Close() }
	var promos []fiber.Map
	for rows.Next() {
		var id, code, pType string
		var value float64
		var count int
		var limit *int
		var startsAt, expiresAt time.Time
		var isActive bool
		rows.Scan(&id, &code, &pType, &value, &count, &limit, &startsAt, &expiresAt, &isActive)
		promos = append(promos, fiber.Map{
			"id": id, "code": code, "type": pType, "value": value,
			"usage_count": count, "usage_limit": limit,
			"starts_at": startsAt, "expires_at": expiresAt, "is_active": isActive,
		})
	}
	return c.JSON(fiber.Map{"success": true, "data": promos})
}

func (h *AdminHandler) UpdatePromotion(c *fiber.Ctx) error {
	var body struct{ IsActive bool `json:"is_active"` }
	c.BodyParser(&body)
	h.db.Exec(context.Background(), "UPDATE promotions SET is_active=$1 WHERE id=$2", body.IsActive, c.Params("id"))
	return c.JSON(fiber.Map{"success": true})
}

func (h *AdminHandler) DeletePromotion(c *fiber.Ctx) error {
	h.db.Exec(context.Background(), "DELETE FROM promotions WHERE id=$1", c.Params("id"))
	return c.JSON(fiber.Map{"success": true})
}

func (h *AdminHandler) ListCategories(c *fiber.Ctx) error {
	return (&RestaurantHandler{h.BaseHandler}).ListCategories(c)
}

func (h *AdminHandler) CreateCategory(c *fiber.Ctx) error {
	var body struct {
		Name    string `json:"name"`
		Slug    string `json:"slug"`
		IconURL string `json:"icon_url"`
	}
	c.BodyParser(&body)
	ctx := context.Background()
	var id string
	h.db.QueryRow(ctx,
		"INSERT INTO restaurant_categories (name, slug, icon_url) VALUES ($1,$2,$3) RETURNING id",
		body.Name, body.Slug, body.IconURL).Scan(&id)
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"success": true, "data": fiber.Map{"id": id}})
}

func (h *AdminHandler) UpdateCategory(c *fiber.Ctx) error {
	var body struct {
		Name    string `json:"name"`
		IconURL string `json:"icon_url"`
	}
	c.BodyParser(&body)
	h.db.Exec(context.Background(),
		"UPDATE restaurant_categories SET name=COALESCE(NULLIF($1,''),name), icon_url=COALESCE(NULLIF($2,''),icon_url) WHERE id=$3",
		body.Name, body.IconURL, c.Params("id"))
	return c.JSON(fiber.Map{"success": true})
}

func (h *AdminHandler) DeleteCategory(c *fiber.Ctx) error {
	h.db.Exec(context.Background(), "DELETE FROM restaurant_categories WHERE id=$1", c.Params("id"))
	return c.JSON(fiber.Map{"success": true})
}

// ============================================
// PAYMENT HANDLER
// ============================================

type PaymentHandler struct{ *BaseHandler }

func (h *PaymentHandler) CreateIntent(c *fiber.Ctx) error {
	var body struct {
		OrderID string  `json:"order_id"`
		Amount  float64 `json:"amount"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	// Production: call stripe.PaymentIntents.Create(...)
	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"client_secret": "pi_placeholder_secret_" + body.OrderID,
			"amount":        int(body.Amount * 100),
			"currency":      "gbp",
		},
	})
}

func (h *PaymentHandler) ConfirmPayment(c *fiber.Ctx) error {
	var body struct {
		OrderID         string `json:"order_id"`
		PaymentIntentID string `json:"payment_intent_id"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid body")
	}
	ctx := context.Background()
	h.db.Exec(ctx,
		"UPDATE orders SET payment_status='paid', stripe_payment_intent_id=$1 WHERE id=$2",
		body.PaymentIntentID, body.OrderID)
	h.hub.BroadcastToRoom("order:"+body.OrderID, websocket.MsgOrderStatusUpdate, fiber.Map{
		"order_id": body.OrderID, "payment_status": "paid",
	})
	return c.JSON(fiber.Map{"success": true, "message": "payment confirmed"})
}

func (h *PaymentHandler) StripeWebhook(c *fiber.Ctx) error {
	// Production: verify signature with stripe.ConstructEvent(...)
	return c.JSON(fiber.Map{"received": true})
}

// suppress unused import warnings at compile time
var _ = strconv.Atoi
var _ = strings.ToLower
