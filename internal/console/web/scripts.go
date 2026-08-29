package web

// Page-specific scripts embedded at the end of each page.

// Servers page script
var serversPageScript = `
<script>
let allServers = [];
let currentDeployServerId = null;

async function loadServers() {
	const res = await apiCall('/console/api/servers');
	if (res.ok && res.data.success) {
		allServers = res.data.data;
		renderServers(allServers);
		updateDashboardStats(allServers);
	}
}

function renderServers(servers) {
	const tbody = document.getElementById('servers-tbody');
	if (!tbody) return;
	
	if (servers.length === 0) {
		tbody.innerHTML = '<tr><td colspan="7" class="empty-cell">暂无服务器</td></tr>';
		return;
	}

	let html = '';
	servers.forEach(function(s) {
		html += '<tr>' +
			'<td><strong>' + escapeHtml(s.name) + '</strong></td>' +
			'<td>' + escapeHtml(s.host) + '</td>' +
			'<td>' + s.port + '</td>' +
			'<td>' + escapeHtml(s.user) + '</td>' +
			'<td>' + s.agent_port + '</td>' +
			'<td><span class="badge badge-' + statusClass(s.status) + '">' + statusText(s.status) + '</span></td>' +
			'<td><div class="action-btns">' +
				'<button class="btn btn-success btn-sm" onclick="openDeploy(\'' + s.id + '\')">🚀 部署</button> ' +
				'<button class="btn btn-secondary btn-sm" onclick="editServer(\'' + s.id + '\')">✏️ 编辑</button> ' +
				'<button class="btn btn-danger btn-sm" onclick="deleteServer(\'' + s.id + '\', \'' + escapeHtml(s.name) + '\')">🗑️</button>' +
			'</div></td>' +
		'</tr>';
	});
	tbody.innerHTML = html;
}

function updateDashboardStats(servers) {
	const onlineEl = document.getElementById('stat-online');
	const totalEl = document.getElementById('stat-servers');
	const offlineEl = document.getElementById('stat-offline');
	const listEl = document.getElementById('server-list');
	
	if (totalEl) totalEl.textContent = servers.length;
	
	let online = 0, offline = 0;
	servers.forEach(function(s) {
		if (s.status === 'online') online++;
		else offline++;
	});
	
	if (onlineEl) onlineEl.textContent = online;
	if (offlineEl) offlineEl.textContent = offline;

	if (listEl && servers.length > 0) {
		let html = '';
		servers.forEach(function(s) {
			html += '<div class="server-item">' +
				'<div class="server-item-icon">🖥️</div>' +
				'<div class="server-item-info">' +
					'<div class="server-item-name">' + escapeHtml(s.name) + '</div>' +
					'<div class="server-item-host">' + escapeHtml(s.host) + ':' + s.agent_port + '</div>' +
				'</div>' +
				'<div class="server-item-status">' +
					'<span class="badge badge-' + statusClass(s.status) + '">' + statusText(s.status) + '</span>' +
				'</div>' +
			'</div>';
		});
		listEl.innerHTML = html;
	}
}

function statusClass(s) {
	switch(s) {
		case 'online': return 'success';
		case 'offline': return 'danger';
		default: return 'secondary';
	}
}
function statusText(s) {
	switch(s) {
		case 'online': return '在线';
		case 'offline': return '离线';
		default: return '未知';
	}
}

function filterServers() {
	const q = document.getElementById('search-server').value.toLowerCase();
	const filtered = allServers.filter(function(s) {
		return s.name.toLowerCase().indexOf(q) >= 0 || s.host.toLowerCase().indexOf(q) >= 0;
	});
	renderServers(filtered);
}

function openAddServer() {
	document.getElementById('srv-name').value = '';
	document.getElementById('srv-host').value = '';
	document.getElementById('srv-port').value = '22';
	document.getElementById('srv-user').value = 'root';
	document.getElementById('srv-agent-port').value = '8080';
	document.getElementById('srv-keyfile').value = '';
	openModal('add-server-modal');
}

async function addServer() {
	const name = document.getElementById('srv-name').value.trim();
	const host = document.getElementById('srv-host').value.trim();
	const port = parseInt(document.getElementById('srv-port').value) || 22;
	const user = document.getElementById('srv-user').value.trim();
	const agentPort = parseInt(document.getElementById('srv-agent-port').value) || 8080;
	const keyFile = document.getElementById('srv-keyfile').value.trim();

	if (!name || !host) {
		toast('请填写名称和地址', 'error');
		return;
	}

	const res = await apiCall('/console/api/servers', 'POST', {
		name: name, host: host, port: port, user: user, agent_port: agentPort, key_file: keyFile
	});
	if (res.ok && res.data.success) {
		toast('服务器添加成功', 'success');
		closeModal('add-server-modal');
		loadServers();
	} else {
		toast(res.data.error || '添加失败', 'error');
	}
}

function editServer(id) {
	const s = allServers.find(function(x) { return x.id === id; });
	if (!s) return;
	document.getElementById('srv-name').value = s.name;
	document.getElementById('srv-host').value = s.host;
	document.getElementById('srv-port').value = s.port;
	document.getElementById('srv-user').value = s.user;
	document.getElementById('srv-agent-port').value = s.agent_port;
	document.getElementById('srv-keyfile').value = s.key_file || '';
	openModal('add-server-modal');
	toast('编辑功能：修改后点击确认添加会更新', 'info');
}

async function deleteServer(id, name) {
	if (!confirm('确定要删除服务器 "' + name + '" 吗？此操作不可撤销。')) return;
	const res = await apiCall('/console/api/servers/' + id, 'DELETE');
	if (res.ok && res.data.success) {
		toast('已删除', 'success');
		loadServers();
	} else {
		toast(res.data.error || '删除失败', 'error');
	}
}

function openDeploy(id) {
	currentDeployServerId = id;
	const s = allServers.find(function(x) { return x.id === id; });
	if (!s) return;
	document.getElementById('deploy-info').innerHTML =
		'即将部署到 <strong>' + escapeHtml(s.name) + '</strong> (' + escapeHtml(s.host) + ')';
	document.getElementById('deploy-output').textContent = '';
	document.getElementById('deploy-btn').disabled = false;
	document.getElementById('deploy-btn').textContent = '开始部署';
	openModal('deploy-modal');
}

async function doDeploy() {
	const btn = document.getElementById('deploy-btn');
	const output = document.getElementById('deploy-output');
	const binaryPath = document.getElementById('deploy-binary').value;

	btn.disabled = true;
	btn.textContent = '部署中...';
	output.textContent = '正在准备部署...\n';

	const res = await apiCall('/console/api/deploy', 'POST', {
		server_id: currentDeployServerId,
		binary_path: binaryPath
	});

	if (res.ok && res.data.success) {
		output.textContent = res.data.data.output || '部署完成';
		btn.textContent = '部署完成';
		toast('部署成功', 'success');
	} else {
		output.textContent = '部署失败: ' + (res.data.error || '未知错误');
		btn.disabled = false;
		btn.textContent = '重试';
		toast('部署失败', 'error');
	}
}

function escapeHtml(str) {
	if (!str) return '';
	const div = document.createElement('div');
	div.appendChild(document.createTextNode(str));
	return div.innerHTML;
}

if (document.getElementById('servers-tbody')) {
	loadServers();
}
if (document.getElementById('server-list')) {
	loadServers();
	setInterval(loadServers, 30000);
}
</script>
`

// Users page script
var usersPageScript = `
<script>
async function loadUsers() {
	const res = await apiCall('/console/api/users');
	if (res.ok && res.data.success) {
		renderUsers(res.data.data);
	}
}

function renderUsers(users) {
	const tbody = document.getElementById('users-tbody');
	if (!tbody) return;

	let html = '';
	users.forEach(function(u) {
		const roleBadge = (u.role === 'owner' || u.role === 'director' || u.role === 'admin') ? 'info' : 'secondary';
		const roleText = ({owner:'服主', director:'技术总监', admin:'管理员', operator:'运维人员', reporter:'报告提交者', custom:'自定义权限组', viewer:'查看者'})[u.role] || u.role;
		const statusBadge = u.locked ? 'danger' : 'success';
		const statusText = u.locked ? '🔒 已锁定' : '✅ 正常';
		const unlockBtn = u.locked ? '<button class="btn btn-warning btn-sm" onclick="unlockUser(\'' + escapeHtml(u.username) + '\')">🔓 解锁</button> ' : '';
		const deleteBtn = u.username !== currentUser ? '<button class="btn btn-danger btn-sm" onclick="deleteUser(\'' + escapeHtml(u.username) + '\')">🗑️</button>' : '';
		html += '<tr>' +
			'<td><strong>' + escapeHtml(u.username) + '</strong></td>' +
			'<td><span class="badge badge-' + roleBadge + '">' + roleText + '</span></td>' +
			'<td><span class="badge badge-' + statusBadge + '">' + statusText + '</span></td>' +
			'<td>' + u.failed_attempts + '/3</td>' +
			'<td>' + (u.last_login ? formatDate(u.last_login) : '从未') + '</td>' +
			'<td>' + formatDate(u.created_at) + '</td>' +
			'<td><div class="action-btns">' +
				unlockBtn +
				'<button class="btn btn-secondary btn-sm" onclick="resetPassword(\'' + escapeHtml(u.username) + '\')">🔑 重置密码</button> ' +
				'<button class="btn btn-secondary btn-sm" onclick="rotateKey(\'' + escapeHtml(u.username) + '\')">🔄 换密钥</button> ' +
				deleteBtn +
			'</div></td>' +
		'</tr>';
	});
	tbody.innerHTML = html;
}

function openAddUser() {
	document.getElementById('new-username').value = '';
	document.getElementById('new-password').value = '';
	document.getElementById('new-role').value = 'viewer';
	document.getElementById('custom-permissions-group').style.display = 'none';
	document.getElementById('custom-permissions').selectedIndex = -1;
	openModal('add-user-modal');
}

document.getElementById('new-role').addEventListener('change', function() {
	document.getElementById('custom-permissions-group').style.display = this.value === 'custom' ? 'block' : 'none';
});

async function addUser() {
	const username = document.getElementById('new-username').value.trim();
	const password = document.getElementById('new-password').value;
	const role = document.getElementById('new-role').value;
	const permissions = Array.from(document.getElementById('custom-permissions').selectedOptions).map(function(option) { return option.value; });

	if (!username || !password) {
		toast('请填写用户名和密码', 'error');
		return;
	}
	if (password.length < 12) {
		toast('密码至少 12 位，并包含大小写字母、数字和特殊字符', 'error');
		return;
	}

	const res = await apiCall('/console/api/users', 'POST', { username: username, password: password, role: role, permissions: permissions });
	if (res.ok && res.data.success) {
		closeModal('add-user-modal');
		document.getElementById('apikey-display').textContent = res.data.data.api_key;
		openModal('apikey-modal');
		loadUsers();
	} else {
		toast(res.data.error || '创建失败', 'error');
	}
}

async function unlockUser(username) {
	const reason = prompt('请输入解锁原因：');
	if (!reason) return;
	const res = await apiCall('/console/api/users/' + username, 'POST', {
		action: 'unlock',
		reason: reason
	});
	if (res.ok && res.data.success) {
		toast('账户已解锁', 'success');
		loadUsers();
	} else {
		toast(res.data.error || '解锁失败', 'error');
	}
}

async function resetPassword(username) {
	const newPwd = prompt('请输入新密码（至少 12 位，含大小写字母、数字和特殊字符）：');
	if (!newPwd) return;
	if (newPwd.length < 12) {
		toast('密码至少 12 位，并包含大小写字母、数字和特殊字符', 'error');
		return;
	}
	const res = await apiCall('/console/api/users/' + username, 'POST', {
		action: 'reset_password',
		new_password: newPwd
	});
	if (res.ok && res.data.success) {
		toast('密码已重置', 'success');
	} else {
		toast(res.data.error || '重置失败', 'error');
	}
}

async function rotateKey(username) {
	if (!confirm('确定要轮换 ' + username + ' 的 API 密钥吗？旧密钥将立即失效。')) return;
	const res = await apiCall('/console/api/users/' + username, 'POST', {
		action: 'rotate_key'
	});
	if (res.ok && res.data.success) {
		document.getElementById('apikey-display').textContent = res.data.data.api_key;
		openModal('apikey-modal');
		loadUsers();
	} else {
		toast(res.data.error || '轮换失败', 'error');
	}
}

async function deleteUser(username) {
	if (!confirm('确定要删除用户 "' + username + '" 吗？此操作不可撤销。')) return;
	const res = await apiCall('/console/api/users/' + username, 'DELETE');
	if (res.ok && res.data.success) {
		toast('用户已删除', 'success');
		loadUsers();
	} else {
		toast(res.data.error || '删除失败', 'error');
	}
}

function formatDate(d) {
	if (!d) return '-';
	const date = new Date(d);
	if (isNaN(date)) return d;
	return date.toLocaleString('zh-CN');
}

function escapeHtml(str) {
	if (!str) return '';
	const div = document.createElement('div');
	div.appendChild(document.createTextNode(str));
	return div.innerHTML;
}

if (document.getElementById('users-tbody')) {
	loadUsers();
}
</script>
`

// Audit page script
var auditPageScript = `
<script>
async function refreshAudit() {
	const user = document.getElementById('audit-user').value;
	const action = document.getElementById('audit-action').value;
	let url = '/console/api/audit?';
	if (user) url += 'user=' + encodeURIComponent(user) + '&';
	if (action) url += 'action=' + encodeURIComponent(action);

	const res = await apiCall(url);
	if (res.ok && res.data.success) {
		renderAudit(res.data.data);
	}
}

function renderAudit(entries) {
	const tbody = document.getElementById('audit-tbody');
	if (!tbody) return;

	if (entries.length === 0) {
		tbody.innerHTML = '<tr><td colspan="7" class="empty-cell">暂无记录</td></tr>';
		return;
	}

	let html = '';
	entries.forEach(function(e) {
		html += '<tr>' +
			'<td style="white-space:nowrap">' + escapeHtml(e.time) + '</td>' +
			'<td><strong>' + escapeHtml(e.user) + '</strong></td>' +
			'<td><span class="badge badge-' + actionClass(e.action) + '">' + actionText(e.action) + '</span></td>' +
			'<td>' + escapeHtml(e.target || '-') + '</td>' +
			'<td style="max-width:300px;overflow:hidden;text-overflow:ellipsis">' + escapeHtml(e.detail || '-') + '</td>' +
			'<td style="font-family:monospace;font-size:12px">' + escapeHtml(e.ip || '-') + '</td>' +
			'<td><span class="badge badge-' + (e.success ? 'success' : 'danger') + '">' + (e.success ? '成功' : '失败') + '</span></td>' +
		'</tr>';
	});
	tbody.innerHTML = html;
}

function actionClass(a) {
	switch(a) {
		case 'login': return 'success';
		case 'login_failed': return 'danger';
		case 'user_create': return 'info';
		case 'user_delete': return 'danger';
		case 'user_unlock': return 'warning';
		case 'server_add': return 'info';
		case 'server_delete': return 'danger';
		case 'deploy': return 'info';
		default: return 'secondary';
	}
}

function actionText(a) {
	const map = {
		'login': '登录',
		'login_failed': '登录失败',
		'logout': '登出',
		'user_create': '创建用户',
		'user_delete': '删除用户',
		'user_password_change': '修改密码',
		'user_key_rotate': '轮换密钥',
		'user_unlock': '解锁账户',
		'server_add': '添加服务器',
		'server_update': '更新服务器',
		'server_delete': '删除服务器',
		'deploy': '部署',
		'config_change': '配置变更',
		'view_dashboard': '查看仪表盘',
	};
	return map[a] || a;
}

function escapeHtml(str) {
	if (!str) return '';
	const div = document.createElement('div');
	div.appendChild(document.createTextNode(str));
	return div.innerHTML;
}

if (document.getElementById('audit-tbody')) {
	refreshAudit();
}
</script>
`
