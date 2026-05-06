package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/deliveroo-clone/internal/models"
	"github.com/deliveroo-clone/internal/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
)

type OrderHandler struct {
	*BaseHandler
}

func (h *OrderHandler) Create(c *fiber.Ctx) error {
	customerID := c.Locals("user_id").(string)
	var req models.CreateOrderRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if err := h.validate.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	ctx := context.Background()

	// Validate restaurant exists and is open
	var restaurant models.Restaurant
	err := h.db.QueryRow(ctx,
		`SELECT id, name, delivery_fee, min_order_amount, is_open, status, owner_id
		 FROM restaurants WHERE id = $1`,
		req.RestaurantID,
	).Scan(&restaurant.ID, &restaurant.Name, &restaurant.DeliveryFee,
		&restaurant.MinOrderAmount, &restaurant.IsOpen, &restaurant.Status, &restaurant.OwnerID)
	if err == pgx.ErrNoRows {
		return fiber.NewError(fiber.StatusNotFound, "restaurant not found")
	}
	if restaurant.Status != "active" {
		return fiber.NewError(fiber.StatusBadRequest, "restaurant is not accepting orders")
	}
	if !restaurant.IsOpen {
		return fiber.NewError(fiber.StatusBadRequest, "restaurant is currently closed")
	}

	// Validate delivery address belongs to customer
	var addrExists bool
	h.db.QueryRow(ctx,
		"SELECT EXISTS(SELECT 1 FROM addresses WHERE id = $1 AND user_id = $2)",
		req.DeliveryAddressID, customerID,
	).Scan(&addrExists)
	if !addrExists {
		return fiber.NewError(fiber.StatusBadRequest, "invalid delivery address")
	}

	// Fetch address snapshot
	var addrSnapshot map[string]interface{}
	h.db.QueryRow(ctx,
		`SELECT row_to_json(a) FROM addresses a WHERE id = $1`,
		req.DeliveryAddressID,
	).Scan(&addrSnapshot)

	// Calculate order totals
	var subtotal float64
	type orderItemCalc struct {
		menuItemID   string
		name         string
		description  *string
		imageURL     *string
		unitPrice    float64
		quantity     int
		optionsPrice float64
		subtotal     float64
		options      interface{}
		instructions string
	}
	var calcItems []orderItemCalc

	for _, reqItem := range req.Items {
		var item struct {
			ID          string  `db:"id"`
			Name        string  `db:"name"`
			Description *string `db:"description"`
			ImageURL    *string `db:"image_url"`
			Price       float64 `db:"price"`
			Status      string  `db:"status"`
		}
		err := h.db.QueryRow(ctx,
			"SELECT id, name, description, image_url, price, status FROM menu_items WHERE id = $1 AND restaurant_id = $2",
			reqItem.MenuItemID, req.RestaurantID,
		).Scan(&item.ID, &item.Name, &item.Description, &item.ImageURL, &item.Price, &item.Status)

		if err == pgx.ErrNoRows {
			return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("menu item %s not found", reqItem.MenuItemID))
		}
		if item.Status != "available" {
			return fiber.NewError(fiber.StatusBadRequest, fmt.Sprintf("item '%s' is not available", item.Name))
		}

		// Calculate options price
		var optionsPrice float64
		if len(reqItem.SelectedOptions) > 0 {
			for _, optID := range reqItem.SelectedOptions {
				var priceAdd float64
				h.db.QueryRow(ctx,
					"SELECT price_addition FROM item_options WHERE id = $1 AND is_available = true",
					optID,
				).Scan(&priceAdd)
				optionsPrice += priceAdd
			}
		}

		itemSubtotal := (item.Price + optionsPrice) * float64(reqItem.Quantity)
		subtotal += itemSubtotal

		optJSON, _ := json.Marshal(reqItem.SelectedOptions)
		calcItems = append(calcItems, orderItemCalc{
			menuItemID:   item.ID,
			name:         item.Name,
			description:  item.Description,
			imageURL:     item.ImageURL,
			unitPrice:    item.Price,
			quantity:     reqItem.Quantity,
			optionsPrice: optionsPrice,
			subtotal:     itemSubtotal,
			options:      string(optJSON),
			instructions: reqItem.SpecialInstructions,
		})
	}

	if subtotal < restaurant.MinOrderAmount {
		return fiber.NewError(fiber.StatusBadRequest,
			fmt.Sprintf("minimum order amount is £%.2f", restaurant.MinOrderAmount))
	}

	// Apply promo code
	var discountAmount float64
	if req.PromoCode != "" {
		var promo struct {
			ID         string
			Type       string
			Value      float64
			MaxDiscount *float64
		}
		err := h.db.QueryRow(ctx, `
			SELECT id, type, value, max_discount_amount FROM promotions
			WHERE code = $1 AND is_active = true
			  AND starts_at <= NOW() AND expires_at >= NOW()
			  AND (restaurant_id IS NULL OR restaurant_id = $2)
			  AND (usage_limit IS NULL OR usage_count < usage_limit)`,
			strings.ToUpper(req.PromoCode), req.RestaurantID,
		).Scan(&promo.ID, &promo.Type, &promo.Value, &promo.MaxDiscount)

		if err == nil {
			switch promo.Type {
			case "percentage":
				discountAmount = subtotal * promo.Value / 100
			case "fixed_amount":
				discountAmount = promo.Value
			case "free_delivery":
				discountAmount = restaurant.DeliveryFee
			}
			if promo.MaxDiscount != nil && discountAmount > *promo.MaxDiscount {
				discountAmount = *promo.MaxDiscount
			}
		}
	}

	serviceFee := subtotal * 0.05 // 5% service fee
	if serviceFee > 1.99 { serviceFee = 1.99 }
	totalAmount := subtotal + restaurant.DeliveryFee + serviceFee - discountAmount + req.TipAmount

	// Generate order number
	var orderNumber string
	h.db.QueryRow(ctx, "SELECT generate_order_number()").Scan(&orderNumber)
	orderNumber = "ORD-" + orderNumber

	addrJSON, _ := json.Marshal(addrSnapshot)

	// Create order in transaction
	tx, err := h.db.Begin(ctx)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to start transaction")
	}
	defer tx.Rollback(ctx)

	var orderID string
	err = tx.QueryRow(ctx, `
		INSERT INTO orders (
			order_number, customer_id, restaurant_id, status, payment_method, payment_status,
			subtotal, delivery_fee, service_fee, discount_amount, tip_amount, total_amount,
			promo_code, delivery_address_id, delivery_address_snapshot, delivery_instructions,
			restaurant_note, estimated_delivery_at
		) VALUES ($1, $2, $3, 'pending', $4, 'pending', $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		RETURNING id`,
		orderNumber, customerID, req.RestaurantID, req.PaymentMethod,
		subtotal, restaurant.DeliveryFee, serviceFee, discountAmount, req.TipAmount, totalAmount,
		nullString(req.PromoCode), req.DeliveryAddressID, addrJSON,
		nullString(req.DeliveryInstructions), nullString(req.RestaurantNote),
		time.Now().Add(45*time.Minute),
	).Scan(&orderID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create order")
	}

	// Insert order items
	for _, item := range calcItems {
		tx.Exec(ctx, `
			INSERT INTO order_items (order_id, menu_item_id, name, description, image_url,
			unit_price, quantity, selected_options, options_price, subtotal, special_instructions)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
			orderID, item.menuItemID, item.name, item.description, item.imageURL,
			item.unitPrice, item.quantity, item.options, item.optionsPrice, item.subtotal,
			nullString(item.instructions),
		)
	}

	// Insert status history
	tx.Exec(ctx,
		"INSERT INTO order_status_history (order_id, status, changed_by) VALUES ($1, 'pending', $2)",
		orderID, customerID)

	if err := tx.Commit(ctx); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to commit order")
	}

	// Join WebSocket room
	h.hub.JoinRoom(customerID, "order:"+orderID)

	// Notify restaurant owner via WebSocket
	h.hub.SendToUser(restaurant.OwnerID, websocket.MsgNewOrder, map[string]interface{}{
		"order_id":     orderID,
		"order_number": orderNumber,
		"total":        totalAmount,
	})

	// Create notification for customer
	h.db.Exec(ctx, `
		INSERT INTO notifications (user_id, type, title, body, data)
		VALUES ($1, 'order_confirmed', 'Order Placed!', $2, $3)`,
		customerID,
		fmt.Sprintf("Your order #%s has been placed successfully", orderNumber),
		fmt.Sprintf(`{"order_id":"%s"}`, orderID),
	)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"order_id":     orderID,
			"order_number": orderNumber,
			"total_amount": totalAmount,
			"status":       "pending",
			"estimated_delivery_at": time.Now().Add(45 * time.Minute),
		},
	})
}

func (h *OrderHandler) GetByID(c *fiber.Ctx) error {
	orderID := c.Params("id")
	userID := c.Locals("user_id").(string)
	userRole := c.Locals("user_role").(string)
	ctx := context.Background()

	var order models.Order
	err := h.db.QueryRow(ctx, `
		SELECT o.id, o.order_number, o.customer_id, o.restaurant_id, o.driver_id,
		       o.status, o.payment_method, o.payment_status,
		       o.subtotal, o.delivery_fee, o.service_fee, o.discount_amount,
		       o.tip_amount, o.total_amount, o.promo_code,
		       o.delivery_address_snapshot, o.delivery_instructions,
		       o.estimated_delivery_at, o.actual_delivery_at,
		       o.cancelled_at, o.cancellation_reason, o.restaurant_note,
		       o.created_at, o.updated_at
		FROM orders o WHERE o.id = $1`, orderID,
	).Scan(&order.ID, &order.OrderNumber, &order.CustomerID, &order.RestaurantID, &order.DriverID,
		&order.Status, &order.PaymentMethod, &order.PaymentStatus,
		&order.Subtotal, &order.DeliveryFee, &order.ServiceFee, &order.DiscountAmount,
		&order.TipAmount, &order.TotalAmount, &order.PromoCode,
		&order.DeliveryAddressSnapshot, &order.DeliveryInstructions,
		&order.EstimatedDeliveryAt, &order.ActualDeliveryAt,
		&order.CancelledAt, &order.CancellationReason, &order.RestaurantNote,
		&order.CreatedAt, &order.UpdatedAt)

	if err == pgx.ErrNoRows {
		return fiber.NewError(fiber.StatusNotFound, "order not found")
	}

	// Access control
	if userRole != "admin" {
		if order.CustomerID != userID && order.RestaurantID != userID {
			if order.DriverID == nil || *order.DriverID != userID {
				return fiber.NewError(fiber.StatusForbidden, "access denied")
			}
		}
	}

	// Fetch items
	iRows, _ := h.db.Query(ctx,
		"SELECT id, name, description, image_url, unit_price, quantity, selected_options, options_price, subtotal, special_instructions FROM order_items WHERE order_id = $1",
		orderID)
	if iRows != nil {
		defer iRows.Close()
		for iRows.Next() {
			var item models.OrderItem
			iRows.Scan(&item.ID, &item.Name, &item.Description, &item.ImageURL,
				&item.UnitPrice, &item.Quantity, &item.SelectedOptions,
				&item.OptionsPrice, &item.Subtotal, &item.SpecialInstructions)
			order.Items = append(order.Items, item)
		}
	}

	// Join tracking room
	h.hub.JoinRoom(userID, "order:"+orderID)

	return c.JSON(fiber.Map{"success": true, "data": order})
}

func (h *OrderHandler) Track(c *fiber.Ctx) error {
	orderID := c.Params("id")
	userID := c.Locals("user_id").(string)
	ctx := context.Background()

	var status models.OrderStatus
	var driverLat, driverLng *float64
	var estimatedAt *time.Time

	h.db.QueryRow(ctx, `
		SELECT o.status, o.estimated_delivery_at,
		       dp.current_latitude, dp.current_longitude
		FROM orders o
		LEFT JOIN driver_profiles dp ON dp.user_id = o.driver_id
		WHERE o.id = $1 AND (o.customer_id = $2 OR o.driver_id = $2)`,
		orderID, userID,
	).Scan(&status, &estimatedAt, &driverLat, &driverLng)

	h.hub.JoinRoom(userID, "order:"+orderID)

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"order_id":              orderID,
			"status":                status,
			"estimated_delivery_at": estimatedAt,
			"driver_location": fiber.Map{
				"latitude":  driverLat,
				"longitude": driverLng,
			},
		},
	})
}

func (h *OrderHandler) Cancel(c *fiber.Ctx) error {
	orderID := c.Params("id")
	customerID := c.Locals("user_id").(string)
	ctx := context.Background()

	var body struct {
		Reason string `json:"reason"`
	}
	c.BodyParser(&body)

	var status models.OrderStatus
	h.db.QueryRow(ctx,
		"SELECT status FROM orders WHERE id = $1 AND customer_id = $2",
		orderID, customerID,
	).Scan(&status)

	if status == "" {
		return fiber.NewError(fiber.StatusNotFound, "order not found")
	}

	cancelableStatuses := map[models.OrderStatus]bool{
		models.OrderPending:   true,
		models.OrderConfirmed: true,
	}
	if !cancelableStatuses[status] {
		return fiber.NewError(fiber.StatusBadRequest, "order cannot be cancelled at this stage")
	}

	h.db.Exec(ctx, `
		UPDATE orders SET status = 'cancelled', cancelled_at = NOW(), cancellation_reason = $1
		WHERE id = $2`, nullString(body.Reason), orderID)

	h.db.Exec(ctx,
		"INSERT INTO order_status_history (order_id, status, changed_by, note) VALUES ($1, 'cancelled', $2, $3)",
		orderID, customerID, body.Reason)

	h.hub.BroadcastToRoom("order:"+orderID, websocket.MsgOrderStatusUpdate, fiber.Map{
		"order_id": orderID,
		"status":   "cancelled",
	})

	return c.JSON(fiber.Map{"success": true, "message": "order cancelled successfully"})
}

func (h *OrderHandler) ValidatePromo(c *fiber.Ctx) error {
	var body struct {
		Code         string  `json:"code" validate:"required"`
		RestaurantID string  `json:"restaurant_id" validate:"required"`
		Subtotal     float64 `json:"subtotal" validate:"required"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request")
	}

	ctx := context.Background()
	var promo struct {
		ID          string
		Type        string
		Value       float64
		MinOrder    float64
		MaxDiscount *float64
	}

	err := h.db.QueryRow(ctx, `
		SELECT id, type, value, min_order_amount, max_discount_amount
		FROM promotions
		WHERE code = $1 AND is_active = true
		  AND starts_at <= NOW() AND expires_at >= NOW()
		  AND (restaurant_id IS NULL OR restaurant_id = $2)
		  AND min_order_amount <= $3`,
		strings.ToUpper(body.Code), body.RestaurantID, body.Subtotal,
	).Scan(&promo.ID, &promo.Type, &promo.Value, &promo.MinOrder, &promo.MaxDiscount)

	if err == pgx.ErrNoRows {
		return fiber.NewError(fiber.StatusBadRequest, "invalid or expired promo code")
	}

	var discount float64
	switch promo.Type {
	case "percentage":
		discount = body.Subtotal * promo.Value / 100
	case "fixed_amount":
		discount = promo.Value
	}
	if promo.MaxDiscount != nil && discount > *promo.MaxDiscount {
		discount = *promo.MaxDiscount
	}

	return c.JSON(fiber.Map{
		"success": true,
		"data": fiber.Map{
			"valid":           true,
			"promo_type":      promo.Type,
			"discount_amount": discount,
		},
	})
}
