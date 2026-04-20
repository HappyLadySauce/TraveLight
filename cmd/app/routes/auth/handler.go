package auth

import (
	"context"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"k8s.io/klog/v2"
	"gorm.io/gorm"

	"github.com/HappyLadySauce/TraveLight/cmd/app/router"
	"github.com/HappyLadySauce/TraveLight/cmd/app/svc"
	v1 "github.com/HappyLadySauce/TraveLight/cmd/app/types/api/v1"
	"github.com/HappyLadySauce/TraveLight/cmd/app/types/common"
	"github.com/HappyLadySauce/TraveLight/pkg/model"
	"github.com/HappyLadySauce/TraveLight/pkg/utils/jwt"
	"github.com/HappyLadySauce/TraveLight/pkg/utils/passwd"
)

const userIDContextKey = "userID"

var phonePattern = regexp.MustCompile(`^[0-9+\-]{11,20}$`)

type Handler struct {
	svcCtx *svc.ServiceContext
}

func RegisterRoutes(svcCtx *svc.ServiceContext) {
	h := &Handler{svcCtx: svcCtx}
	group := router.V1().Group("/auth")
	group.POST("/register", h.register)
	group.POST("/login", h.login)
	group.POST("/refresh", h.refresh)
	group.GET("/me", AuthMiddleware(svcCtx), h.me)
}

// AuthMiddleware validates access token and injects user id.
// AuthMiddleware 校验访问令牌并注入用户ID。
func AuthMiddleware(svcCtx *svc.ServiceContext) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := strings.TrimSpace(c.GetHeader("Authorization"))
		if raw == "" || !strings.HasPrefix(raw, "Bearer ") {
			common.Fail(c, http.StatusUnauthorized, "missing bearer token")
			c.Abort()
			return
		}
		token := strings.TrimSpace(strings.TrimPrefix(raw, "Bearer "))
		claims, err := jwt.ParseToken(svcCtx.Config.AuthOptions.JWTSecret, token)
		if err != nil || claims.TokenType != jwt.TokenTypeAccess {
			common.Fail(c, http.StatusUnauthorized, "invalid access token")
			c.Abort()
			return
		}
		c.Set(userIDContextKey, claims.UserID)
		c.Next()
	}
}

// CurrentUserID returns user id from context.
// CurrentUserID 从上下文读取用户ID。
func CurrentUserID(c *gin.Context) (uint64, bool) {
	value, exists := c.Get(userIDContextKey)
	if !exists {
		return 0, false
	}
	userID, ok := value.(uint64)
	return userID, ok
}

// register godoc
//
//	@Summary		用户注册
//	@Description	注册新用户账号
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		v1.RegisterRequest	true	"注册信息"
//	@Success		200		{object}	common.BaseResponse
//	@Failure		400		{object}	common.BaseResponse
//	@Failure		409		{object}	common.BaseResponse
//	@Failure		500		{object}	common.BaseResponse
//	@Router			/api/v1/auth/register [post]
func (h *Handler) register(c *gin.Context) {
	var req v1.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, http.StatusBadRequest, "invalid request payload")
		return
	}
	if req.Password != req.ConfirmPassword {
		common.Fail(c, http.StatusBadRequest, "password confirmation mismatch")
		return
	}
	if !phonePattern.MatchString(req.Phone) {
		common.Fail(c, http.StatusBadRequest, "invalid phone format")
		return
	}

	username := strings.TrimSpace(req.Username)
	phone := strings.TrimSpace(req.Phone)
	var exists int64
	if err := h.svcCtx.DB.Model(&model.User{}).
		Where("username = ? OR phone = ?", username, phone).
		Count(&exists).Error; err != nil {
		common.Fail(c, http.StatusInternalServerError, "query user failed")
		return
	}
	if exists > 0 {
		common.Fail(c, http.StatusConflict, "username or phone already exists")
		return
	}

	hash, err := passwd.HashPassword(req.Password, "")
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, "hash password failed")
		return
	}
	user := model.User{
		Username:     username,
		PasswordHash: hash,
		Name:         strings.TrimSpace(req.Name),
		Gender:       req.Gender,
		Phone:        phone,
	}
	if err := h.svcCtx.DB.Create(&user).Error; err != nil {
		common.Fail(c, http.StatusInternalServerError, "create user failed")
		return
	}

	common.Success(c, v1.RegisterResponse{UserID: user.ID})
}

// login godoc
//
//	@Summary		用户登录
//	@Description	使用用户名和密码登录并获取 access/refresh token
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		v1.LoginRequest	true	"登录信息"
//	@Success		200		{object}	common.BaseResponse
//	@Failure		400		{object}	common.BaseResponse
//	@Failure		401		{object}	common.BaseResponse
//	@Failure		500		{object}	common.BaseResponse
//	@Router			/api/v1/auth/login [post]
func (h *Handler) login(c *gin.Context) {
	var req v1.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, http.StatusBadRequest, "invalid request payload")
		return
	}
	var user model.User
	if err := h.svcCtx.DB.Where("username = ?", strings.TrimSpace(req.Username)).First(&user).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			common.Fail(c, http.StatusUnauthorized, "invalid credentials")
			return
		}
		common.Fail(c, http.StatusInternalServerError, "query user failed")
		return
	}
	if !passwd.VerifyPassword(req.Password, "", user.PasswordHash) {
		common.Fail(c, http.StatusUnauthorized, "invalid credentials")
		return
	}

	resp, err := h.issueTokenPair(user.ID, user.Username)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, "issue token failed")
		return
	}
	common.Success(c, resp)
}

// refresh godoc
//
//	@Summary		刷新令牌
//	@Description	使用 refresh token 重新签发 access/refresh token
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Param			request	body		v1.RefreshTokenRequest	true	"刷新令牌请求"
//	@Success		200		{object}	common.BaseResponse
//	@Failure		400		{object}	common.BaseResponse
//	@Failure		401		{object}	common.BaseResponse
//	@Failure		500		{object}	common.BaseResponse
//	@Router			/api/v1/auth/refresh [post]
func (h *Handler) refresh(c *gin.Context) {
	var req v1.RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		common.Fail(c, http.StatusBadRequest, "invalid request payload")
		return
	}
	claims, err := jwt.ParseToken(h.svcCtx.Config.AuthOptions.JWTSecret, req.RefreshToken)
	if err != nil || claims.TokenType != jwt.TokenTypeRefresh {
		common.Fail(c, http.StatusUnauthorized, "invalid refresh token")
		return
	}

	var user model.User
	if err := h.svcCtx.DB.First(&user, claims.UserID).Error; err != nil {
		common.Fail(c, http.StatusUnauthorized, "user not found")
		return
	}
	resp, err := h.issueTokenPair(user.ID, user.Username)
	if err != nil {
		common.Fail(c, http.StatusInternalServerError, "issue token failed")
		return
	}
	common.Success(c, resp)
}

// me godoc
//
//	@Summary		获取当前用户
//	@Description	基于 access token 获取当前登录用户信息
//	@Tags			auth
//	@Accept			json
//	@Produce		json
//	@Security		ApiKeyAuth
//	@Success		200	{object}	common.BaseResponse
//	@Failure		401	{object}	common.BaseResponse
//	@Router			/api/v1/auth/me [get]
func (h *Handler) me(c *gin.Context) {
	userID, ok := CurrentUserID(c)
	if !ok {
		common.Fail(c, http.StatusUnauthorized, "unauthorized")
		return
	}
	var user model.User
	if err := h.svcCtx.DB.First(&user, userID).Error; err != nil {
		common.Fail(c, http.StatusUnauthorized, "user not found")
		return
	}
	common.Success(c, v1.MeResponse{
		ID:       user.ID,
		Username: user.Username,
		Name:     user.Name,
		Gender:   user.Gender,
		Phone:    user.Phone,
	})
}

func (h *Handler) issueTokenPair(userID uint64, username string) (*v1.TokenResponse, error) {
	accessTTL := h.svcCtx.Config.AuthOptions.AccessTokenTTL()
	refreshTTL := h.svcCtx.Config.AuthOptions.RefreshTokenTTL()

	accessToken, err := jwt.GenerateToken(
		h.svcCtx.Config.AuthOptions.JWTSecret,
		userID,
		username,
		jwt.TokenTypeAccess,
		accessTTL,
	)
	if err != nil {
		return nil, err
	}
	refreshToken, err := jwt.GenerateToken(
		h.svcCtx.Config.AuthOptions.JWTSecret,
		userID,
		username,
		jwt.TokenTypeRefresh,
		refreshTTL,
	)
	if err != nil {
		return nil, err
	}

	// Store refresh token issue time to support future revocation strategy.
	// 存储刷新令牌签发时间，为后续撤销策略预留。
	key := "auth:refresh:last_issue:" + username
	if redisErr := h.svcCtx.Redis.Set(context.Background(), key, time.Now().Unix(), refreshTTL).Err(); redisErr != nil {
		klog.ErrorS(redisErr, "Failed to cache refresh token metadata")
	}

	return &v1.TokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(accessTTL.Seconds()),
	}, nil
}
