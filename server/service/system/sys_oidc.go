package system

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/flipped-aurora/gin-vue-admin/server/global"
	"github.com/flipped-aurora/gin-vue-admin/server/model/system"
	systemReq "github.com/flipped-aurora/gin-vue-admin/server/model/system/request"
	systemRes "github.com/flipped-aurora/gin-vue-admin/server/model/system/response"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type OidcService struct{}

var OidcServiceApp = new(OidcService)

type StorageAdapter interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Del(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) (int64, error)
	Incr(ctx context.Context, key string) error
	PTTL(ctx context.Context, key string) (time.Duration, error)
	TxPipeline() interface{}
}

type RedisStorageAdapter struct{}

func (r *RedisStorageAdapter) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return global.GVA_REDIS.Set(ctx, key, value, expiration).Err()
}

func (r *RedisStorageAdapter) Get(ctx context.Context, key string) (string, error) {
	return global.GVA_REDIS.Get(ctx, key).Result()
}

func (r *RedisStorageAdapter) Del(ctx context.Context, key string) error {
	return global.GVA_REDIS.Del(ctx, key).Err()
}

func (r *RedisStorageAdapter) Exists(ctx context.Context, key string) (int64, error) {
	return global.GVA_REDIS.Exists(ctx, key).Result()
}

func (r *RedisStorageAdapter) Incr(ctx context.Context, key string) error {
	return global.GVA_REDIS.Incr(ctx, key).Err()
}

func (r *RedisStorageAdapter) PTTL(ctx context.Context, key string) (time.Duration, error) {
	return global.GVA_REDIS.PTTL(ctx, key).Result()
}

func (r *RedisStorageAdapter) TxPipeline() interface{} {
	return global.GVA_REDIS.TxPipeline()
}

type MemoryStorageAdapter struct{}

func (m *MemoryStorageAdapter) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return GetMemoryStore().Set(ctx, key, value, expiration)
}

func (m *MemoryStorageAdapter) Get(ctx context.Context, key string) (string, error) {
	return GetMemoryStore().Get(ctx, key)
}

func (m *MemoryStorageAdapter) Del(ctx context.Context, key string) error {
	return GetMemoryStore().Del(ctx, key)
}

func (m *MemoryStorageAdapter) Exists(ctx context.Context, key string) (int64, error) {
	return GetMemoryStore().Exists(ctx, key)
}

func (m *MemoryStorageAdapter) Incr(ctx context.Context, key string) error {
	return GetMemoryStore().Incr(ctx, key)
}

func (m *MemoryStorageAdapter) PTTL(ctx context.Context, key string) (time.Duration, error) {
	return GetMemoryStore().PTTL(ctx, key)
}

func (m *MemoryStorageAdapter) TxPipeline() interface{} {
	return GetMemoryStore().TxPipeline()
}

func (o *OidcService) GetStorage() StorageAdapter {
	if global.GVA_CONFIG.System.UseRedis && global.GVA_REDIS != nil {
		return &RedisStorageAdapter{}
	}
	return &MemoryStorageAdapter{}
}

// OidcConfig OIDC配置结构
type OidcConfig struct {
	ClientID      string
	ClientSecret  string
	RedirectURL   string
	Scopes        string
	AuthURL       string
	TokenURL      string
	UserInfoURL   string
	EndSessionURL string
	DiscoveryURL  string
}

// OidcDiscoveryResponse OIDC发现响应
type OidcDiscoveryResponse struct {
	Issuer        string   `json:"issuer"`
	AuthURL       string   `json:"authorization_endpoint"`
	TokenURL      string   `json:"token_endpoint"`
	UserInfoURL   string   `json:"userinfo_endpoint"`
	EndSessionURL string   `json:"end_session_endpoint"`
	JWKSURL       string   `json:"jwks_uri"`
	ResponseTypes []string `json:"response_types_supported"`
	GrantTypes    []string `json:"grant_types_supported"`
	Scopes        []string `json:"scopes_supported"`
}

// OidcTokenResponse OIDC令牌响应
type OidcTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
}

// OidcUserInfo OIDC用户信息
type OidcUserInfo struct {
	Sub               string `json:"sub"`
	Username          string `json:"username"`
	Name              string `json:"name"`
	PreferredUsername string `json:"preferred_username"`
	Email             string `json:"email"`
	Picture           string `json:"picture"`
	Nickname          string `json:"nickname"`
	EmailVerified     bool   `json:"email_verified"`
}

// GetAuthURL 获取授权URL
func (o *OidcService) GetAuthURL(provider string) (*systemRes.OidcLoginResponse, error) {
	if !global.GVA_CONFIG.OIDC.Enabled {
		return nil, fmt.Errorf("OIDC功能未启用")
	}

	// 生成状态参数
	state, err := o.generateState()
	if err != nil {
		return nil, fmt.Errorf("生成状态参数失败: %v", err)
	}

	// 存储状态（10分钟过期）
	ctx := context.Background()
	storage := o.GetStorage()
	err = storage.Set(ctx, fmt.Sprintf("oidc_state:%s", state), provider, 10*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("存储状态参数失败: %v", err)
	}

	config := o.getOidcConfig(provider)
	if config == nil {
		return nil, fmt.Errorf("不支持的OIDC服务提供方: %s", provider)
	}

	// 如果有发现端点，先测试连接
	if config.DiscoveryURL != "" {
		if err := o.testOIDCConnection(config.DiscoveryURL); err != nil {
			return nil, fmt.Errorf("OIDC服务提供方连接失败: %v，请检查服务是否正常运行", err)
		}
	}

	// 构建授权URL
	authURL, err := o.buildAuthURL(config, state)
	if err != nil {
		return nil, fmt.Errorf("构建授权URL失败: %v", err)
	}

	return &systemRes.OidcLoginResponse{
		AuthURL: authURL,
		State:   state,
	}, nil
}

func (o *OidcService) HandleCallback(req systemReq.OidcCallbackRequest) (*system.SysUser, error) {
	// 验证状态参数
	ctx := context.Background()
	storage := o.GetStorage()
	provider, err := storage.Get(ctx, fmt.Sprintf("oidc_state:%s", req.State))
	if err != nil {
		provider = "clinx"
	}

	// 删除状态参数
	storage.Del(ctx, fmt.Sprintf("oidc_state:%s", req.State))

	config := o.getOidcConfig(provider)
	if config == nil {
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}

	// 交换授权码获取令牌
	tokenResp, err := o.exchangeCodeForToken(config, req.Code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %v", err)
	}

	// 获取用户信息
	userInfo, err := o.getUserInfo(config, tokenResp.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %v", err)
	}

	// 查找或创建用户
	user, err := o.findOrCreateUser(provider, userInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to find or create user: %v", err)
	}

	return user, nil
}

// getOidcConfig 获取OIDC配置
func (o *OidcService) getOidcConfig(provider string) *OidcConfig {
	oidcConfig := global.GVA_CONFIG.OIDC

	if oidcConfig.Provider != provider {
		return nil
	}

	config := &OidcConfig{
		ClientID:     oidcConfig.ClientID,
		ClientSecret: oidcConfig.ClientSecret,
		RedirectURL:  oidcConfig.RedirectURL,
		Scopes:       oidcConfig.Scopes,
		DiscoveryURL: oidcConfig.DiscoveryURL,
	}

	// 如果有发现端点，则自动获取配置
	if oidcConfig.DiscoveryURL != "" {
		discoveryConfig, err := o.fetchDiscoveryConfig(oidcConfig.DiscoveryURL)
		if err != nil {
			// 记录错误但不阻止流程，使用手动配置的URL
			fmt.Printf("Failed to fetch discovery config: %v\n", err)
		} else {
			config.AuthURL = discoveryConfig.AuthURL
			config.TokenURL = discoveryConfig.TokenURL
			config.UserInfoURL = discoveryConfig.UserInfoURL
			config.EndSessionURL = discoveryConfig.EndSessionURL
		}
	} else {
		// 使用手动配置的URL
		config.AuthURL = oidcConfig.AuthURL
		config.TokenURL = oidcConfig.TokenURL
		config.UserInfoURL = oidcConfig.UserInfoURL
		config.EndSessionURL = oidcConfig.EndSessionURL
	}

	return config
}

// fetchDiscoveryConfig 获取OIDC发现配置
func (o *OidcService) fetchDiscoveryConfig(discoveryURL string) (*OidcDiscoveryResponse, error) {
	client := &http.Client{Timeout: 10 * time.Second}

	resp, err := client.Get(discoveryURL)
	if err != nil {
		return nil, fmt.Errorf("无法获取OIDC发现配置: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("OIDC服务提供方返回错误状态码 %d: %s", resp.StatusCode, string(body))
	}

	var discoveryResp OidcDiscoveryResponse
	if err := json.NewDecoder(resp.Body).Decode(&discoveryResp); err != nil {
		return nil, fmt.Errorf("解析OIDC发现配置失败: %v", err)
	}

	return &discoveryResp, nil
}

// generateState 生成状态参数
func (o *OidcService) generateState() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}

// buildAuthURL 构建授权URL
func (o *OidcService) buildAuthURL(config *OidcConfig, state string) (string, error) {
	params := url.Values{}
	params.Add("response_type", "code")
	params.Add("client_id", config.ClientID)
	params.Add("redirect_uri", config.RedirectURL)
	params.Add("scope", config.Scopes)
	params.Add("state", state)

	// 生成nonce参数用于防止重放攻击
	nonce := uuid.New().String()
	params.Add("nonce", nonce)

	// 存储nonce（10分钟过期）
	ctx := context.Background()
	storage := o.GetStorage()
	err := storage.Set(ctx, fmt.Sprintf("oidc_nonce:%s", nonce), state, 10*time.Minute)
	if err != nil {
		return "", fmt.Errorf("failed to store nonce: %v", err)
	}

	// 构建URL，确保参数正确编码但不重复编码
	authURL, err := url.Parse(config.AuthURL)
	if err != nil {
		return "", fmt.Errorf("invalid auth URL: %v", err)
	}

	// 合并查询参数
	if authURL.RawQuery != "" {
		authURL.RawQuery += "&" + params.Encode()
	} else {
		authURL.RawQuery = params.Encode()
	}

	return authURL.String(), nil
}

// exchangeCodeForToken 交换授权码获取令牌
func (o *OidcService) exchangeCodeForToken(config *OidcConfig, code string) (*OidcTokenResponse, error) {
	data := url.Values{}
	data.Set("grant_type", "authorization_code")
	data.Set("code", code)
	data.Set("redirect_uri", config.RedirectURL)
	data.Set("client_id", config.ClientID)
	data.Set("client_secret", config.ClientSecret)

	resp, err := http.PostForm(config.TokenURL, data)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("token request failed: %s", string(body))
	}

	var tokenResp OidcTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return nil, err
	}

	return &tokenResp, nil
}

// getUserInfo 获取用户信息
func (o *OidcService) getUserInfo(config *OidcConfig, accessToken string) (*OidcUserInfo, error) {
	req, err := http.NewRequest("GET", config.UserInfoURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("userinfo request failed: %s", string(body))
	}

	var userInfo OidcUserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, err
	}

	return &userInfo, nil
}

// findOrCreateUser 查找或创建用户
func (o *OidcService) findOrCreateUser(provider string, userInfo *OidcUserInfo) (*system.SysUser, error) {
	getUsername := func() string {
		if userInfo.PreferredUsername != "" {
			return userInfo.PreferredUsername
		}
		if userInfo.Username != "" {
			return userInfo.Username
		}
		if userInfo.Email != "" {
			return userInfo.Email
		}
		return userInfo.Sub
	}

	getNickname := func() string {
		if userInfo.Nickname != "" {
			return userInfo.Nickname
		}
		if userInfo.Name != "" {
			return userInfo.Name
		}
		if userInfo.PreferredUsername != "" {
			return userInfo.PreferredUsername
		}
		if len(userInfo.Sub) > 8 {
			return userInfo.Sub[:8]
		}
		return userInfo.Sub
	}

	var oidcUser system.SysOidcUser
	if err := global.GVA_DB.Where("provider = ? AND subject = ?", provider, userInfo.Sub).First(&oidcUser).Error; err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("database error when finding OIDC user: %v", err)
		}

		if !global.GVA_CONFIG.OIDC.AutoCreateUser {
			return nil, fmt.Errorf("user not found and auto-create is disabled")
		}

		user := system.SysUser{
			UUID:      uuid.New(),
			Username:  getUsername(),
			NickName:  getNickname(),
			HeaderImg: userInfo.Picture,
			Enable:    1,
			Password:  uuid.New().String()[:16],
		}

		user.AuthorityId = global.GVA_CONFIG.OIDC.DefaultAuthority

		if err := global.GVA_DB.Create(&user).Error; err != nil {
			return nil, err
		}

		oidcUser = system.SysOidcUser{
			Username: user.Username,
			Email:    userInfo.Email,
			Nickname: user.NickName,
			Avatar:   userInfo.Picture,
			Provider: provider,
			Subject:  userInfo.Sub,
			UserID:   user.ID,
		}

		if err := global.GVA_DB.Create(&oidcUser).Error; err != nil {
			global.GVA_DB.Delete(&user)
			return nil, err
		}

		if err := global.GVA_DB.Preload("Authorities").Preload("Authority").First(&user, user.ID).Error; err != nil {
			return nil, err
		}

		return &user, nil
	}

	user := system.SysUser{GVA_MODEL: global.GVA_MODEL{ID: oidcUser.UserID}}
	if err := global.GVA_DB.Preload("Authorities").Preload("Authority").First(&user).Error; err != nil {
		return nil, err
	}

	oidcUser.Username = getUsername()
	oidcUser.Email = userInfo.Email
	oidcUser.Nickname = getNickname()
	oidcUser.Avatar = userInfo.Picture

	if err := global.GVA_DB.Save(&oidcUser).Error; err != nil {
		return nil, err
	}

	return &user, nil
}

// UnlinkOidc 解绑OIDC
func (o *OidcService) UnlinkOidc(userID uint, provider string) error {
	return global.GVA_DB.Where("user_id = ? AND provider = ?", userID, provider).Delete(&system.SysOidcUser{}).Error
}

// GetOidcUsers 获取用户的OIDC绑定
func (o *OidcService) GetOidcUsers(userID uint) ([]system.SysOidcUser, error) {
	var oidcUsers []system.SysOidcUser
	err := global.GVA_DB.Preload("SysUser").Where("user_id = ?", userID).Find(&oidcUsers).Error
	return oidcUsers, err
}

// GetLogoutURL 获取OIDC登出URL
func (o *OidcService) GetLogoutURL(postLogoutRedirectURI string) (*systemRes.OidcLogoutResponse, error) {
	if !global.GVA_CONFIG.OIDC.Enabled {
		return &systemRes.OidcLogoutResponse{
			Enabled:   false,
			LogoutURL: "",
		}, nil
	}

	provider := global.GVA_CONFIG.OIDC.Provider
	config := o.getOidcConfig(provider)
	if config == nil || config.EndSessionURL == "" {
		return &systemRes.OidcLogoutResponse{
			Enabled:   false,
			LogoutURL: "",
		}, nil // 没有配置登出端点，返回未启用
	}

	// 构建登出URL
	params := url.Values{}
	if postLogoutRedirectURI != "" {
		params.Add("post_logout_redirect_uri", postLogoutRedirectURI)
	}

	logoutURL, err := url.Parse(config.EndSessionURL)
	if err != nil {
		return nil, fmt.Errorf("invalid end session URL: %v", err)
	}

	if params.Encode() != "" {
		if logoutURL.RawQuery != "" {
			logoutURL.RawQuery += "&" + params.Encode()
		} else {
			logoutURL.RawQuery = params.Encode()
		}
	}

	return &systemRes.OidcLogoutResponse{
		Enabled:   true,
		LogoutURL: logoutURL.String(),
	}, nil
}

// testOIDCConnection 测试OIDC服务提供方连接
func (o *OidcService) testOIDCConnection(discoveryURL string) error {
	client := &http.Client{Timeout: 5 * time.Second}

	resp, err := client.Get(discoveryURL)
	if err != nil {
		return fmt.Errorf("无法连接到OIDC服务提供方: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("OIDC服务提供方返回错误状态码: %d", resp.StatusCode)
	}

	return nil
}
