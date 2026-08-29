package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

// UserRole defines access level.
type UserRole string

const (
	RoleOwner    UserRole = "owner"
	RoleDirector UserRole = "director"
	RoleAdmin    UserRole = "admin"
	RoleOperator UserRole = "operator"
	RoleViewer   UserRole = "viewer"
	RoleReporter UserRole = "reporter"
	RoleCustom   UserRole = "custom"
)

type Permission string

const (
	PermissionViewDashboard      Permission = "dashboard:view"
	PermissionViewServers        Permission = "servers:view"
	PermissionManageServers      Permission = "servers:manage"
	PermissionDeploy             Permission = "deploy:execute"
	PermissionViewSensitive      Permission = "sensitive:view"
	PermissionManageUsers        Permission = "users:manage"
	PermissionViewAudit          Permission = "audit:view"
	PermissionManageTickets      Permission = "tickets:manage"
	PermissionCreateTickets      Permission = "tickets:create"
	PermissionManageGroups       Permission = "groups:manage"
	PermissionViewSecurity       Permission = "security:view"
	PermissionManageSecurity     Permission = "security:manage"
	PermissionViewDiagnostics    Permission = "diagnostics:view"
	PermissionExecuteRemediation Permission = "remediation:execute"
	PermissionManageAdapters     Permission = "adapters:manage"
)

// User represents a console user.
type User struct {
	Username          string       `json:"username"`
	PasswordHash      string       `json:"password_hash"` // bcrypt hash
	APIKeyHash        string       `json:"api_key_hash"`  // SHA256 of API key (for 2FA)
	Role              UserRole     `json:"role"`
	CustomPermissions []Permission `json:"custom_permissions,omitempty"`
	CreatedAt         time.Time    `json:"created_at"`
	LastLogin         time.Time    `json:"last_login"`
	PasswordChangedAt time.Time    `json:"password_changed_at"` // zero = first login, must change password
}

// LoginState tracks failed attempts and lockouts.
type LoginState struct {
	FailedAttempts int       `json:"failed_attempts"`
	Locked         bool      `json:"locked"`
	LockedAt       time.Time `json:"locked_at"`
	LockReason     string    `json:"lock_reason"`
}

// AuthManager handles authentication, users, and lockouts.
type AuthManager struct {
	mu       sync.RWMutex
	users    map[string]*User
	logins   map[string]*LoginState
	dataPath string

	// Config
	MaxFailedAttempts int
	LockDuration      time.Duration // 0 = permanent (manual unlock only)
	TokenTTL          time.Duration
}

// NewAuthManager creates an auth manager.
func NewAuthManager(dataPath string) *AuthManager {
	return &AuthManager{
		users:             make(map[string]*User),
		logins:            make(map[string]*LoginState),
		dataPath:          dataPath,
		MaxFailedAttempts: 3,
		LockDuration:      0, // permanent lock by default (manual unlock only)
		TokenTTL:          8 * time.Hour,
	}
}

// --- Password / Key hashing ---

// HashPassword hashes a password using bcrypt.
func HashPassword(password string) (string, error) {
	if err := ValidatePassword(password, ""); err != nil {
		return "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// CheckPassword compares password with bcrypt hash.
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// ValidatePassword enforces a policy suitable for privileged console access.
func ValidatePassword(password, username string) error {
	if len([]byte(password)) < 12 {
		return errors.New("密码至少需要 12 个字符")
	}
	if username != "" && strings.EqualFold(password, username) {
		return errors.New("密码不能与用户名相同")
	}
	var upper, lower, digit, special bool
	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			upper = true
		case unicode.IsLower(char):
			lower = true
		case unicode.IsDigit(char):
			digit = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			special = true
		}
	}
	if !upper || !lower || !digit || !special {
		return errors.New("密码必须包含大写字母、小写字母、数字和特殊字符")
	}
	return nil
}

// HashAPIKey hashes an API key using SHA256.
func HashAPIKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// GenerateAPIKey creates a random 32-byte API key.
func GenerateAPIKey() (string, string, error) {
	// raw key = 32 bytes hex = 64 chars
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", "", err
	}
	key := hex.EncodeToString(buf)
	hash := HashAPIKey(key)
	return key, hash, nil
}

// --- User management ---

// CreateUser adds a new user. Returns the raw API key (shown only once).
func (a *AuthManager) CreateUser(username, password string, role UserRole) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if username == "" || password == "" {
		return "", errors.New("用户名和密码不能为空")
	}
	if err := ValidatePassword(password, username); err != nil {
		return "", err
	}
	if _, exists := a.users[username]; exists {
		return "", errors.New("用户已存在")
	}
	if !ValidRole(role) {
		role = RoleViewer
	}

	pwHash, err := HashPassword(password)
	if err != nil {
		return "", err
	}

	apiKey, apiHash, err := GenerateAPIKey()
	if err != nil {
		return "", err
	}

	a.users[username] = &User{
		Username:     username,
		PasswordHash: pwHash,
		APIKeyHash:   apiHash,
		Role:         role,
		CreatedAt:    time.Now(),
	}
	a.logins[username] = &LoginState{}

	a.save()
	return apiKey, nil
}

func ValidRole(role UserRole) bool {
	switch role {
	case RoleOwner, RoleDirector, RoleAdmin, RoleOperator, RoleViewer, RoleReporter, RoleCustom:
		return true
	default:
		return false
	}
}

func HasPermission(role UserRole, permission Permission) bool {
	if role == RoleOwner || role == RoleAdmin {
		return true
	}
	switch role {
	case RoleDirector:
		return permission == PermissionViewDashboard || permission == PermissionViewServers || permission == PermissionManageServers || permission == PermissionDeploy || permission == PermissionViewSensitive || permission == PermissionViewAudit || permission == PermissionManageTickets || permission == PermissionCreateTickets || permission == PermissionManageGroups || permission == PermissionManageUsers || permission == PermissionViewSecurity || permission == PermissionManageSecurity || permission == PermissionViewDiagnostics || permission == PermissionExecuteRemediation || permission == PermissionManageAdapters
	case RoleOperator:
		return permission == PermissionViewDashboard || permission == PermissionViewServers || permission == PermissionManageServers || permission == PermissionDeploy || permission == PermissionCreateTickets || permission == PermissionManageTickets || permission == PermissionViewDiagnostics
	case RoleViewer:
		return permission == PermissionViewDashboard || permission == PermissionViewServers || permission == PermissionCreateTickets || permission == PermissionViewDiagnostics
	case RoleReporter:
		return permission == PermissionViewDashboard || permission == PermissionCreateTickets
	default:
		return false
	}
}

func (a *AuthManager) CustomPermissions(username string) []Permission {
	a.mu.RLock()
	defer a.mu.RUnlock()
	user, ok := a.users[username]
	if !ok {
		return nil
	}
	return append([]Permission(nil), user.CustomPermissions...)
}

func (a *AuthManager) SetCustomPermissions(username string, permissions []Permission) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	user, ok := a.users[username]
	if !ok {
		return errors.New("用户不存在")
	}
	if user.Role != RoleCustom {
		return errors.New("只有自定义权限组用户可以设置权限")
	}
	seen := make(map[Permission]bool)
	for _, permission := range permissions {
		if permission == PermissionManageUsers || permission == PermissionManageGroups || permission == PermissionViewSensitive || permission == PermissionExecuteRemediation || permission == PermissionManageAdapters {
			return errors.New("自定义组不能授予核心管理权限")
		}
		if !seen[permission] {
			seen[permission] = true
		}
	}
	user.CustomPermissions = append([]Permission(nil), permissions...)
	a.save()
	return nil
}

func HasCustomPermissions(role UserRole, permissions []Permission, permission Permission) bool {
	if role != RoleCustom {
		return HasPermission(role, permission)
	}
	for _, granted := range permissions {
		if granted == permission {
			return true
		}
	}
	return false
}

func CanManagePermissionGroups(role UserRole) bool {
	return role == RoleOwner || role == RoleDirector
}

func CanAssignRole(actor, target UserRole) bool {
	switch target {
	case RoleOwner:
		return actor == RoleOwner
	case RoleDirector, RoleCustom:
		return actor == RoleOwner || actor == RoleDirector
	default:
		return actor == RoleOwner || actor == RoleDirector || actor == RoleAdmin
	}
}

// DeleteUser removes a user.
func (a *AuthManager) DeleteUser(username string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.users[username]; !exists {
		return errors.New("用户不存在")
	}
	delete(a.users, username)
	delete(a.logins, username)
	a.save()
	return nil
}

// ChangePassword updates a user's password (admin reset). The password change
// timestamp is cleared so the user must change the password again on next login.
func (a *AuthManager) ChangePassword(username, newPassword string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	u, exists := a.users[username]
	if !exists {
		return errors.New("用户不存在")
	}
	if err := ValidatePassword(newPassword, username); err != nil {
		return err
	}

	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	u.PasswordHash = hash
	u.PasswordChangedAt = time.Time{}
	a.save()
	return nil
}

// ChangeOwnPassword lets a user change their own password after verifying the
// current password. Passwords changed this way are marked as changed.
func (a *AuthManager) ChangeOwnPassword(username, currentPassword, newPassword string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	u, exists := a.users[username]
	if !exists {
		return errors.New("用户不存在")
	}
	if !CheckPassword(currentPassword, u.PasswordHash) {
		return errors.New("当前密码错误")
	}
	if err := ValidatePassword(newPassword, username); err != nil {
		return err
	}

	hash, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	u.PasswordHash = hash
	u.PasswordChangedAt = time.Now()
	a.save()
	return nil
}

// RotateAPIKey generates a new API key for a user. Returns the raw key.
func (a *AuthManager) RotateAPIKey(username string) (string, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	u, exists := a.users[username]
	if !exists {
		return "", errors.New("用户不存在")
	}

	key, hash, err := GenerateAPIKey()
	if err != nil {
		return "", err
	}
	u.APIKeyHash = hash
	a.save()
	return key, nil
}

// NeedsPasswordChange reports whether the user must change their password
// (set at creation or after an admin password reset, cleared on self-service change).
func (a *AuthManager) NeedsPasswordChange(username string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	u, ok := a.users[username]
	if !ok {
		return false
	}
	return u.PasswordChangedAt.IsZero()
}

// UnlockUser unlocks a locked account (admin only).
func (a *AuthManager) UnlockUser(username, reason string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	ls, exists := a.logins[username]
	if !exists {
		return errors.New("用户不存在")
	}
	if !ls.Locked {
		return errors.New("用户未被锁定")
	}
	ls.Locked = false
	ls.FailedAttempts = 0
	ls.LockReason = "手动解锁: " + reason
	a.save()
	return nil
}

// GetUser returns a copy of the user data (no password hash).
func (a *AuthManager) GetUser(username string) (*User, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	u, ok := a.users[username]
	if !ok {
		return nil, false
	}
	// Return copy without sensitive data
	copy := *u
	copy.PasswordHash = "***"
	copy.APIKeyHash = "***"
	return &copy, true
}

// ListUsers returns all users (without sensitive data).
func (a *AuthManager) ListUsers() []User {
	a.mu.RLock()
	defer a.mu.RUnlock()
	result := make([]User, 0, len(a.users))
	for _, u := range a.users {
		copy := *u
		copy.PasswordHash = "***"
		copy.APIKeyHash = "***"
		result = append(result, copy)
	}
	return result
}

// GetLoginState returns login state for a user.
func (a *AuthManager) GetLoginState(username string) (*LoginState, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	ls, ok := a.logins[username]
	if !ok {
		return nil, false
	}
	copy := *ls
	return &copy, true
}

// --- Authentication ---

// LoginResult is returned from a login attempt.
type LoginResult struct {
	Success            bool
	Token              string
	User               *User
	Error              string
	Locked             bool
	Remaining          int // remaining attempts
	MustChangePassword bool // password changed at is zero (first login / admin reset)
}

// Login authenticates a user with password + API key (2FA).
func (a *AuthManager) Login(username, password, apiKey string) LoginResult {
	a.mu.Lock()
	defer a.mu.Unlock()

	result := LoginResult{}

	u, exists := a.users[username]
	if !exists {
		result.Error = "用户名或密码错误"
		return result
	}

	ls, ok := a.logins[username]
	if !ok {
		ls = &LoginState{}
		a.logins[username] = ls
	}

	// Check if already locked
	if ls.Locked {
		if a.LockDuration > 0 && time.Since(ls.LockedAt) > a.LockDuration {
			ls.Locked = false
			ls.FailedAttempts = 0
		} else {
			result.Locked = true
			result.Error = "账户已被锁定，请联系管理员解锁"
			return result
		}
	}

	// Verify password
	if !CheckPassword(password, u.PasswordHash) {
		ls.FailedAttempts++
		remaining := a.MaxFailedAttempts - ls.FailedAttempts
		result.Remaining = remaining
		if ls.FailedAttempts >= a.MaxFailedAttempts {
			ls.Locked = true
			ls.LockedAt = time.Now()
			ls.LockReason = "连续密码错误超过 " + string(rune('0'+a.MaxFailedAttempts)) + " 次"
			result.Locked = true
			result.Error = "连续错误次数过多，账户已被锁定"
		} else {
			result.Error = "用户名或密码错误"
		}
		a.save()
		return result
	}

	// Verify API key (2nd factor)
	if apiKey == "" || subtle.ConstantTimeCompare([]byte(HashAPIKey(apiKey)), []byte(u.APIKeyHash)) != 1 {
		ls.FailedAttempts++
		remaining := a.MaxFailedAttempts - ls.FailedAttempts
		result.Remaining = remaining
		if ls.FailedAttempts >= a.MaxFailedAttempts {
			ls.Locked = true
			ls.LockedAt = time.Now()
			ls.LockReason = "连续密钥错误超过限制"
			result.Locked = true
			result.Error = "连续密钥错误次数过多，账户已被锁定"
		} else {
			result.Error = "密钥错误"
		}
		a.save()
		return result
	}

	// Login success
	ls.FailedAttempts = 0
	u.LastLogin = time.Now()

	// Generate token
	token := GenerateToken(username, string(u.Role), a.TokenTTL)

	result.Success = true
	result.Token = token
	result.MustChangePassword = u.PasswordChangedAt.IsZero()
	userCopy := *u
	userCopy.PasswordHash = "***"
	userCopy.APIKeyHash = "***"
	result.User = &userCopy
	a.save()
	return result
}

var processTokenSecret = generateProcessTokenSecret()

// --- Token (simple signed JWT-like) ---

// GenerateToken creates a simple auth token.
func GenerateToken(username, role string, ttl time.Duration) string {
	// Simple token: base64(payload).hash
	// For production, use proper JWT. This is lightweight.
	expiry := time.Now().Add(ttl).Unix()
	payload := username + "|" + role + "|" + itoa(expiry)
	hash := sha256.Sum256([]byte(payload + tokenSecret()))
	return hex.EncodeToString([]byte(payload)) + "." + hex.EncodeToString(hash[:])
}

// ValidateToken checks a token and returns username and role.
func ValidateToken(token string) (string, string, bool) {
	parts := splitToken(token)
	if len(parts) != 2 {
		return "", "", false
	}
	payloadHex, hashHex := parts[0], parts[1]

	payloadBytes, err := hex.DecodeString(payloadHex)
	if err != nil {
		return "", "", false
	}
	payload := string(payloadBytes)

	// Verify hash
	expected := sha256.Sum256([]byte(payload + tokenSecret()))
	if hex.EncodeToString(expected[:]) != hashHex {
		return "", "", false
	}

	// Parse payload
	fields := splitPayload(payload)
	if len(fields) != 3 {
		return "", "", false
	}
	username := fields[0]
	role := fields[1]
	expiry := atoi(fields[2])

	if time.Now().Unix() > int64(expiry) {
		return "", "", false
	}

	return username, role, true
}

func tokenSecret() string {
	if secret := os.Getenv("CONSOLE_TOKEN_SECRET"); secret != "" {
		return secret
	}
	return processTokenSecret
}

func generateProcessTokenSecret() string {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		panic("无法生成控制台令牌密钥")
	}
	return hex.EncodeToString(buf)
}

func splitToken(s string) []string {
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			return []string{s[:i], s[i+1:]}
		}
	}
	return nil
}

func splitPayload(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '|' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		parts = append(parts, s[start:])
	}
	return parts
}

func itoa(n int64) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

func atoi(s string) int {
	n := 0
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			n = n*10 + int(s[i]-'0')
		}
	}
	return n
}

// --- Persistence (simple JSON) ---

type authData struct {
	Users  map[string]*User       `json:"users"`
	Logins map[string]*LoginState `json:"logins"`
}

// Load reads users and login state from JSON file.
func (a *AuthManager) Load() error {
	dataFile := filepath.Join(a.dataPath, "users.json")
	f, err := os.ReadFile(dataFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var data authData
	if err := json.Unmarshal(f, &data); err != nil {
		return err
	}
	if data.Users != nil {
		a.users = data.Users
	}
	if data.Logins != nil {
		a.logins = data.Logins
	}
	return nil
}

func (a *AuthManager) save() {
	dataFile := filepath.Join(a.dataPath, "users.json")
	os.MkdirAll(a.dataPath, 0750)

	data := authData{
		Users:  a.users,
		Logins: a.logins,
	}

	f, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return
	}
	os.WriteFile(dataFile, f, 0600)
}
