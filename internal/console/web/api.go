package web

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/Oxen112774/ServerHealthMonitor/internal/console/audit"
	"github.com/Oxen112774/ServerHealthMonitor/internal/console/auth"
	"github.com/Oxen112774/ServerHealthMonitor/internal/console/servers"
	"github.com/Oxen112774/ServerHealthMonitor/internal/console/tickets"
	"github.com/Oxen112774/ServerHealthMonitor/internal/diagnostics"
)

// --- Login API ---

func (h *Handler) handleLoginAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.jsonError(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		Username string `json:"username"`
		Password string `json:"password"`
		APIKey   string `json:"api_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.jsonError(w, "请求格式错误", http.StatusBadRequest)
		return
	}

	ip := h.getClientIP(r)
	result := h.auth.Login(req.Username, req.Password, req.APIKey)

	if result.Success {
		h.audit.Log(req.Username, audit.ActionLogin, "", "登录成功", ip, true)

		// Set cookie
		http.SetCookie(w, &http.Cookie{
			Name:     "console_token",
			Value:    result.Token,
			Path:     "/console",
			HttpOnly: true,
			Secure:   cookieSecure(), // set CONSOLE_COOKIE_SECURE=1 behind HTTPS
			SameSite: http.SameSiteStrictMode,
			MaxAge:   int(8 * time.Hour / time.Second),
		})

		h.jsonOK(w, map[string]interface{}{
			"token":                result.Token,
			"user":                 result.User,
			"must_change_password": result.MustChangePassword,
		})
		return
	}

	// Login failed
	h.audit.Log(req.Username, audit.ActionLoginFailed, "", result.Error, ip, false)

	resp := map[string]interface{}{
		"error": result.Error,
	}
	if result.Locked {
		resp["locked"] = true
	} else {
		resp["remaining"] = result.Remaining
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusUnauthorized)
	json.NewEncoder(w).Encode(resp)
}

// --- Change password (self-service) ---

func (h *Handler) handleChangePasswordAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.jsonError(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		CurrentPassword string `json:"current_password"`
		NewPassword     string `json:"new_password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.jsonError(w, "请求格式错误", http.StatusBadRequest)
		return
	}

	username := r.Header.Get("X-Username")
	if err := h.auth.ChangeOwnPassword(username, req.CurrentPassword, req.NewPassword); err != nil {
		h.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}

	h.audit.Log(username, audit.ActionUserPassword, username, "用户修改自己的密码", h.getClientIP(r), true)
	h.jsonOK(w, nil)
}

func (h *Handler) handleDiagnosticsAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost && r.Method != http.MethodGet {
		h.jsonError(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	report, err := diagnostics.Collect(r.Context())
	if err != nil {
		h.jsonError(w, "诊断被取消", http.StatusRequestTimeout)
		return
	}
	username := r.Header.Get("X-Username")
	h.audit.Log(username, audit.ActionDiagnostics, report.Environment.Hostname, "生成通用环境诊断和风险评分", h.getClientIP(r), true)
	h.jsonOK(w, map[string]interface{}{
		"report": report,
		"plan":   diagnostics.BuildPlan(report),
	})
}

func (h *Handler) handleLogoutAPI(w http.ResponseWriter, r *http.Request) {
	username := r.Header.Get("X-Username")
	ip := h.getClientIP(r)

	// Clear cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "console_token",
		Value:    "",
		Path:     "/console",
		HttpOnly: true,
		MaxAge:   -1,
	})

	h.audit.Log(username, audit.ActionLogout, "", "用户登出", ip, true)
	h.jsonOK(w, nil)
}

func (h *Handler) handleMeAPI(w http.ResponseWriter, r *http.Request) {
	username := r.Header.Get("X-Username")
	role := r.Header.Get("X-Role")

	user, ok := h.auth.GetUser(username)
	if !ok {
		h.jsonError(w, "用户不存在", http.StatusNotFound)
		return
	}

	h.jsonOK(w, map[string]interface{}{
		"username": username,
		"role":     role,
		"user":     user,
	})
}

// --- Servers API ---

func (h *Handler) handleServersAPI(w http.ResponseWriter, r *http.Request) {
	username := r.Header.Get("X-Username")
	ip := h.getClientIP(r)
	role := auth.UserRole(r.Header.Get("X-Role"))
	if r.Method != http.MethodGet && !auth.HasPermission(role, auth.PermissionManageServers) {
		h.jsonError(w, "需要服务器管理权限", http.StatusForbidden)
		return
	}

	switch r.Method {
	case http.MethodGet:
		servers := h.servers.ListServers()
		result := make([]map[string]interface{}, 0, len(servers))
		for _, server := range servers {
			item := map[string]interface{}{"id": server.ID, "name": server.Name, "host": server.Host, "port": server.Port, "user": server.User, "agent_port": server.AgentPort, "status": server.Status, "last_check": server.LastCheck, "created_at": server.CreatedAt}
			if auth.HasPermission(role, auth.PermissionViewSensitive) {
				item["key_file"] = server.KeyFile
			}
			result = append(result, item)
		}
		h.jsonOK(w, result)

	case http.MethodPost:
		var s servers.Server
		if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
			h.jsonError(w, "请求格式错误", http.StatusBadRequest)
			return
		}
		result, err := h.servers.AddServer(s)
		if err != nil {
			h.jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		h.audit.Log(username, audit.ActionServerAdd, s.Name, "添加服务器: "+s.Host, ip, true)
		h.jsonOK(w, result)

	default:
		h.jsonError(w, "方法不允许", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleServerDetailAPI(w http.ResponseWriter, r *http.Request) {
	username := r.Header.Get("X-Username")
	ip := h.getClientIP(r)
	id := strings.TrimPrefix(r.URL.Path, "/console/api/servers/")
	if r.Method != http.MethodGet && !auth.HasPermission(auth.UserRole(r.Header.Get("X-Role")), auth.PermissionManageServers) {
		h.jsonError(w, "需要服务器管理权限", http.StatusForbidden)
		return
	}

	switch r.Method {
	case http.MethodGet:
		s, ok := h.servers.GetServer(id)
		if !ok {
			h.jsonError(w, "服务器不存在", http.StatusNotFound)
			return
		}
		result := map[string]interface{}{"id": s.ID, "name": s.Name, "host": s.Host, "port": s.Port, "user": s.User, "agent_port": s.AgentPort, "status": s.Status, "last_check": s.LastCheck, "created_at": s.CreatedAt}
		if auth.HasPermission(auth.UserRole(r.Header.Get("X-Role")), auth.PermissionViewSensitive) {
			result["key_file"] = s.KeyFile
		}
		h.jsonOK(w, result)

	case http.MethodPut:
		var s servers.Server
		if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
			h.jsonError(w, "请求格式错误", http.StatusBadRequest)
			return
		}
		result, err := h.servers.UpdateServer(id, s)
		if err != nil {
			h.jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		h.audit.Log(username, audit.ActionServerUpdate, s.Name, "更新服务器配置", ip, true)
		h.jsonOK(w, result)

	case http.MethodDelete:
		s, _ := h.servers.GetServer(id)
		err := h.servers.DeleteServer(id)
		if err != nil {
			h.jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		name := ""
		if s != nil {
			name = s.Name
		}
		h.audit.Log(username, audit.ActionServerDelete, name, "删除服务器", ip, true)
		h.jsonOK(w, nil)

	default:
		h.jsonError(w, "方法不允许", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleTicketsAPI(w http.ResponseWriter, r *http.Request) {
	username := r.Header.Get("X-Username")
	role := auth.UserRole(r.Header.Get("X-Role"))
	switch r.Method {
	case http.MethodGet:
		h.jsonOK(w, h.tickets.List(username, auth.HasPermission(role, auth.PermissionManageTickets)))
	case http.MethodPost:
		var req struct {
			Title       string `json:"title"`
			Description string `json:"description"`
			Priority    string `json:"priority"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.jsonError(w, "请求格式错误", http.StatusBadRequest)
			return
		}
		ticket, err := h.tickets.Create(req.Title, req.Description, req.Priority, username)
		if err != nil {
			h.jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		h.audit.Log(username, audit.ActionTicketCreate, ticket.ID, "提交工单", h.getClientIP(r), true)
		h.jsonOK(w, ticket)
	default:
		h.jsonError(w, "方法不允许", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleTicketDetailAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut || !auth.HasPermission(auth.UserRole(r.Header.Get("X-Role")), auth.PermissionManageTickets) {
		h.jsonError(w, "需要工单管理权限", http.StatusForbidden)
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/console/api/tickets/")
	var req struct {
		Status   tickets.Status `json:"status"`
		Assignee string         `json:"assignee"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.jsonError(w, "请求格式错误", http.StatusBadRequest)
		return
	}
	ticket, err := h.tickets.Update(id, req.Status, req.Assignee)
	if err != nil {
		h.jsonError(w, err.Error(), http.StatusBadRequest)
		return
	}
	h.audit.Log(r.Header.Get("X-Username"), audit.ActionTicketUpdate, id, "更新工单状态", h.getClientIP(r), true)
	h.jsonOK(w, ticket)
}

// --- Deploy API ---

func (h *Handler) handleDeployAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		h.jsonError(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	username := r.Header.Get("X-Username")
	ip := h.getClientIP(r)

	var req struct {
		ServerID   string `json:"server_id"`
		BinaryPath string `json:"binary_path"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.jsonError(w, "请求格式错误", http.StatusBadRequest)
		return
	}

	if req.ServerID == "" {
		h.jsonError(w, "请选择服务器", http.StatusBadRequest)
		return
	}

	srv, ok := h.servers.GetServer(req.ServerID)
	if !ok {
		h.jsonError(w, "服务器不存在", http.StatusNotFound)
		return
	}

	result, err := h.servers.DeployToServer(req.ServerID, username, req.BinaryPath)
	if err != nil {
		h.audit.Log(username, audit.ActionDeploy, srv.Name, "部署失败: "+err.Error(), ip, false)
		h.jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}

	h.audit.Log(username, audit.ActionDeploy, srv.Name, "部署成功", ip, true)
	h.jsonOK(w, result)
}

// --- Users API ---

func (h *Handler) handleUsersAPI(w http.ResponseWriter, r *http.Request) {
	username := r.Header.Get("X-Username")
	ip := h.getClientIP(r)

	switch r.Method {
	case http.MethodGet:
		users := h.auth.ListUsers()
		// Add login state
		result := make([]map[string]interface{}, len(users))
		for i, u := range users {
			ls, _ := h.auth.GetLoginState(u.Username)
			locked := false
			if ls != nil {
				locked = ls.Locked
			}
			result[i] = map[string]interface{}{
				"username":   u.Username,
				"role":       u.Role,
				"created_at": u.CreatedAt,
				"last_login": u.LastLogin,
				"locked":     locked,
				"failed_attempts": func() int {
					if ls != nil {
						return ls.FailedAttempts
					}
					return 0
				}(),
			}
		}
		h.jsonOK(w, result)

	case http.MethodPost:
		var req struct {
			Username    string            `json:"username"`
			Password    string            `json:"password"`
			Role        string            `json:"role"`
			Permissions []auth.Permission `json:"permissions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.jsonError(w, "请求格式错误", http.StatusBadRequest)
			return
		}

		requestedRole := auth.UserRole(req.Role)
		if requestedRole == auth.RoleCustom && !auth.CanManagePermissionGroups(auth.UserRole(r.Header.Get("X-Role"))) {
			h.jsonError(w, "只有服主或技术总监可以创建自定义权限组", http.StatusForbidden)
			return
		}
		apiKey, err := h.auth.CreateUser(req.Username, req.Password, requestedRole)
		if err != nil {
			h.jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		if requestedRole == auth.RoleCustom && len(req.Permissions) > 0 {
			if err := h.auth.SetCustomPermissions(req.Username, req.Permissions); err != nil {
				h.jsonError(w, err.Error(), http.StatusBadRequest)
				return
			}
		}

		h.audit.Log(username, audit.ActionUserCreate, req.Username, "创建用户，角色: "+req.Role, ip, true)
		h.jsonOK(w, map[string]interface{}{
			"username": req.Username,
			"api_key":  apiKey, // show only once
			"role":     req.Role,
		})

	default:
		h.jsonError(w, "方法不允许", http.StatusMethodNotAllowed)
	}
}

func (h *Handler) handleUserDetailAPI(w http.ResponseWriter, r *http.Request) {
	username := r.Header.Get("X-Username")
	ip := h.getClientIP(r)
	targetUser := strings.TrimPrefix(r.URL.Path, "/console/api/users/")

	switch r.Method {
	case http.MethodGet:
		user, ok := h.auth.GetUser(targetUser)
		if !ok {
			h.jsonError(w, "用户不存在", http.StatusNotFound)
			return
		}
		ls, _ := h.auth.GetLoginState(targetUser)
		h.jsonOK(w, map[string]interface{}{
			"user":        user,
			"login_state": ls,
		})

	case http.MethodDelete:
		if targetUser == username {
			h.jsonError(w, "不能删除自己", http.StatusBadRequest)
			return
		}
		err := h.auth.DeleteUser(targetUser)
		if err != nil {
			h.jsonError(w, err.Error(), http.StatusBadRequest)
			return
		}
		h.audit.Log(username, audit.ActionUserDelete, targetUser, "删除用户", ip, true)
		h.jsonOK(w, nil)

	case http.MethodPost:
		// Sub-actions: unlock, reset_password, rotate_key
		var req struct {
			Action      string            `json:"action"`
			NewPassword string            `json:"new_password"`
			Reason      string            `json:"reason"`
			Permissions []auth.Permission `json:"permissions"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			h.jsonError(w, "请求格式错误", http.StatusBadRequest)
			return
		}

		switch req.Action {
		case "set_permissions":
			if !auth.CanManagePermissionGroups(auth.UserRole(r.Header.Get("X-Role"))) {
				h.jsonError(w, "只有服主或技术总监可以修改自定义权限组", http.StatusForbidden)
				return
			}
			user, ok := h.auth.GetUser(targetUser)
			if !ok || user.Role != auth.RoleCustom {
				h.jsonError(w, "目标用户不是自定义权限组用户", http.StatusBadRequest)
				return
			}
			if err := h.auth.SetCustomPermissions(targetUser, req.Permissions); err != nil {
				h.jsonError(w, err.Error(), http.StatusBadRequest)
				return
			}
			h.audit.Log(username, audit.ActionUserPermission, targetUser, "修改自定义权限组", ip, true)
			h.jsonOK(w, nil)

		case "unlock":
			err := h.auth.UnlockUser(targetUser, req.Reason)
			if err != nil {
				h.jsonError(w, err.Error(), http.StatusBadRequest)
				return
			}
			h.audit.Log(username, audit.ActionUserUnlock, targetUser, "解锁账户: "+req.Reason, ip, true)
			h.jsonOK(w, nil)

		case "reset_password":
			if req.NewPassword == "" {
				h.jsonError(w, "新密码不能为空", http.StatusBadRequest)
				return
			}
			err := h.auth.ChangePassword(targetUser, req.NewPassword)
			if err != nil {
				h.jsonError(w, err.Error(), http.StatusBadRequest)
				return
			}
			h.audit.Log(username, audit.ActionUserPassword, targetUser, "重置密码", ip, true)
			h.jsonOK(w, nil)

		case "rotate_key":
			newKey, err := h.auth.RotateAPIKey(targetUser)
			if err != nil {
				h.jsonError(w, err.Error(), http.StatusBadRequest)
				return
			}
			h.audit.Log(username, audit.ActionUserKeyRotate, targetUser, "轮换密钥", ip, true)
			h.jsonOK(w, map[string]string{"api_key": newKey})

		default:
			h.jsonError(w, "未知操作", http.StatusBadRequest)
		}

	default:
		h.jsonError(w, "方法不允许", http.StatusMethodNotAllowed)
	}
}

// --- Audit API ---

func (h *Handler) handleAuditAPI(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		h.jsonError(w, "方法不允许", http.StatusMethodNotAllowed)
		return
	}

	userFilter := r.URL.Query().Get("user")
	actionFilter := r.URL.Query().Get("action")

	entries := h.audit.List(100, userFilter, actionFilter)
	h.jsonOK(w, entries)
}
