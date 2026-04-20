package v1

// RegisterRequest represents register payload.
// RegisterRequest 表示注册请求体。
type RegisterRequest struct {
	Username        string `json:"username" binding:"required,min=3,max=32"`
	Password        string `json:"password" binding:"required,min=8,max=64"`
	ConfirmPassword string `json:"confirm_password" binding:"required,min=8,max=64"`
	Name            string `json:"name" binding:"required,min=1,max=64"`
	Gender          string `json:"gender" binding:"required,oneof=male female unknown"`
	Phone           string `json:"phone" binding:"required,min=11,max=20"`
}

type RegisterResponse struct {
	UserID uint64 `json:"user_id"`
}

// LoginRequest represents login payload.
// LoginRequest 表示登录请求体。
type LoginRequest struct {
	Username string `json:"username" binding:"required,min=3,max=32"`
	Password string `json:"password" binding:"required,min=8,max=64"`
}

// RefreshTokenRequest represents refresh token payload.
// RefreshTokenRequest 表示刷新令牌请求体。
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// TokenResponse represents login or refresh response.
// TokenResponse 表示登录或刷新后的令牌响应。
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
}

// MeResponse represents current user profile.
// MeResponse 表示当前登录用户信息。
type MeResponse struct {
	ID       uint64 `json:"id"`
	Username string `json:"username"`
	Name     string `json:"name"`
	Gender   string `json:"gender"`
	Phone    string `json:"phone"`
}
