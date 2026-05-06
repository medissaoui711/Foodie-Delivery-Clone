package handlers

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"strings"
	"time"

	"github.com/deliveroo-clone/internal/middleware"
	"github.com/deliveroo-clone/internal/models"
	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	*BaseHandler
}

func (h *AuthHandler) Register(c *fiber.Ctx) error {
	var req models.RegisterRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if err := h.validate.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	ctx := context.Background()

	// Check email exists
	var exists bool
	err := h.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)", strings.ToLower(req.Email)).Scan(&exists)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}
	if exists {
		return fiber.NewError(fiber.StatusConflict, "email already registered")
	}

	// Hash password
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to hash password")
	}

	role := req.Role
	if role == "" {
		role = models.RoleCustomer
	}

	// Create user
	var user models.User
	err = h.db.QueryRow(ctx,
		`INSERT INTO users (email, password_hash, role, first_name, last_name, phone, auth_provider)
		 VALUES ($1, $2, $3, $4, $5, $6, 'email')
		 RETURNING id, email, phone, role, first_name, last_name, is_email_verified, is_active, created_at, updated_at`,
		strings.ToLower(req.Email), string(hash), role, req.FirstName, req.LastName,
		nullString(req.Phone),
	).Scan(&user.ID, &user.Email, &user.Phone, &user.Role, &user.FirstName, &user.LastName,
		&user.IsEmailVerified, &user.IsActive, &user.CreatedAt, &user.UpdatedAt)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to create user")
	}

	// Create wallet for customer/driver
	if role == models.RoleCustomer || role == models.RoleDriver {
		h.db.Exec(ctx, "INSERT INTO wallets (user_id) VALUES ($1)", user.ID)
	}

	// Generate tokens
	accessToken, err := middleware.GenerateAccessToken(user.ID, user.Email, string(user.Role), h.cfg.JWTSecret, h.cfg.JWTExpiry)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to generate token")
	}

	refreshToken, err := middleware.GenerateRefreshToken(user.ID, h.cfg.JWTRefreshSecret)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "failed to generate refresh token")
	}

	// Store refresh token
	h.db.Exec(ctx,
		"INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)",
		user.ID, hashToken(refreshToken), time.Now().Add(7*24*time.Hour),
	)

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"success": true,
		"data": models.AuthResponse{
			User:         &user,
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			ExpiresIn:    900,
		},
	})
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req models.LoginRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}
	if err := h.validate.Struct(req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}

	ctx := context.Background()

	var user models.User
	err := h.db.QueryRow(ctx,
		`SELECT id, email, phone, password_hash, role, first_name, last_name, avatar_url,
		 is_email_verified, is_active, created_at, updated_at
		 FROM users WHERE email = $1 AND auth_provider = 'email'`,
		strings.ToLower(req.Email),
	).Scan(&user.ID, &user.Email, &user.Phone, &user.PasswordHash, &user.Role,
		&user.FirstName, &user.LastName, &user.AvatarURL, &user.IsEmailVerified,
		&user.IsActive, &user.CreatedAt, &user.UpdatedAt)

	if err == pgx.ErrNoRows {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid email or password")
	}
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, "database error")
	}

	if !user.IsActive {
		return fiber.NewError(fiber.StatusForbidden, "account has been suspended")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid email or password")
	}

	// Update last login
	h.db.Exec(ctx, "UPDATE users SET last_login_at = NOW() WHERE id = $1", user.ID)

	accessToken, _ := middleware.GenerateAccessToken(user.ID, user.Email, string(user.Role), h.cfg.JWTSecret, h.cfg.JWTExpiry)
	refreshToken, _ := middleware.GenerateRefreshToken(user.ID, h.cfg.JWTRefreshSecret)

	h.db.Exec(ctx,
		"INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)",
		user.ID, hashToken(refreshToken), time.Now().Add(7*24*time.Hour),
	)

	user.PasswordHash = ""
	return c.JSON(fiber.Map{
		"success": true,
		"data": models.AuthResponse{
			User:         &user,
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			ExpiresIn:    900,
		},
	})
}

func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	ctx := context.Background()
	h.db.Exec(ctx, "DELETE FROM refresh_tokens WHERE user_id = $1", userID)
	return c.JSON(fiber.Map{"success": true, "message": "logged out successfully"})
}

func (h *AuthHandler) RefreshToken(c *fiber.Ctx) error {
	var req models.RefreshRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	ctx := context.Background()
	tokenHash := hashToken(req.RefreshToken)

	var userID string
	var expiresAt time.Time
	err := h.db.QueryRow(ctx,
		"SELECT user_id, expires_at FROM refresh_tokens WHERE token_hash = $1",
		tokenHash,
	).Scan(&userID, &expiresAt)

	if err == pgx.ErrNoRows {
		return fiber.NewError(fiber.StatusUnauthorized, "invalid refresh token")
	}
	if time.Now().After(expiresAt) {
		h.db.Exec(ctx, "DELETE FROM refresh_tokens WHERE token_hash = $1", tokenHash)
		return fiber.NewError(fiber.StatusUnauthorized, "refresh token expired")
	}

	var user models.User
	h.db.QueryRow(ctx,
		"SELECT id, email, role, first_name, last_name FROM users WHERE id = $1 AND is_active = true",
		userID,
	).Scan(&user.ID, &user.Email, &user.Role, &user.FirstName, &user.LastName)

	if user.ID == "" {
		return fiber.NewError(fiber.StatusUnauthorized, "user not found")
	}

	newAccessToken, _ := middleware.GenerateAccessToken(user.ID, user.Email, string(user.Role), h.cfg.JWTSecret, h.cfg.JWTExpiry)
	newRefreshToken, _ := middleware.GenerateRefreshToken(user.ID, h.cfg.JWTRefreshSecret)

	h.db.Exec(ctx, "DELETE FROM refresh_tokens WHERE token_hash = $1", tokenHash)
	h.db.Exec(ctx,
		"INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)",
		userID, hashToken(newRefreshToken), time.Now().Add(7*24*time.Hour),
	)

	return c.JSON(fiber.Map{
		"success":       true,
		"access_token":  newAccessToken,
		"refresh_token": newRefreshToken,
		"expires_in":    900,
	})
}

func (h *AuthHandler) Me(c *fiber.Ctx) error {
	userID := c.Locals("user_id").(string)
	ctx := context.Background()

	var user models.User
	err := h.db.QueryRow(ctx,
		`SELECT id, email, phone, role, first_name, last_name, avatar_url,
		 is_email_verified, is_phone_verified, is_active, last_login_at, created_at, updated_at
		 FROM users WHERE id = $1`,
		userID,
	).Scan(&user.ID, &user.Email, &user.Phone, &user.Role, &user.FirstName, &user.LastName,
		&user.AvatarURL, &user.IsEmailVerified, &user.IsPhoneVerified, &user.IsActive,
		&user.LastLoginAt, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return fiber.NewError(fiber.StatusNotFound, "user not found")
	}

	return c.JSON(fiber.Map{"success": true, "data": user})
}

func (h *AuthHandler) ForgotPassword(c *fiber.Ctx) error {
	var body struct {
		Email string `json:"email" validate:"required,email"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	ctx := context.Background()
	token := generateSecureToken(32)

	var userID string
	h.db.QueryRow(ctx, "SELECT id FROM users WHERE email = $1", strings.ToLower(body.Email)).Scan(&userID)

	if userID != "" {
		h.db.Exec(ctx,
			"INSERT INTO password_resets (user_id, token, expires_at) VALUES ($1, $2, $3)",
			userID, token, time.Now().Add(time.Hour),
		)
		// In production: send email with reset link
		// emailService.SendPasswordReset(body.Email, token)
	}

	// Always return success to prevent email enumeration
	return c.JSON(fiber.Map{
		"success": true,
		"message": "if that email exists, a reset link has been sent",
	})
}

func (h *AuthHandler) ResetPassword(c *fiber.Ctx) error {
	var body struct {
		Token    string `json:"token" validate:"required"`
		Password string `json:"password" validate:"required,min=8"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request body")
	}

	ctx := context.Background()
	var userID string
	var expiresAt time.Time
	var usedAt *time.Time

	h.db.QueryRow(ctx,
		"SELECT user_id, expires_at, used_at FROM password_resets WHERE token = $1",
		body.Token,
	).Scan(&userID, &expiresAt, &usedAt)

	if userID == "" || usedAt != nil || time.Now().After(expiresAt) {
		return fiber.NewError(fiber.StatusBadRequest, "invalid or expired reset token")
	}

	hash, _ := bcrypt.GenerateFromPassword([]byte(body.Password), bcrypt.DefaultCost)
	h.db.Exec(ctx, "UPDATE users SET password_hash = $1, updated_at = NOW() WHERE id = $2", string(hash), userID)
	h.db.Exec(ctx, "UPDATE password_resets SET used_at = NOW() WHERE token = $1", body.Token)
	h.db.Exec(ctx, "DELETE FROM refresh_tokens WHERE user_id = $1", userID)

	return c.JSON(fiber.Map{"success": true, "message": "password reset successful"})
}

func (h *AuthHandler) VerifyEmail(c *fiber.Ctx) error {
	var body struct {
		Token string `json:"token" validate:"required"`
	}
	if err := c.BodyParser(&body); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "invalid request")
	}

	ctx := context.Background()
	var userID string
	var expiresAt time.Time
	var usedAt *time.Time

	h.db.QueryRow(ctx,
		"SELECT user_id, expires_at, used_at FROM email_verifications WHERE token = $1",
		body.Token,
	).Scan(&userID, &expiresAt, &usedAt)

	if userID == "" || usedAt != nil || time.Now().After(expiresAt) {
		return fiber.NewError(fiber.StatusBadRequest, "invalid or expired verification token")
	}

	h.db.Exec(ctx, "UPDATE users SET is_email_verified = true WHERE id = $1", userID)
	h.db.Exec(ctx, "UPDATE email_verifications SET used_at = NOW() WHERE token = $1", body.Token)

	return c.JSON(fiber.Map{"success": true, "message": "email verified successfully"})
}

// Helpers
func hashToken(token string) string {
	h := make([]byte, 32)
	copy(h, token)
	return hex.EncodeToString(h)
}

func generateSecureToken(length int) string {
	b := make([]byte, length)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func nullString(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
