package handlers

import (
	"context"
	"math"
	"strconv"

	"github.com/deliveroo-clone/internal/models"
	"github.com/gofiber/fiber/v2"
)

type RestaurantHandler struct {
	*BaseHandler
}

func (h *RestaurantHandler) List(c *fiber.Ctx) error {
	ctx := context.Background()
	lat, _ := strconv.ParseFloat(c.Query("lat", "0"), 64)
	lng, _ := strconv.ParseFloat(c.Query("lng", "0"), 64)
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	categorySlug := c.Query("category", "")

	if page < 1 { page = 1 }
	if limit < 1 || limit > 50 { limit = 20 }
	offset := (page - 1) * limit

	query := `
		SELECT r.id, r.name, r.slug, r.description, r.logo_url, r.cover_image_url,
		       r.city, r.postal_code, r.latitude, r.longitude,
		       r.delivery_fee, r.min_order_amount, r.estimated_delivery_time_min,
		       r.estimated_delivery_time_max, r.rating, r.total_reviews,
		       r.is_open, r.is_featured, r.tags, r.status
		FROM restaurants r
		WHERE r.status = 'active'
	`
	args := []interface{}{}
	argIdx := 1

	if categorySlug != "" {
		query += ` AND r.id IN (
			SELECT rcm.restaurant_id FROM restaurant_category_map rcm
			JOIN restaurant_categories rc ON rc.id = rcm.category_id
			WHERE rc.slug = $` + strconv.Itoa(argIdx) + `)`
		args = append(args, categorySlug)
		argIdx++
	}

	query += ` ORDER BY r.is_featured DESC, r.rating DESC LIMIT $` + strconv.Itoa(argIdx) + ` OFFSET $` + strconv.Itoa(argIdx+1)
	args = append(args, limit, offset)

	rows, err := h.db.Query(ctx, query, args...)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to fetch restaurants")
	}
	defer rows.Close()

	var restaurants []models.Restaurant
	for rows.Next() {
		var r models.Restaurant
		err := rows.Scan(
			&r.ID, &r.Name, &r.Slug, &r.Description, &r.LogoURL, &r.CoverImageURL,
			&r.City, &r.PostalCode, &r.Latitude, &r.Longitude,
			&r.DeliveryFee, &r.MinOrderAmount, &r.EstimatedDeliveryTimeMin,
			&r.EstimatedDeliveryTimeMax, &r.Rating, &r.TotalReviews,
			&r.IsOpen, &r.IsFeatured, &r.Tags, &r.Status,
		)
		if err != nil {
			continue
		}
		if lat != 0 && lng != 0 {
			r.DistanceKm = haversine(lat, lng, r.Latitude, r.Longitude)
		}
		restaurants = append(restaurants, r)
	}

	var total int64
	h.db.QueryRow(ctx, "SELECT COUNT(*) FROM restaurants WHERE status = 'active'").Scan(&total)

	return c.JSON(fiber.Map{
		"success": true,
		"data": models.PaginatedResponse{
			Data:       restaurants,
			Total:      total,
			Page:       page,
			Limit:      limit,
			TotalPages: int(math.Ceil(float64(total) / float64(limit))),
		},
	})
}

func (h *RestaurantHandler) Search(c *fiber.Ctx) error {
	ctx := context.Background()
	q := c.Query("q", "")
	if q == "" {
		return fiber.NewError(fiber.StatusBadRequest, "search query required")
	}

	rows, err := h.db.Query(ctx, `
		SELECT r.id, r.name, r.slug, r.description, r.logo_url, r.cover_image_url,
		       r.delivery_fee, r.min_order_amount, r.estimated_delivery_time_min,
		       r.estimated_delivery_time_max, r.rating, r.total_reviews,
		       r.is_open, r.is_featured, r.tags, r.status, r.latitude, r.longitude
		FROM restaurants r
		WHERE r.status = 'active'
		  AND (r.name ILIKE $1 OR r.description ILIKE $1 OR $1 = ANY(r.tags))
		UNION
		SELECT DISTINCT r.id, r.name, r.slug, r.description, r.logo_url, r.cover_image_url,
		       r.delivery_fee, r.min_order_amount, r.estimated_delivery_time_min,
		       r.estimated_delivery_time_max, r.rating, r.total_reviews,
		       r.is_open, r.is_featured, r.tags, r.status, r.latitude, r.longitude
		FROM restaurants r
		JOIN menu_items mi ON mi.restaurant_id = r.id
		WHERE r.status = 'active' AND mi.name ILIKE $1
		LIMIT 30`,
		"%"+q+"%",
	)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "search failed")
	}
	defer rows.Close()

	var restaurants []models.Restaurant
	for rows.Next() {
		var r models.Restaurant
		rows.Scan(
			&r.ID, &r.Name, &r.Slug, &r.Description, &r.LogoURL, &r.CoverImageURL,
			&r.DeliveryFee, &r.MinOrderAmount, &r.EstimatedDeliveryTimeMin,
			&r.EstimatedDeliveryTimeMax, &r.Rating, &r.TotalReviews,
			&r.IsOpen, &r.IsFeatured, &r.Tags, &r.Status, &r.Latitude, &r.Longitude,
		)
		restaurants = append(restaurants, r)
	}

	return c.JSON(fiber.Map{"success": true, "data": restaurants})
}

func (h *RestaurantHandler) ListCategories(c *fiber.Ctx) error {
	ctx := context.Background()
	rows, err := h.db.Query(ctx,
		"SELECT id, name, slug, icon_url, sort_order FROM restaurant_categories ORDER BY sort_order")
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to fetch categories")
	}
	defer rows.Close()

	var categories []models.Category
	for rows.Next() {
		var cat models.Category
		rows.Scan(&cat.ID, &cat.Name, &cat.Slug, &cat.IconURL, &cat.SortOrder)
		categories = append(categories, cat)
	}
	return c.JSON(fiber.Map{"success": true, "data": categories})
}

func (h *RestaurantHandler) ListFeatured(c *fiber.Ctx) error {
	ctx := context.Background()
	rows, err := h.db.Query(ctx, `
		SELECT id, name, slug, description, logo_url, cover_image_url,
		       delivery_fee, min_order_amount, estimated_delivery_time_min,
		       estimated_delivery_time_max, rating, total_reviews, is_open, tags, status
		FROM restaurants
		WHERE status = 'active' AND is_featured = true
		ORDER BY rating DESC LIMIT 10`)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to fetch featured restaurants")
	}
	defer rows.Close()

	var restaurants []models.Restaurant
	for rows.Next() {
		var r models.Restaurant
		rows.Scan(&r.ID, &r.Name, &r.Slug, &r.Description, &r.LogoURL, &r.CoverImageURL,
			&r.DeliveryFee, &r.MinOrderAmount, &r.EstimatedDeliveryTimeMin,
			&r.EstimatedDeliveryTimeMax, &r.Rating, &r.TotalReviews, &r.IsOpen, &r.Tags, &r.Status)
		restaurants = append(restaurants, r)
	}
	return c.JSON(fiber.Map{"success": true, "data": restaurants})
}

func (h *RestaurantHandler) GetBySlug(c *fiber.Ctx) error {
	slug := c.Params("slug")
	ctx := context.Background()

	var r models.Restaurant
	err := h.db.QueryRow(ctx, `
		SELECT id, owner_id, name, slug, description, logo_url, cover_image_url,
		       phone, email, status, address_line1, address_line2, city, postal_code,
		       latitude, longitude, delivery_radius_km, min_order_amount, delivery_fee,
		       estimated_delivery_time_min, estimated_delivery_time_max,
		       rating, total_reviews, total_orders, is_featured, is_open, tags, created_at
		FROM restaurants WHERE slug = $1 AND status = 'active'`, slug,
	).Scan(&r.ID, &r.OwnerID, &r.Name, &r.Slug, &r.Description, &r.LogoURL, &r.CoverImageURL,
		&r.Phone, &r.Email, &r.Status, &r.AddressLine1, &r.AddressLine2, &r.City, &r.PostalCode,
		&r.Latitude, &r.Longitude, &r.DeliveryRadiusKm, &r.MinOrderAmount, &r.DeliveryFee,
		&r.EstimatedDeliveryTimeMin, &r.EstimatedDeliveryTimeMax,
		&r.Rating, &r.TotalReviews, &r.TotalOrders, &r.IsFeatured, &r.IsOpen, &r.Tags, &r.CreatedAt)

	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "restaurant not found")
	}

	// Fetch hours
	rows, _ := h.db.Query(ctx,
		"SELECT id, day_of_week, open_time, close_time, is_closed FROM restaurant_hours WHERE restaurant_id = $1 ORDER BY day_of_week",
		r.ID)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var hour models.RestaurantHour
			rows.Scan(&hour.ID, &hour.DayOfWeek, &hour.OpenTime, &hour.CloseTime, &hour.IsClosed)
			r.Hours = append(r.Hours, hour)
		}
	}

	return c.JSON(fiber.Map{"success": true, "data": r})
}

func (h *RestaurantHandler) GetMenu(c *fiber.Ctx) error {
	restaurantID := c.Params("id")
	ctx := context.Background()

	// Fetch sections
	sRows, err := h.db.Query(ctx, `
		SELECT id, name, description, sort_order, is_active
		FROM menu_sections WHERE restaurant_id = $1 AND is_active = true ORDER BY sort_order`,
		restaurantID)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to fetch menu")
	}
	defer sRows.Close()

	var sections []models.MenuSection
	for sRows.Next() {
		var s models.MenuSection
		sRows.Scan(&s.ID, &s.Name, &s.Description, &s.SortOrder, &s.IsActive)
		sections = append(sections, s)
	}

	// Fetch items for each section
	for i, section := range sections {
		iRows, _ := h.db.Query(ctx, `
			SELECT id, name, description, price, image_url, status, is_featured,
			       calories, allergens, dietary_tags, preparation_time_min, sort_order, rating
			FROM menu_items WHERE section_id = $1 AND status = 'available' ORDER BY sort_order`,
			section.ID)
		if iRows == nil { continue }
		defer iRows.Close()

		for iRows.Next() {
			var item models.MenuItem
			iRows.Scan(&item.ID, &item.Name, &item.Description, &item.Price, &item.ImageURL,
				&item.Status, &item.IsFeatured, &item.Calories, &item.Allergens,
				&item.DietaryTags, &item.PreparationTime, &item.SortOrder, &item.Rating)
			sections[i].Items = append(sections[i].Items, item)
		}
	}

	return c.JSON(fiber.Map{"success": true, "data": sections})
}

func (h *RestaurantHandler) GetReviews(c *fiber.Ctx) error {
	restaurantID := c.Params("id")
	ctx := context.Background()
	page, _ := strconv.Atoi(c.Query("page", "1"))
	limit, _ := strconv.Atoi(c.Query("limit", "10"))
	if page < 1 { page = 1 }
	offset := (page - 1) * limit

	rows, err := h.db.Query(ctx, `
		SELECT r.id, r.food_rating, r.delivery_rating, r.overall_rating, r.comment,
		       r.created_at, u.first_name, u.last_name, u.avatar_url, r.is_anonymous
		FROM reviews r
		JOIN users u ON u.id = r.customer_id
		WHERE r.restaurant_id = $1
		ORDER BY r.created_at DESC LIMIT $2 OFFSET $3`,
		restaurantID, limit, offset)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to fetch reviews")
	}
	defer rows.Close()

	type ReviewItem struct {
		ID             string  `json:"id"`
		FoodRating     *int    `json:"food_rating"`
		DeliveryRating *int    `json:"delivery_rating"`
		OverallRating  *int    `json:"overall_rating"`
		Comment        *string `json:"comment"`
		CreatedAt      string  `json:"created_at"`
		CustomerName   string  `json:"customer_name"`
		AvatarURL      *string `json:"avatar_url"`
	}

	var reviews []ReviewItem
	for rows.Next() {
		var rv ReviewItem
		var firstName, lastName string
		var isAnonymous bool
		rows.Scan(&rv.ID, &rv.FoodRating, &rv.DeliveryRating, &rv.OverallRating,
			&rv.Comment, &rv.CreatedAt, &firstName, &lastName, &rv.AvatarURL, &isAnonymous)
		if isAnonymous {
			rv.CustomerName = "Anonymous"
			rv.AvatarURL = nil
		} else {
			rv.CustomerName = firstName + " " + string([]rune(lastName)[0]) + "."
		}
		reviews = append(reviews, rv)
	}

	return c.JSON(fiber.Map{"success": true, "data": reviews})
}

// haversine calculates distance between two lat/lng points in km
func haversine(lat1, lng1, lat2, lng2 float64) float64 {
	const R = 6371
	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180
	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLng/2)*math.Sin(dLng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}
