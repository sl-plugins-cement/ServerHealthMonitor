package web

func consoleLayoutHTML(content, active string) string {
	return `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>服务器运维控制台</title>
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body {
	font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "Noto Sans CJK SC", sans-serif;
	background: #0f172a;
	color: #e2e8f0;
	min-height: 100vh;
	display: flex;
}

/* Sidebar */
.sidebar {
	width: 240px;
	background: #1e293b;
	border-right: 1px solid #334155;
	display: flex;
	flex-direction: column;
	position: fixed;
	height: 100vh;
	z-index: 100;
}
.sidebar-logo {
	padding: 24px 20px;
	border-bottom: 1px solid #334155;
	display: flex;
	align-items: center;
	gap: 12px;
}
.sidebar-logo .icon { font-size: 28px; }
.sidebar-logo h1 { font-size: 16px; font-weight: 600; color: #fff; }
.sidebar-logo p { font-size: 11px; color: #94a3b8; }

.sidebar-nav { flex: 1; padding: 16px 12px; }
.nav-item {
	display: flex;
	align-items: center;
	gap: 12px;
	padding: 10px 14px;
	border-radius: 8px;
	color: #94a3b8;
	text-decoration: none;
	font-size: 14px;
	margin-bottom: 4px;
	transition: all 0.2s;
	cursor: pointer;
}
.nav-item:hover { background: #334155; color: #e2e8f0; }
.nav-item.active {
	background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
	color: #fff;
}
.nav-item .nav-icon { font-size: 18px; width: 24px; text-align: center; }

.sidebar-footer {
	padding: 16px;
	border-top: 1px solid #334155;
}
.user-info {
	display: flex;
	align-items: center;
	gap: 10px;
	margin-bottom: 12px;
}
.user-avatar {
	width: 36px; height: 36px;
	background: linear-gradient(135deg, #6366f1, #8b5cf6);
	border-radius: 50%;
	display: flex;
	align-items: center;
	justify-content: center;
	font-weight: 600;
	font-size: 14px;
}
.user-meta .user-name { font-size: 13px; font-weight: 500; color: #e2e8f0; }
.user-meta .user-role { font-size: 11px; color: #64748b; }
.logout-btn {
	width: 100%;
	padding: 8px;
	background: transparent;
	border: 1px solid #475569;
	border-radius: 6px;
	color: #94a3b8;
	font-size: 13px;
	cursor: pointer;
	transition: all 0.2s;
}
.logout-btn:hover { background: #ef4444; border-color: #ef4444; color: #fff; }
.logout-btn + .logout-btn { margin-top: 8px; }
.change-pass-btn:hover { background: #4f46e5; border-color: #6366f1; color: #fff; }

/* Main content */
.main-content {
	flex: 1;
	margin-left: 240px;
	padding: 24px 32px;
}

.page-header {
	display: flex;
	justify-content: space-between;
	align-items: flex-start;
	margin-bottom: 24px;
}
.page-header h2 { font-size: 24px; font-weight: 600; color: #fff; }
.page-desc { color: #64748b; font-size: 14px; margin-top: 4px; }

/* Stats grid */
.stats-grid {
	display: grid;
	grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
	gap: 16px;
	margin-bottom: 24px;
}
.stat-card {
	background: #1e293b;
	border: 1px solid #334155;
	border-radius: 12px;
	padding: 20px;
	display: flex;
	align-items: center;
	gap: 16px;
	transition: all 0.2s;
}
.stat-card:hover { transform: translateY(-2px); border-color: #475569; }
.stat-icon {
	width: 48px; height: 48px;
	background: rgba(99,102,241,0.15);
	border-radius: 12px;
	display: flex;
	align-items: center;
	justify-content: center;
	font-size: 24px;
}
.stat-online .stat-icon { background: rgba(34,197,94,0.15); }
.stat-offline .stat-icon { background: rgba(239,68,68,0.15); }
.stat-alerts .stat-icon { background: rgba(251,191,36,0.15); }
.stat-value { font-size: 28px; font-weight: 700; color: #fff; }
.stat-label { font-size: 13px; color: #64748b; margin-top: 2px; }

/* Sections */
.section {
	background: #1e293b;
	border: 1px solid #334155;
	border-radius: 12px;
	padding: 20px;
	margin-bottom: 20px;
}
.section h3 {
	font-size: 16px;
	font-weight: 600;
	color: #fff;
	margin-bottom: 16px;
}

/* Buttons */
.btn {
	padding: 8px 16px;
	border-radius: 8px;
	font-size: 13px;
	font-weight: 500;
	cursor: pointer;
	border: none;
	transition: all 0.2s;
	display: inline-flex;
	align-items: center;
	gap: 6px;
}
.btn-primary {
	background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
	color: #fff;
}
.btn-primary:hover { transform: translateY(-1px); box-shadow: 0 4px 12px rgba(99,102,241,0.3); }
.btn-secondary {
	background: #334155;
	color: #e2e8f0;
	border: 1px solid #475569;
}
.btn-secondary:hover { background: #475569; }
.btn-danger {
	background: #ef4444;
	color: #fff;
}
.btn-danger:hover { background: #dc2626; }
.btn-sm { padding: 5px 10px; font-size: 12px; }
.btn-success {
	background: #22c55e;
	color: #fff;
}
.btn-success:hover { background: #16a34a; }
.btn-warning {
	background: #f59e0b;
	color: #fff;
}
.btn-warning:hover { background: #d97706; }

/* Table */
.data-table {
	width: 100%;
	border-collapse: collapse;
}
.data-table th {
	text-align: left;
	padding: 12px 14px;
	font-size: 12px;
	font-weight: 600;
	color: #64748b;
	text-transform: uppercase;
	letter-spacing: 0.5px;
	border-bottom: 1px solid #334155;
}
.data-table td {
	padding: 12px 14px;
	font-size: 13px;
	color: #cbd5e1;
	border-bottom: 1px solid #334155;
}
.data-table tr:hover td { background: rgba(255,255,255,0.02); }
.empty-cell {
	text-align: center;
	color: #64748b !important;
	padding: 40px !important;
}

/* Status badges */
.badge {
	display: inline-block;
	padding: 3px 10px;
	border-radius: 12px;
	font-size: 11px;
	font-weight: 500;
}
.badge-success { background: rgba(34,197,94,0.15); color: #4ade80; }
.badge-danger { background: rgba(239,68,68,0.15); color: #f87171; }
.badge-warning { background: rgba(251,191,36,0.15); color: #fbbf24; }
.badge-info { background: rgba(99,102,241,0.15); color: #a5b4fc; }
.badge-secondary { background: rgba(100,116,139,0.15); color: #94a3b8; }

/* Action buttons in table */
.action-btns { display: flex; gap: 6px; }
.action-btns .btn { padding: 4px 10px; font-size: 12px; }

/* Modal */
.modal {
	display: none;
	position: fixed;
	top: 0; left: 0; right: 0; bottom: 0;
	background: rgba(0,0,0,0.6);
	z-index: 1000;
	align-items: center;
	justify-content: center;
}
.modal.show { display: flex; }
.modal-content {
	background: #1e293b;
	border: 1px solid #334155;
	border-radius: 12px;
	width: 100%;
	max-width: 500px;
	max-height: 90vh;
	overflow-y: auto;
}
.modal-header {
	padding: 20px;
	border-bottom: 1px solid #334155;
	display: flex;
	justify-content: space-between;
	align-items: center;
}
.modal-header h3 { font-size: 18px; font-weight: 600; color: #fff; }
.modal-close {
	background: transparent;
	border: none;
	color: #64748b;
	font-size: 20px;
	cursor: pointer;
	padding: 4px;
}
.modal-close:hover { color: #fff; }
.modal-body { padding: 20px; }
.modal-footer {
	padding: 16px 20px;
	border-top: 1px solid #334155;
	display: flex;
	justify-content: flex-end;
	gap: 10px;
}

/* Forms */
.form-group { margin-bottom: 16px; }
.form-group label {
	display: block;
	margin-bottom: 6px;
	font-size: 13px;
	font-weight: 500;
	color: #cbd5e1;
}
.form-group input, .form-group select {
	width: 100%;
	padding: 10px 12px;
	background: #0f172a;
	border: 1px solid #334155;
	border-radius: 8px;
	color: #e2e8f0;
	font-size: 13px;
	transition: all 0.2s;
}
.form-group input:focus, .form-group select:focus {
	outline: none;
	border-color: #6366f1;
	box-shadow: 0 0 0 3px rgba(99,102,241,0.15);
}
.form-row { display: flex; gap: 12px; }
.form-row .form-group { flex: 1; }

/* Security notice */
.security-notice {
	background: rgba(251,191,36,0.1);
	border: 1px solid rgba(251,191,36,0.3);
	border-radius: 8px;
	padding: 14px 16px;
	font-size: 13px;
	color: #fcd34d;
}

/* Notice box */
.notice-box {
	background: rgba(251,191,36,0.1);
	border: 1px solid rgba(251,191,36,0.3);
	border-radius: 8px;
	padding: 12px 14px;
	font-size: 13px;
	color: #fcd34d;
	margin-top: 8px;
}

/* API key display */
.apikey-box {
	background: #0f172a;
	border: 2px dashed #6366f1;
	border-radius: 8px;
	padding: 16px;
	font-family: monospace;
	font-size: 13px;
	color: #a5b4fc;
	word-break: break-all;
	margin: 12px 0;
	user-select: all;
}
.apikey-hint { font-size: 12px; color: #64748b; }

/* Toolbar */
.toolbar {
	display: flex;
	gap: 10px;
	margin-bottom: 16px;
}
.search-input, .select-input {
	padding: 8px 12px;
	background: #0f172a;
	border: 1px solid #334155;
	border-radius: 8px;
	color: #e2e8f0;
	font-size: 13px;
}
.search-input { flex: 1; max-width: 300px; }
.search-input:focus, .select-input:focus {
	outline: none;
	border-color: #6366f1;
}

/* Deploy output */
.deploy-output {
	background: #0f172a;
	border: 1px solid #334155;
	border-radius: 8px;
	padding: 12px;
	font-family: monospace;
	font-size: 12px;
	color: #94a3b8;
	max-height: 200px;
	overflow-y: auto;
	margin-top: 12px;
	white-space: pre-wrap;
}
.deploy-info {
	background: rgba(99,102,241,0.1);
	border: 1px solid rgba(99,102,241,0.3);
	border-radius: 8px;
	padding: 12px 14px;
	font-size: 13px;
	color: #a5b4fc;
	margin-bottom: 16px;
}

/* Empty state */
.empty-state {
	text-align: center;
	padding: 40px 20px;
	color: #64748b;
}
.empty-icon { font-size: 48px; margin-bottom: 12px; }
.empty-hint { font-size: 13px; margin-top: 8px; }
.empty-hint a { color: #6366f1; text-decoration: none; }
.empty-hint a:hover { text-decoration: underline; }

/* Server list on dashboard */
.server-list { display: flex; flex-direction: column; gap: 10px; }
.server-item {
	display: flex;
	align-items: center;
	padding: 14px 16px;
	background: #0f172a;
	border: 1px solid #334155;
	border-radius: 10px;
	transition: all 0.2s;
}
.server-item:hover { border-color: #475569; }
.server-item-icon {
	width: 40px; height: 40px;
	background: rgba(99,102,241,0.15);
	border-radius: 10px;
	display: flex;
	align-items: center;
	justify-content: center;
	font-size: 20px;
	margin-right: 14px;
}
.server-item-info { flex: 1; }
.server-item-name { font-size: 14px; font-weight: 500; color: #e2e8f0; }
.server-item-host { font-size: 12px; color: #64748b; margin-top: 2px; }
.server-item-status {
	display: flex;
	align-items: center;
	gap: 8px;
}

/* Responsive */
@media (max-width: 768px) {
	.sidebar { width: 60px; }
	.sidebar-logo h1, .sidebar-logo p, .nav-item span, .user-meta, .logout-btn span { display: none; }
	.main-content { margin-left: 60px; padding: 16px; }
	.stats-grid { grid-template-columns: repeat(2, 1fr); }
	.form-row { flex-direction: column; gap: 0; }
}
</style>
</head>
<body>
<aside class="sidebar">
	<div class="sidebar-logo">
		<div class="icon">🛡️</div>
		<div>
			<h1>监控控制台</h1>
			<p>Server Health Monitor</p>
		</div>
	</div>
	<nav class="sidebar-nav">
		<a href="/console/dashboard" class="nav-item ` + activeClass(active, "dashboard") + `">
			<span class="nav-icon">📊</span><span>仪表盘</span>
		</a>
		<a href="/console/servers" class="nav-item ` + activeClass(active, "servers") + `">
			<span class="nav-icon">🖥️</span><span>服务器管理</span>
		</a>
		<a href="/console/tickets" class="nav-item ` + activeClass(active, "tickets") + `">
			<span class="nav-icon">🎫</span><span>工单中心</span>
		</a>
		<a href="/console/users" class="nav-item ` + activeClass(active, "users") + `">
			<span class="nav-icon">👥</span><span>用户管理</span>
		</a>
		<a href="/console/audit" class="nav-item ` + activeClass(active, "audit") + `">
			<span class="nav-icon">📋</span><span>审计日志</span>
		</a>
		<a href="/console/tools" class="nav-item ` + activeClass(active, "tools") + `">
			<span class="nav-icon">🧰</span><span>工具箱</span>
		</a>
	</nav>
	<div class="sidebar-footer">
		<div class="user-info">
			<div class="user-avatar" id="user-avatar">A</div>
			<div class="user-meta">
				<div class="user-name" id="user-name">加载中...</div>
				<div class="user-role" id="user-role">-</div>
			</div>
		</div>
		<button class="logout-btn change-pass-btn" onclick="window.location.href='/console/change-password'">🔑 修改密码</button>
		<button class="logout-btn" onclick="doLogout()">🚪 退出登录</button>
	</div>
</aside>

<main class="main-content">
` + content + `
</main>

<script>
let currentUser = null;
let currentRole = null;

// Load user info
async function loadUser() {
	try {
		const res = await fetch('/console/api/me');
		const data = await res.json();
		if (data.success) {
			currentUser = data.data.username;
			currentRole = data.data.role;
			document.getElementById('user-name').textContent = currentUser;
			document.getElementById('user-role').textContent = currentRole === 'admin' ? '管理员' : '查看者';
			document.getElementById('user-avatar').textContent = currentUser.charAt(0).toUpperCase();
		}
	} catch(e) { console.error(e); }
}

async function doLogout() {
	if (!confirm('确定要退出登录吗？')) return;
	await fetch('/console/api/logout', { method: 'POST' });
	window.location.href = '/console/login';
}

// Modal helpers
function openModal(id) { document.getElementById(id).classList.add('show'); }
function closeModal(id) { document.getElementById(id).classList.remove('show'); }

// Close modal on background click
document.addEventListener('click', (e) => {
	if (e.target.classList.contains('modal')) {
		e.target.classList.remove('show');
	}
});

// API helper
async function apiCall(url, method, body) {
	const opts = { method: method || 'GET', headers: {} };
	if (body) {
		opts.headers['Content-Type'] = 'application/json';
		opts.body = JSON.stringify(body);
	}
	const res = await fetch(url, opts);
	const data = await res.json();
	return { ok: res.ok, data, status: res.status };
}

// Show toast notification
function toast(msg, type) {
	const t = document.createElement('div');
	t.style.cssText = 'position:fixed;top:20px;right:20px;padding:12px 20px;border-radius:8px;z-index:9999;font-size:14px;box-shadow:0 4px 12px rgba(0,0,0,0.3);';
	if (type === 'error') {
		t.style.background = '#ef4444';
		t.style.color = '#fff';
	} else if (type === 'success') {
		t.style.background = '#22c55e';
		t.style.color = '#fff';
	} else {
		t.style.background = '#6366f1';
		t.style.color = '#fff';
	}
	t.textContent = msg;
	document.body.appendChild(t);
	setTimeout(() => { t.style.opacity = '0'; t.style.transition = 'opacity 0.3s'; }, 2500);
	setTimeout(() => t.remove(), 3000);
}

loadUser();
</script>
</body>
</html>`
}

func activeClass(current, target string) string {
	if current == target {
		return "active"
	}
	return ""
}
