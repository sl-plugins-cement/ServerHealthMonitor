package web

// Public status page (read-only, no login required)
var publicStatusHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>服务器状态 - Server Health Monitor</title>
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body {
	font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "Noto Sans CJK SC", sans-serif;
	background: linear-gradient(135deg, #0f0c29 0%, #302b63 50%, #24243e 100%);
	min-height: 100vh;
	color: #e2e8f0;
}
.container {
	max-width: 800px;
	margin: 0 auto;
	padding: 40px 20px;
}
.header {
	text-align: center;
	margin-bottom: 40px;
}
.header h1 {
	font-size: 32px;
	font-weight: 700;
	margin-bottom: 8px;
	background: linear-gradient(135deg, #fff, #a5b4fc);
	-webkit-background-clip: text;
	-webkit-text-fill-color: transparent;
	background-clip: text;
}
.header .subtitle {
	color: #94a3b8;
	font-size: 14px;
}
.header .updated {
	color: #64748b;
	font-size: 12px;
	margin-top: 8px;
}

.stats {
	display: grid;
	grid-template-columns: repeat(3, 1fr);
	gap: 16px;
	margin-bottom: 32px;
}
.stat-card {
	background: rgba(255,255,255,0.05);
	backdrop-filter: blur(10px);
	border: 1px solid rgba(255,255,255,0.1);
	border-radius: 16px;
	padding: 24px;
	text-align: center;
}
.stat-value {
	font-size: 36px;
	font-weight: 700;
	margin-bottom: 4px;
}
.stat-value.total { color: #fff; }
.stat-value.online { color: #4ade80; }
.stat-value.offline { color: #f87171; }
.stat-label {
	font-size: 13px;
	color: #94a3b8;
}

.server-list {
	background: rgba(255,255,255,0.05);
	backdrop-filter: blur(10px);
	border: 1px solid rgba(255,255,255,0.1);
	border-radius: 16px;
	overflow: hidden;
}
.server-item {
	display: flex;
	align-items: center;
	padding: 16px 20px;
	border-bottom: 1px solid rgba(255,255,255,0.05);
}
.server-item:last-child { border-bottom: none; }
.server-status-dot {
	width: 10px;
	height: 10px;
	border-radius: 50%;
	margin-right: 14px;
	flex-shrink: 0;
}
.server-status-dot.online {
	background: #4ade80;
	box-shadow: 0 0 10px #4ade80;
	animation: pulse 2s infinite;
}
.server-status-dot.offline {
	background: #f87171;
}
.server-status-dot.unknown {
	background: #64748b;
}
@keyframes pulse {
	0%, 100% { opacity: 1; }
	50% { opacity: 0.6; }
}
.server-name {
	flex: 1;
	font-size: 15px;
	font-weight: 500;
	color: #e2e8f0;
}
.server-status-text {
	font-size: 13px;
	color: #94a3b8;
}
.empty {
	text-align: center;
	padding: 40px;
	color: #64748b;
}
.footer {
	text-align: center;
	margin-top: 32px;
	color: #64748b;
	font-size: 12px;
}
.footer a {
	color: #94a3b8;
	text-decoration: none;
}
.footer a:hover { color: #fff; }

@media (max-width: 480px) {
	.stats { grid-template-columns: repeat(3, 1fr); gap: 8px; }
	.stat-card { padding: 16px 8px; }
	.stat-value { font-size: 24px; }
	.stat-label { font-size: 11px; }
}
</style>
</head>
<body>
<div class="container">
	<div class="header">
		<h1>🛡️ 服务器状态</h1>
		<p class="subtitle">Server Health Monitor · 公开状态页</p>
		<p class="updated" id="updated">加载中...</p>
	</div>

	<div class="stats">
		<div class="stat-card">
			<div class="stat-value total" id="stat-total">--</div>
			<div class="stat-label">总数</div>
		</div>
		<div class="stat-card">
			<div class="stat-value online" id="stat-online">--</div>
			<div class="stat-label">在线</div>
		</div>
		<div class="stat-card">
			<div class="stat-value offline" id="stat-offline">--</div>
			<div class="stat-label">离线</div>
		</div>
	</div>

	<div class="server-list" id="server-list">
		<div class="empty">加载中...</div>
	</div>

	<div class="footer">
		<p>Powered by SCP:SL Monitor · <a href="/">返回首页</a></p>
	</div>
</div>

<script>
async function loadStatus() {
	try {
		const res = await fetch('/public/api/status');
		const data = await res.json();
		if (data.success) {
			const d = data.data;
			document.getElementById('stat-total').textContent = d.total;
			document.getElementById('stat-online').textContent = d.online;
			document.getElementById('stat-offline').textContent = d.offline;
			document.getElementById('updated').textContent = '更新时间: ' + d.updated;

			const list = document.getElementById('server-list');
			if (d.servers.length === 0) {
				list.innerHTML = '<div class="empty">暂无监控服务器</div>';
			} else {
				let html = '';
				d.servers.forEach(function(s) {
					const cls = s.online ? 'online' : (s.status === 'offline' ? 'offline' : 'unknown');
					const text = s.online ? '在线' : (s.status === 'offline' ? '离线' : '未知');
					html += '<div class="server-item">' +
						'<div class="server-status-dot ' + cls + '"></div>' +
						'<div class="server-name">' + escapeHtml(s.name) + '</div>' +
						'<div class="server-status-text">' + text + '</div>' +
					'</div>';
				});
				list.innerHTML = html;
			}
		}
	} catch(e) {
		document.getElementById('server-list').innerHTML = '<div class="empty">加载失败</div>';
	}
}

function escapeHtml(str) {
	if (!str) return '';
	const div = document.createElement('div');
	div.appendChild(document.createTextNode(str));
	return div.innerHTML;
}

loadStatus();
setInterval(loadStatus, 30000);
</script>
</body>
</html>`

// Tools page (inside console, for connectivity testing etc.)
var toolsPageHTML = `
	<div class="page-header">
		<div>
			<h2>🧰 工具箱</h2>
			<p class="page-desc">连通性检测、网络诊断工具</p>
		</div>
	</div>

	<div class="section">
		<h3>🔌 一键连通性检测</h3>
		<p style="color:#94a3b8;font-size:13px;margin-bottom:16px">快速检测服务器端口、HTTP 服务或 Agent 是否可达</p>
		
		<div style="display:flex;gap:10px;margin-bottom:16px;flex-wrap:wrap">
			<input type="text" id="check-host" placeholder="主机地址 (如 server.example.com)" style="flex:1;min-width:200px;padding:10px 12px;background:#0f172a;border:1px solid #334155;border-radius:8px;color:#e2e8f0;font-size:13px">
			<input type="number" id="check-port" placeholder="端口" value="8080" style="width:120px;padding:10px 12px;background:#0f172a;border:1px solid #334155;border-radius:8px;color:#e2e8f0;font-size:13px">
			<select id="check-type" style="padding:10px 12px;background:#0f172a;border:1px solid #334155;border-radius:8px;color:#e2e8f0;font-size:13px">
				<option value="agent">Agent 检测</option>
				<option value="tcp">TCP 端口</option>
				<option value="http">HTTP 服务</option>
			</select>
			<button class="btn btn-primary" onclick="runCheck()" id="check-btn">🔍 开始检测</button>
		</div>

		<div class="check-result" id="check-result" style="display:none">
			<div class="check-result-header">
				<span class="check-status" id="check-status"></span>
				<span class="check-latency" id="check-latency"></span>
			</div>
			<div class="check-target" id="check-target"></div>
			<div class="check-error" id="check-error" style="display:none"></div>
		</div>
	</div>

	<div class="section">
		<h3>📡 快捷检测模板</h3>
		<p style="color:#94a3b8;font-size:13px;margin-bottom:12px">从已添加的服务器中快速检测</p>
		<div id="quick-check-list" style="display:flex;gap:8px;flex-wrap:wrap">
			<span style="color:#64748b;font-size:13px">暂无服务器</span>
		</div>
	</div>

<script>
async function runCheck() {
	const host = document.getElementById('check-host').value.trim();
	const port = parseInt(document.getElementById('check-port').value) || 0;
	const type = document.getElementById('check-type').value;
	const btn = document.getElementById('check-btn');

	if (!host) {
		toast('请输入主机地址', 'error');
		return;
	}

	btn.disabled = true;
	btn.textContent = '检测中...';

	try {
		const res = await apiCall('/console/api/check', 'POST', {
			host: host, port: port, type: type, timeout: 5
		});
		if (res.ok && res.data.success) {
			const d = res.data.data;
			const resultEl = document.getElementById('check-result');
			const statusEl = document.getElementById('check-status');
			const latencyEl = document.getElementById('check-latency');
			const targetEl = document.getElementById('check-target');
			const errorEl = document.getElementById('check-error');

			resultEl.style.display = 'block';
			if (d.success) {
				statusEl.textContent = '✅ 可达';
				statusEl.className = 'check-status success';
			} else {
				statusEl.textContent = '❌ 不可达';
				statusEl.className = 'check-status fail';
			}
			latencyEl.textContent = d.latency_ms + ' ms';
			targetEl.textContent = d.type.toUpperCase() + ' · ' + d.target;
			if (d.error) {
				errorEl.textContent = d.error;
				errorEl.style.display = 'block';
			} else {
				errorEl.style.display = 'none';
			}
		} else {
			toast(res.data.error || '检测失败', 'error');
		}
	} catch(e) {
		toast('网络错误', 'error');
	}

	btn.disabled = false;
	btn.textContent = '🔍 开始检测';
}

async function loadQuickCheck() {
	const res = await apiCall('/console/api/servers');
	if (res.ok && res.data.success && res.data.data.length > 0) {
		const list = document.getElementById('quick-check-list');
		let html = '';
		res.data.data.forEach(function(s) {
			html += '<button class="btn btn-secondary btn-sm" onclick="quickCheck(\'' + s.host + '\', ' + s.agent_port + ')">' + escapeHtml(s.name) + '</button>';
		});
		list.innerHTML = html;
	}
}

function quickCheck(host, port) {
	document.getElementById('check-host').value = host;
	document.getElementById('check-port').value = port;
	document.getElementById('check-type').value = 'agent';
	runCheck();
}

if (document.getElementById('quick-check-list')) {
	loadQuickCheck();
}
</script>
`

// Add styles for tools page - inject into dashboard via CSS extension
// Actually we'll add these styles to the layout inline
var toolsExtraStyles = `
<style>
.check-result {
	background: #0f172a;
	border: 1px solid #334155;
	border-radius: 10px;
	padding: 16px;
	margin-top: 12px;
}
.check-result-header {
	display: flex;
	justify-content: space-between;
	align-items: center;
	margin-bottom: 8px;
}
.check-status {
	font-size: 16px;
	font-weight: 600;
}
.check-status.success { color: #4ade80; }
.check-status.fail { color: #f87171; }
.check-latency {
	font-family: monospace;
	font-size: 14px;
	color: #a5b4fc;
	background: rgba(99,102,241,0.15);
	padding: 4px 10px;
	border-radius: 6px;
}
.check-target {
	font-size: 13px;
	color: #94a3b8;
	font-family: monospace;
}
.check-error {
	margin-top: 8px;
	padding: 8px 12px;
	background: rgba(239,68,68,0.1);
	border-left: 3px solid #ef4444;
	border-radius: 4px;
	font-size: 12px;
	color: #fca5a5;
}
</style>
`
