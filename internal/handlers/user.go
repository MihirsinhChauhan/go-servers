package handlers

import (
	"chirpy/internal/api"
	"chirpy/internal/auth"
	"chirpy/internal/database"
	"chirpy/internal/logger"
	"chirpy/internal/models"
	"chirpy/internal/utils"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

func HandleCreateUser(cfg *api.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		logger.Logger.Infow("Create user request",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
		)

		if r.Method != http.MethodPost {
			utils.RespondWithError(w, http.StatusMethodNotAllowed, "Method Not Allowed")
			return
		}

		var req models.CreateUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Logger.Warnw("Invalid JSON for user creation",
				"error", err,
			)
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid JSON")
			return
		}

		if req.Email == "" {
			logger.Logger.Warnw("Empty email supplied")
			utils.RespondWithError(w, http.StatusBadRequest, "Email is required")
			return
		}

		if req.Password == "" {
			logger.Logger.Warnw("Empty Password Supplied")
			utils.RespondWithError(w, http.StatusBadRequest, "Password is required")
		}

		hash, err := auth.HashPassword(req.Password)

		if err != nil {
			logger.Logger.Errorw("Failed to hash password", "error", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to process password")
			return
		}



		ctx := context.Background()
		user, err := cfg.DB.CreateUser(ctx, database.CreateUserParams{
			Email: req.Email,
			HashedPassword: hash,
		})
		if err != nil {
			if strings.Contains(err.Error(), "duplicate key") {
				logger.Logger.Warnw("Duplicate email attempt",
					"email", req.Email,
				)
				utils.RespondWithError(w, http.StatusConflict, "Email already exists")
				return
			}
			logger.Logger.Errorw("Failed to insert user into DB",
				"email", req.Email,
				"error", err,
			)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to create user")
			return
		}

		resp := models.CreateUserResponse{
			ID:        user.ID,
			Email:     user.Email,
			CreatedAt: user.CreatedAt.Format(time.RFC3339),
			UpdatedAt: user.UpdatedAt.Format(time.RFC3339),
		}

		logger.Logger.Infow("User created successfully",
			"user_id", user.ID,
			"email", user.Email,
		)

		utils.RespondWithJSON(w, http.StatusCreated, resp)
	}
}

func HandleUpdateUser(cfg *api.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// === 1. Extract and validate JWT ===
		tokenStr, err := auth.GetBearerToken(r.Header)
		if err != nil {
			logger.Logger.Warnw("Missing or malformed Authorization header",
				"error", err,
				"path", r.URL.Path,
			)
			utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		userID, err := auth.ValidateJWT(tokenStr, cfg.JWTSecret)
		if err != nil {
			logger.Logger.Infow("Invalid or expired access token",
				"error", err,
				"token_preview", auth.TruncateToken(tokenStr),
			)
			utils.RespondWithError(w, http.StatusUnauthorized, "Unauthorized")
			return
		}

		// === 2. Parse request body ===
		var req models.UpdateUserRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Logger.Warnw("Invalid JSON payload for user update",
				"error", err,
			)
			utils.RespondWithError(w, http.StatusBadRequest, "Invalid request payload")
			return
		}

		// === 3. Validate fields ===
		if req.Email == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "Email is required")
			return
		}
		if req.Password == "" {
			utils.RespondWithError(w, http.StatusBadRequest, "Password is required")
			return
		}


		// === 4. Hash new password ===
		hashedPassword, err := auth.HashPassword(req.Password)
		if err != nil {
			logger.Logger.Errorw("Failed to hash password", "error", err)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update user")
			return
		}

		// === 5. Update user in DB ===
		ctx := r.Context()
		updatedUser, err := cfg.DB.UpdateUser(ctx, database.UpdateUserParams{
			ID:             userID,
			Email:          req.Email,
			HashedPassword: hashedPassword,
		})
		if err != nil {
			logger.Logger.Errorw("Failed to update user in database",
				"error", err,
				"user_id", userID,
			)
			utils.RespondWithError(w, http.StatusInternalServerError, "Failed to update user")
			return
		}
		// === 6. Build response (omit password) ===
		resp := models.UpdateUserResponse{
			ID:        updatedUser.ID,
			Email:     updatedUser.Email,
			CreatedAt: updatedUser.CreatedAt.Format(time.RFC3339),
			UpdatedAt: updatedUser.UpdatedAt.Format(time.RFC3339),
		}

		// === 7. Log success ===
		logger.Logger.Infow("User updated successfully",
			"user_id", updatedUser.ID,
			"new_email", updatedUser.Email,
		)

		utils.RespondWithJSON(w, http.StatusOK, resp)
	
	}
}