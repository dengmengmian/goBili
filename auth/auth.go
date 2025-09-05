package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/skip2/go-qrcode"
)

// AuthManager handles Bilibili authentication
type AuthManager struct {
	cookies   map[string]string
	userAgent string
	client    *http.Client
	logger    *logrus.Logger
	configDir string
}

// UserInfo represents user information
type UserInfo struct {
	Mid       int64  `json:"mid"`
	Name      string `json:"name"`
	Face      string `json:"face"`
	Sign      string `json:"sign"`
	Level     int    `json:"level"`
	VipStatus int    `json:"vip_status"`
}

// QRCodeInfo represents QR code login information
type QRCodeInfo struct {
	URL       string `json:"url"`
	OAuthKey  string `json:"oauthKey"`
	QRCodeURL string `json:"qrCodeUrl"`
}

// QRCodeStatus represents QR code scan status
type QRCodeStatus struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    struct {
		URL          string `json:"url"`
		RefreshToken string `json:"refresh_token"`
		Timestamp    int64  `json:"timestamp"`
		Code         int    `json:"code"`
		Message      string `json:"message"`
	} `json:"data"`
}

// NewAuthManager creates a new authentication manager
func NewAuthManager(configDir string, logger *logrus.Logger) *AuthManager {
	return &AuthManager{
		cookies:   make(map[string]string),
		userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		logger:    logger,
		configDir: configDir,
	}
}

// LoadCookies loads cookies from file
func (am *AuthManager) LoadCookies() error {
	cookieFile := filepath.Join(am.configDir, "cookies.json")

	if _, err := os.Stat(cookieFile); os.IsNotExist(err) {
		am.logger.Info("No cookie file found, starting without authentication")
		return nil
	}

	data, err := os.ReadFile(cookieFile)
	if err != nil {
		return fmt.Errorf("failed to read cookie file: %v", err)
	}

	if err := json.Unmarshal(data, &am.cookies); err != nil {
		return fmt.Errorf("failed to parse cookie file: %v", err)
	}

	am.logger.Info("Loaded cookies from file")
	return nil
}

// SaveCookies saves cookies to file
func (am *AuthManager) SaveCookies() error {
	if err := os.MkdirAll(am.configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %v", err)
	}

	cookieFile := filepath.Join(am.configDir, "cookies.json")
	data, err := json.MarshalIndent(am.cookies, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cookies: %v", err)
	}

	if err := os.WriteFile(cookieFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write cookie file: %v", err)
	}

	am.logger.Info("Saved cookies to file")
	return nil
}

// SetCookie sets a cookie
func (am *AuthManager) SetCookie(name, value string) {
	am.cookies[name] = value
}

// GetCookie gets a cookie value
func (am *AuthManager) GetCookie(name string) string {
	return am.cookies[name]
}

// ClearCookies clears all cookies from memory
func (am *AuthManager) ClearCookies() {
	am.cookies = make(map[string]string)
}

// SetCookiesFromString parses and sets cookies from a cookie string
func (am *AuthManager) SetCookiesFromString(cookieStr string) error {
	cookies := strings.Split(cookieStr, ";")
	for _, cookie := range cookies {
		cookie = strings.TrimSpace(cookie)
		if cookie == "" {
			continue
		}

		parts := strings.SplitN(cookie, "=", 2)
		if len(parts) != 2 {
			continue
		}

		am.cookies[strings.TrimSpace(parts[0])] = strings.TrimSpace(parts[1])
	}

	am.logger.Info("Set cookies from string")
	return nil
}

// GetUserInfo gets current user information
func (am *AuthManager) GetUserInfo() (*UserInfo, error) {
	req, err := http.NewRequest("GET", "https://api.bilibili.com/x/space/myinfo", nil)
	if err != nil {
		return nil, err
	}

	am.setHeaders(req)

	resp, err := am.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var apiResp struct {
		Code int      `json:"code"`
		Data UserInfo `json:"data"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, err
	}

	if apiResp.Code != 0 {
		return nil, fmt.Errorf("API error: %d", apiResp.Code)
	}

	return &apiResp.Data, nil
}

// GenerateQRCode generates QR code for login
func (am *AuthManager) GenerateQRCode() (*QRCodeInfo, error) {
	req, err := http.NewRequest("GET", "https://passport.bilibili.com/x/passport-login/web/qrcode/generate", nil)
	if err != nil {
		return nil, err
	}

	am.setHeaders(req)

	resp, err := am.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var apiResp struct {
		Code int `json:"code"`
		Data struct {
			URL       string `json:"url"`
			OAuthKey  string `json:"oauthKey"`
			QRCodeKey string `json:"qrcode_key"`
			QRCodeURL string `json:"qrCodeUrl"`
		} `json:"data"`
	}

	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, err
	}

	// Debug: log the API response
	am.logger.Debugf("QR Code API Response: %s", string(body))

	if apiResp.Code != 0 {
		return nil, fmt.Errorf("failed to generate QR code: %d", apiResp.Code)
	}

	// Use qrcode_key if oauthKey is empty (new API format)
	oauthKey := apiResp.Data.OAuthKey
	if oauthKey == "" {
		oauthKey = apiResp.Data.QRCodeKey
	}

	return &QRCodeInfo{
		URL:       apiResp.Data.URL,
		OAuthKey:  oauthKey,
		QRCodeURL: apiResp.Data.URL, // Use URL as QRCodeURL since API doesn't provide separate QRCodeURL
	}, nil
}

// CheckQRCodeStatus checks QR code scan status
func (am *AuthManager) CheckQRCodeStatus(oauthKey string) (*QRCodeStatus, error) {
	req, err := http.NewRequest("GET", "https://passport.bilibili.com/x/passport-login/web/qrcode/poll", nil)
	if err != nil {
		return nil, err
	}

	q := req.URL.Query()
	q.Add("qrcode_key", oauthKey)
	req.URL.RawQuery = q.Encode()

	am.setHeaders(req)

	resp, err := am.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var status QRCodeStatus
	if err := json.Unmarshal(body, &status); err != nil {
		return nil, err
	}

	return &status, nil
}

// LoginWithQRCode performs QR code login
func (am *AuthManager) LoginWithQRCode() error {
	// Generate QR code
	qrInfo, err := am.GenerateQRCode()
	if err != nil {
		return fmt.Errorf("failed to generate QR code: %v", err)
	}

	fmt.Printf("请使用B站APP扫描二维码登录:\n")
	fmt.Printf("二维码链接: %s\n", qrInfo.QRCodeURL)
	fmt.Printf("或者访问: %s\n", qrInfo.URL)

	// Display QR code in terminal
	if qrInfo.QRCodeURL != "" {
		fmt.Println("\n=== 二维码 ===")
		if err := displayQRCode(qrInfo.QRCodeURL); err != nil {
			am.logger.Warnf("Failed to display QR code: %v", err)
			fmt.Println("无法在终端显示二维码，请使用上面的链接")
		}
		fmt.Println("=== 二维码 ===")
	}

	fmt.Println("\n等待扫描...")

	// Poll for scan status
	for {
		status, err := am.CheckQRCodeStatus(qrInfo.OAuthKey)
		if err != nil {
			return fmt.Errorf("failed to check QR code status: %v", err)
		}

		switch status.Data.Code {
		case 0:
			// Success
			fmt.Println("登录成功!")

			// Parse cookies from the redirect URL
			if err := am.parseCookiesFromURL(status.Data.URL); err != nil {
				return fmt.Errorf("failed to parse cookies: %v", err)
			}

			// Save cookies
			if err := am.SaveCookies(); err != nil {
				am.logger.Warnf("Failed to save cookies: %v", err)
			}

			return nil
		case 86101:
			// Not scanned
			fmt.Print(".")
			time.Sleep(2 * time.Second)
			continue
		case 86090:
			// Scanned but not confirmed
			fmt.Println("\n二维码已扫描，请在手机上确认登录")
			time.Sleep(2 * time.Second)
			continue
		case 86038:
			// Expired
			return fmt.Errorf("二维码已过期，请重新生成")
		default:
			return fmt.Errorf("登录失败: %s", status.Data.Message)
		}
	}
}

// parseCookiesFromURL parses cookies from redirect URL
func (am *AuthManager) parseCookiesFromURL(redirectURL string) error {
	u, err := url.Parse(redirectURL)
	if err != nil {
		return err
	}

	// Extract cookies from URL parameters
	params := u.Query()

	// Common Bilibili cookies
	cookieNames := []string{"SESSDATA", "bili_jct", "DedeUserID", "DedeUserID__ckMd5", "sid"}

	for _, name := range cookieNames {
		if value := params.Get(name); value != "" {
			am.cookies[name] = value
		}
	}

	return nil
}

// IsAuthenticated checks if user is authenticated
func (am *AuthManager) IsAuthenticated() bool {
	// Check if we have essential cookies
	essentialCookies := []string{"SESSDATA", "bili_jct"}
	for _, cookie := range essentialCookies {
		if am.cookies[cookie] == "" {
			return false
		}
	}
	return true
}

// setHeaders sets common headers for requests
func (am *AuthManager) setHeaders(req *http.Request) {
	req.Header.Set("User-Agent", am.userAgent)
	req.Header.Set("Referer", "https://www.bilibili.com/")
	req.Header.Set("Origin", "https://www.bilibili.com")

	// Add cookies
	var cookieParts []string
	for name, value := range am.cookies {
		cookieParts = append(cookieParts, fmt.Sprintf("%s=%s", name, value))
	}
	if len(cookieParts) > 0 {
		req.Header.Set("Cookie", strings.Join(cookieParts, "; "))
	}
}

// GetHTTPClient returns an HTTP client with authentication headers
func (am *AuthManager) GetHTTPClient() *http.Client {
	return am.client
}

// CreateAuthenticatedRequest creates an authenticated HTTP request
func (am *AuthManager) CreateAuthenticatedRequest(method, url string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	am.setHeaders(req)
	return req, nil
}

// displayQRCode displays QR code in terminal
func displayQRCode(url string) error {
	// Generate QR code with low error correction for smaller size
	qr, err := qrcode.New(url, qrcode.Low)
	if err != nil {
		return err
	}

	// Get QR code as ASCII art with smaller size
	ascii := qr.ToSmallString(false)
	fmt.Print(ascii)

	return nil
}
