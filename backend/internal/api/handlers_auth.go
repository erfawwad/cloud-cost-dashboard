package api

import (
	"encoding/json"
	"net/http"

	appauth "cloudcostdash/internal/auth"
	"cloudcostdash/internal/models"
)

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}

	var user models.User
	if err := s.DB.Where("email = ?", req.Email).First(&user).Error; err != nil {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	if !appauth.CheckPassword(user.PasswordHash, req.Password) {
		writeError(w, http.StatusUnauthorized, "invalid email or password")
		return
	}

	token, err := appauth.IssueToken(s.JWTSecret, user)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to issue token")
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"token": token,
		"user":  user,
	})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	claims, ok := appauth.FromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}

	var user models.User
	if err := s.DB.First(&user, claims.UserID).Error; err != nil {
		writeError(w, http.StatusNotFound, "user not found")
		return
	}

	writeJSON(w, http.StatusOK, user)
}

type createUserRequest struct {
	Email     string           `json:"email"`
	Password  string           `json:"password"`
	Role      models.Role      `json:"role"`
	ScopeType models.ScopeType `json:"scopeType"`
	ScopeID   *uint            `json:"scopeId"`
}

func (s *Server) handleCreateUser(w http.ResponseWriter, r *http.Request) {
	var req createUserRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if req.Email == "" || req.Password == "" {
		writeError(w, http.StatusBadRequest, "email and password are required")
		return
	}
	if req.Role != models.RoleAdmin && req.Role != models.RoleManager && req.Role != models.RoleViewer {
		writeError(w, http.StatusBadRequest, "role must be admin, manager, or viewer")
		return
	}

	hash, err := appauth.HashPassword(req.Password)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "failed to hash password")
		return
	}

	scopeType := req.ScopeType
	if scopeType == "" {
		scopeType = models.ScopeNone
	}

	user := models.User{
		Email:        req.Email,
		PasswordHash: hash,
		Role:         req.Role,
		ScopeType:    scopeType,
		ScopeID:      req.ScopeID,
	}
	if err := s.DB.Create(&user).Error; err != nil {
		writeError(w, http.StatusConflict, "could not create user (email may already exist)")
		return
	}

	writeJSON(w, http.StatusCreated, user)
}

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users := []models.User{}
	if err := s.DB.Find(&users).Error; err != nil {
		writeError(w, http.StatusInternalServerError, "failed to list users")
		return
	}
	writeJSON(w, http.StatusOK, users)
}
