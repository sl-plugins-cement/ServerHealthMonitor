package web

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Oxen112774/ServerHealthMonitor/internal/console/audit"
	"github.com/Oxen112774/ServerHealthMonitor/internal/console/auth"
	"github.com/Oxen112774/ServerHealthMonitor/internal/console/servers"
	"github.com/Oxen112774/ServerHealthMonitor/internal/console/tickets"
)

// Handler provides all console HTTP endpoints.
type Handler struct {
	auth         *auth.AuthManager
	servers      *servers.Manager
	audit        *audit.Logger
	tickets      *tickets.Store
	dataDir      string
	loginLimiter *rateLimiter
	apiLimiter   *rateLimiter
}

// NewHandler creates a console web handler.
func NewHandler(authMgr *auth.AuthManager, svcMgr *servers.Manager, auditLog *audit.Logger, dataDir string, ticketStore *tickets.Store) *Handler {
	return &Handler{
		auth:         authMgr,
		servers:      svcMgr,
		audit:        auditLog,
		tickets:      ticketStore,
		dataDir:      dataDir,
		loginLimiter: newRateLimiter(10, time.Minute),  // 10 login attempts per minute per IP
		apiLimiter:   newRateLimiter(120, time.Minute), // 120 API requests per minute per IP
	}
}

// Register routes on the given mux.
func (h *Handler) Register(mux *http.ServeMux) {
	// Landing page (root)
	mux.HandleFunc("/", securityHeaders(h.handleLandingPage))

	// Public status page
	mux.HandleFunc("/public/status", securityHeaders(h.handlePublicStatusPage))
	mux.HandleFunc("/public/api/status", securityHeaders(h.handlePublicStatusAPI))

	// Connectivity test API (public but rate-limited)
	mux.HandleFunc("/public/api/check", securityHeaders(h.rateLimit(h.handleConnectivityCheckAPI, h.apiLimiter)))

	// Login page + API (rate-limited)
	mux.HandleFunc("/console/login", securityHeaders(h.handleLoginPage))
	mux.HandleFunc("/console/api/login", securityHeaders(maxBodySize(4096,
		h.rateLimit(h.handleLoginAPI, h.loginLimiter))))

	// Change password (self-service; first login / admin reset enforces it)
	mux.HandleFunc("/console/change-password", securityHeaders(h.authWrap(h.handleChangePasswordPage)))
	mux.HandleFunc("/console/api/change-password", securityHeaders(h.authWrap(maxBodySize(4096,
		h.rateLimit(h.handleChangePasswordAPI, h.apiLimiter)))))

	// Protected pages
	mux.HandleFunc("/console/", securityHeaders(h.authWrap(h.handleConsolePage)))
	mux.HandleFunc("/console/dashboard", securityHeaders(h.authWrap(h.handleDashboardPage)))
	mux.HandleFunc("/console/servers", securityHeaders(h.authWrap(h.handleServersPage)))
	mux.HandleFunc("/console/users", securityHeaders(h.permissionWrap(auth.PermissionManageUsers, h.handleUsersPage)))
	mux.HandleFunc("/console/audit", securityHeaders(h.permissionWrap(auth.PermissionViewAudit, h.handleAuditPage)))
	mux.HandleFunc("/console/tools", securityHeaders(h.authWrap(h.handleToolsPage)))
	mux.HandleFunc("/console/tickets", securityHeaders(h.authWrap(h.handleTicketsPage)))
	mux.HandleFunc("/console/api/diagnostics", securityHeaders(h.authWrap(h.rateLimit(h.handleDiagnosticsAPI, h.apiLimiter))))

	// Protected APIs (rate-limited)
	mux.HandleFunc("/console/api/me", securityHeaders(h.authWrap(h.rateLimit(h.handleMeAPI, h.apiLimiter))))
	mux.HandleFunc("/console/api/logout", securityHeaders(h.authWrap(h.handleLogoutAPI)))
	mux.HandleFunc("/console/api/servers", securityHeaders(h.permissionWrap(auth.PermissionViewServers, h.rateLimit(h.handleServersAPI, h.apiLimiter))))
	mux.HandleFunc("/console/api/servers/", securityHeaders(h.permissionWrap(auth.PermissionViewServers, h.rateLimit(h.handleServerDetailAPI, h.apiLimiter))))
	mux.HandleFunc("/console/api/deploy", securityHeaders(h.permissionWrap(auth.PermissionDeploy, h.rateLimit(h.handleDeployAPI, h.apiLimiter))))
	mux.HandleFunc("/console/api/users", securityHeaders(h.permissionWrap(auth.PermissionManageUsers, h.rateLimit(h.handleUsersAPI, h.apiLimiter))))
	mux.HandleFunc("/console/api/users/", securityHeaders(h.permissionWrap(auth.PermissionManageUsers, h.rateLimit(h.handleUserDetailAPI, h.apiLimiter))))
	mux.HandleFunc("/console/api/audit", securityHeaders(h.permissionWrap(auth.PermissionViewAudit, h.rateLimit(h.handleAuditAPI, h.apiLimiter))))
	mux.HandleFunc("/console/api/check", securityHeaders(h.authWrap(h.rateLimit(h.handleConnectivityCheckAPI, h.apiLimiter))))
	mux.HandleFunc("/console/api/tickets", securityHeaders(h.permissionWrap(auth.PermissionCreateTickets, h.rateLimit(h.handleTicketsAPI, h.apiLimiter))))
	mux.HandleFunc("/console/api/tickets/", securityHeaders(h.permissionWrap(auth.PermissionCreateTickets, h.rateLimit(h.handleTicketDetailAPI, h.apiLimiter))))
}

// --- Rate limit wrapper ---

func (h *Handler) rateLimit(next http.HandlerFunc, rl *rateLimiter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := getIP(r)
		if !rl.allow(ip) {
			h.jsonError(w, "请求过于频繁，请稍后再试", http.StatusTooManyRequests)
			return
		}
		next(w, r)
	}
}

// --- Landing page ---

func (h *Handler) handleLandingPage(w http.ResponseWriter, r *http.Request) {
	// Only serve landing page on exact "/" path
	if r.URL.Path != "/" {
		// Check if it's a console path first
		if strings.HasPrefix(r.URL.Path, "/console/") || strings.HasPrefix(r.URL.Path, "/public/") {
			http.NotFound(w, r)
			return
		}
		// Redirect to landing page
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(landingPageHTML))
}

// --- Public status page ---

func (h *Handler) handlePublicStatusPage(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(publicStatusHTML))
}

func (h *Handler) handlePublicStatusAPI(w http.ResponseWriter, r *http.Request) {
	serversList := h.servers.ListServers()

	// Return only public info (no credentials, no keys)
	publicServers := make([]map[string]interface{}, 0, len(serversList))
	onlineCount := 0
	for _, s := range serversList {
		isOnline := s.Status == "online"
		if isOnline {
			onlineCount++
		}
		publicServers = append(publicServers, map[string]interface{}{
			"name":   s.Name,
			"status": s.Status,
			"online": isOnline,
		})
	}

	h.jsonOK(w, map[string]interface{}{
		"total":   len(publicServers),
		"online":  onlineCount,
		"offline": len(publicServers) - onlineCount,
		"servers": publicServers,
		"updated": time.Now().Format("2006-01-02 15:04:05"),
	})
}

// --- Connectivity check API ---

func (h *Handler) handleConnectivityCheckAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.jsonError(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Host    string `json:"host"`
		Port    int    `json:"port"`
		Type    string `json:"type"`    // tcp / http / agent
		Timeout int    `json:"timeout"` // seconds
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.jsonError(w, "请求格式错误", http.StatusBadRequest)
		return
	}

	if req.Host == "" {
		h.jsonError(w, "主机地址不能为空", http.StatusBadRequest)
		return
	}
	if req.Port == 0 && req.Type != "http" {
		h.jsonError(w, "端口不能为空", http.StatusBadRequest)
		return
	}
	if req.Timeout == 0 {
		req.Timeout = 5
	}
	if req.Timeout > 30 {
		req.Timeout = 30
	}
	timeout := time.Duration(req.Timeout) * time.Second

	var result connResult

	switch req.Type {
	case "http":
		url := req.Host
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			url = "http://" + url
			if req.Port > 0 {
				url += ":" + strconv.Itoa(req.Port)
			}
		}
		result = checkHTTP(url, timeout)

	case "agent":
		url := "http://" + req.Host
		if req.Port > 0 {
			url += ":" + strconv.Itoa(req.Port)
		}
		url += "/api/health"
		result = checkHTTP(url, timeout)
		result.Type = "agent"

	default: // tcp
		result = checkTCP(req.Host, req.Port, timeout)
	}

	h.jsonOK(w, result)
}

// changePasswordPaths are the only endpoints reachable while a user still must
// change their initial/reset password.
var changePasswordPaths = map[string]bool{
	"/console/change-password":      true,
	"/console/api/change-password":  true,
	"/console/api/logout":           true,
}

// --- Auth middleware ---

func (h *Handler) authWrap(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, role, ok := h.getSession(r)
		if !ok {
			// API requests get JSON error, page requests redirect to login
			if strings.HasPrefix(r.URL.Path, "/console/api/") {
				h.jsonError(w, "未授权", http.StatusUnauthorized)
				return
			}
			http.Redirect(w, r, "/console/login", http.StatusSeeOther)
			return
		}

		// Enforce password change after first login / admin reset.
		if h.auth.NeedsPasswordChange(username) && !changePasswordPaths[r.URL.Path] {
			if strings.HasPrefix(r.URL.Path, "/console/api/") {
				h.jsonError(w, "首次登录必须先修改密码", http.StatusForbidden)
				return
			}
			http.Redirect(w, r, "/console/change-password", http.StatusSeeOther)
			return
		}

		// Set user context in header for handlers
		r.Header.Set("X-Username", username)
		r.Header.Set("X-Role", role)
		next(w, r)
	}
}

func (h *Handler) permissionWrap(permission auth.Permission, next http.HandlerFunc) http.HandlerFunc {
	return h.authWrap(func(w http.ResponseWriter, r *http.Request) {
		role := auth.UserRole(r.Header.Get("X-Role"))
		allowed := auth.HasPermission(role, permission)
		if role == auth.RoleCustom {
			allowed = auth.HasCustomPermissions(role, h.auth.CustomPermissions(r.Header.Get("X-Username")), permission)
		}
		if !allowed {
			if strings.HasPrefix(r.URL.Path, "/console/api/") {
				h.jsonError(w, "权限不足", http.StatusForbidden)
				return
			}
			http.Error(w, "403 权限不足", http.StatusForbidden)
			return
		}
		next(w, r)
	})
}

func (h *Handler) getSession(r *http.Request) (string, string, bool) {
	// Check cookie first
	cookie, err := r.Cookie("console_token")
	if err == nil && cookie.Value != "" {
		username, _, ok := auth.ValidateToken(cookie.Value)
		if ok {
			if user, exists := h.auth.GetUser(username); exists {
				return username, string(user.Role), true
			}
		}
	}

	// Check Authorization header
	authHeader := r.Header.Get("Authorization")
	if strings.HasPrefix(authHeader, "Bearer ") {
		token := strings.TrimPrefix(authHeader, "Bearer ")
		username, _, ok := auth.ValidateToken(token)
		if ok {
			if user, exists := h.auth.GetUser(username); exists {
				return username, string(user.Role), true
			}
		}
	}

	return "", "", false
}

func (h *Handler) getClientIP(r *http.Request) string {
	return getIP(r)
}

// --- JSON helpers ---

func (h *Handler) jsonOK(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"data":    data,
	})
}

func (h *Handler) jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success": false,
		"error":   msg,
	})
}
