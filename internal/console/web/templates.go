package web

var loginPageHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>登录 - SCP:SL 监控控制台</title>
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body {
	font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "Noto Sans CJK SC", sans-serif;
	background: linear-gradient(135deg, #0f0c29 0%, #302b63 50%, #24243e 100%);
	min-height: 100vh;
	display: flex;
	align-items: center;
	justify-content: center;
	color: #e0e0e0;
}
.login-container {
	background: rgba(255,255,255,0.05);
	backdrop-filter: blur(20px);
	border: 1px solid rgba(255,255,255,0.1);
	border-radius: 20px;
	padding: 48px 40px;
	width: 100%;
	max-width: 420px;
	box-shadow: 0 25px 50px -12px rgba(0,0,0,0.5);
}
.login-header {
	text-align: center;
	margin-bottom: 36px;
}
.login-header .logo {
	font-size: 48px;
	margin-bottom: 12px;
}
.login-header h1 {
	font-size: 24px;
	font-weight: 600;
	color: #fff;
	margin-bottom: 8px;
}
.login-header p {
	color: rgba(255,255,255,0.6);
	font-size: 14px;
}
.form-group { margin-bottom: 20px; }
.form-group label {
	display: block;
	margin-bottom: 8px;
	font-size: 13px;
	font-weight: 500;
	color: rgba(255,255,255,0.8);
}
.form-group input {
	width: 100%;
	padding: 12px 16px;
	background: rgba(255,255,255,0.05);
	border: 1px solid rgba(255,255,255,0.1);
	border-radius: 10px;
	color: #fff;
	font-size: 14px;
	transition: all 0.2s;
}
.form-group input:focus {
	outline: none;
	border-color: #6366f1;
	background: rgba(255,255,255,0.08);
	box-shadow: 0 0 0 3px rgba(99,102,241,0.2);
}
.form-group input::placeholder { color: rgba(255,255,255,0.3); }
.btn {
	width: 100%;
	padding: 14px;
	background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
	border: none;
	border-radius: 10px;
	color: #fff;
	font-size: 15px;
	font-weight: 600;
	cursor: pointer;
	transition: all 0.2s;
}
.btn:hover { transform: translateY(-1px); box-shadow: 0 10px 20px rgba(99,102,241,0.3); }
.btn:active { transform: translateY(0); }
.btn:disabled { opacity: 0.6; cursor: not-allowed; transform: none; }
.error-msg {
	background: rgba(239,68,68,0.15);
	border: 1px solid rgba(239,68,68,0.3);
	color: #fca5a5;
	padding: 12px 16px;
	border-radius: 10px;
	font-size: 13px;
	margin-bottom: 20px;
	display: none;
}
.error-msg.show { display: block; }
.error-msg.locked {
	background: rgba(239,68,68,0.2);
	border-color: rgba(239,68,68,0.5);
}
.attempts-info {
	text-align: right;
	font-size: 12px;
	color: rgba(255,255,255,0.5);
	margin-top: -10px;
	margin-bottom: 16px;
}
.attempts-info.danger { color: #f87171; }
.footer-note {
	text-align: center;
	margin-top: 24px;
	font-size: 12px;
	color: rgba(255,255,255,0.4);
}
.secure-badge {
	display: inline-flex;
	align-items: center;
	gap: 6px;
	background: rgba(34,197,94,0.15);
	color: #4ade80;
	padding: 6px 12px;
	border-radius: 20px;
	font-size: 12px;
	margin-top: 12px;
}
@keyframes shake {
	0%,100% { transform: translateX(0); }
	25% { transform: translateX(-5px); }
	75% { transform: translateX(5px); }
}
.shake { animation: shake 0.3s ease-in-out; }
</style>
</head>
<body>
<div class="login-container">
	<div class="login-header">
		<div class="logo">🛡️</div>
		<h1>SCP:SL 监控控制台</h1>
		<p>安全登录以继续</p>
		<div class="secure-badge">🔐 双重验证保护</div>
	</div>

	<div class="error-msg" id="error-msg"></div>
	<div class="attempts-info" id="attempts-info"></div>

	<form id="login-form" onsubmit="doLogin(event)">
		<div class="form-group">
			<label>用户名</label>
			<input type="text" id="username" placeholder="请输入用户名" autocomplete="username" required>
		</div>
		<div class="form-group">
			<label>密码</label>
			<input type="password" id="password" placeholder="请输入密码" autocomplete="current-password" required>
		</div>
		<div class="form-group">
			<label>API 密钥（第二验证因素）</label>
			<input type="password" id="api_key" placeholder="请输入 API 密钥" autocomplete="off">
		</div>
		<button type="submit" class="btn" id="login-btn">登 录</button>
	</form>

	<div class="footer-note">
		连续 3 次错误将锁定账户<br>
		v2.0 · Go Powered
	</div>
</div>

<script>
async function doLogin(e) {
	e.preventDefault();
	const btn = document.getElementById('login-btn');
	const errMsg = document.getElementById('error-msg');
	const attemptsInfo = document.getElementById('attempts-info');
	const form = document.getElementById('login-form');

	btn.disabled = true;
	btn.textContent = '登录中...';
	errMsg.classList.remove('show', 'locked');

	const username = document.getElementById('username').value;
	const password = document.getElementById('password').value;
	const api_key = document.getElementById('api_key').value;

	try {
		const res = await fetch('/console/api/login', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ username, password, api_key })
		});
		const data = await res.json();

		if (res.ok && data.success) {
			window.location.href = data.must_change_password ? '/console/change-password' : '/console/dashboard';
		} else {
			errMsg.textContent = data.error || '登录失败';
			errMsg.classList.add('show');
			form.classList.add('shake');
			setTimeout(() => form.classList.remove('shake'), 300);

			if (data.locked) {
				errMsg.classList.add('locked');
				attemptsInfo.textContent = '🔒 账户已锁定，请联系管理员解锁';
				attemptsInfo.classList.add('danger');
			} else if (data.remaining !== undefined) {
				attemptsInfo.textContent = '剩余尝试次数: ' + data.remaining;
				if (data.remaining <= 1) {
					attemptsInfo.classList.add('danger');
				}
			}
		}
	} catch (err) {
		errMsg.textContent = '网络错误，请重试';
		errMsg.classList.add('show');
	}

	btn.disabled = false;
	btn.textContent = '登 录';
}
</script>
</body>
</html>`

var changePasswordPageHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>修改密码 - 监控控制台</title>
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body {
	font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "Noto Sans CJK SC", sans-serif;
	background: linear-gradient(135deg, #0f0c29 0%, #302b63 50%, #24243e 100%);
	min-height: 100vh;
	display: flex;
	align-items: center;
	justify-content: center;
	color: #e0e0e0;
}
.container {
	background: rgba(255,255,255,0.05);
	backdrop-filter: blur(20px);
	border: 1px solid rgba(255,255,255,0.1);
	border-radius: 20px;
	padding: 48px 40px;
	width: 100%;
	max-width: 420px;
	box-shadow: 0 25px 50px -12px rgba(0,0,0,0.5);
}
.header {
	text-align: center;
	margin-bottom: 32px;
}
.header .logo { font-size: 48px; margin-bottom: 12px; }
.header h1 { font-size: 24px; font-weight: 600; color: #fff; margin-bottom: 8px; }
.header p { color: rgba(255,255,255,0.6); font-size: 14px; }
.warning-note {
	background: rgba(251,191,36,0.12);
	border: 1px solid rgba(251,191,36,0.3);
	color: #fcd34d;
	padding: 10px 14px;
	border-radius: 10px;
	font-size: 13px;
	margin-bottom: 24px;
	text-align: center;
}
.form-group { margin-bottom: 20px; }
.form-group label {
	display: block;
	margin-bottom: 8px;
	font-size: 13px;
	font-weight: 500;
	color: rgba(255,255,255,0.8);
}
.form-group input {
	width: 100%;
	padding: 12px 16px;
	background: rgba(255,255,255,0.05);
	border: 1px solid rgba(255,255,255,0.1);
	border-radius: 10px;
	color: #fff;
	font-size: 14px;
	transition: all 0.2s;
}
.form-group input:focus {
	outline: none;
	border-color: #6366f1;
	background: rgba(255,255,255,0.08);
	box-shadow: 0 0 0 3px rgba(99,102,241,0.2);
}
.form-group input::placeholder { color: rgba(255,255,255,0.3); }
.hint { font-size: 12px; color: rgba(255,255,255,0.4); margin-top: 6px; }
.btn {
	width: 100%;
	padding: 14px;
	background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
	border: none;
	border-radius: 10px;
	color: #fff;
	font-size: 15px;
	font-weight: 600;
	cursor: pointer;
	transition: all 0.2s;
}
.btn:hover { transform: translateY(-1px); box-shadow: 0 10px 20px rgba(99,102,241,0.3); }
.btn:active { transform: translateY(0); }
.btn:disabled { opacity: 0.6; cursor: not-allowed; transform: none; }
.error-msg {
	background: rgba(239,68,68,0.15);
	border: 1px solid rgba(239,68,68,0.3);
	color: #fca5a5;
	padding: 12px 16px;
	border-radius: 10px;
	font-size: 13px;
	margin-bottom: 20px;
	display: none;
}
.error-msg.show { display: block; }
.footer-note {
	text-align: center;
	margin-top: 24px;
	font-size: 12px;
	color: rgba(255,255,255,0.4);
}
.footer-note a { color: rgba(255,255,255,0.6); text-decoration: none; }
.footer-note a:hover { color: #fff; }
</style>
</head>
<body>
<div class="container">
	<div class="header">
		<div class="logo">🔑</div>
		<h1>修改密码</h1>
		<p>首次登录或管理员重置后必须修改密码</p>
	</div>

	<div class="warning-note">⚠️ 密码至少 12 个字符，必须包含大写字母、小写字母、数字和特殊字符</div>

	<div class="error-msg" id="error-msg"></div>

	<form id="change-form" onsubmit="doChange(event)">
		<div class="form-group">
			<label>当前密码</label>
			<input type="password" id="current_password" placeholder="请输入当前密码" autocomplete="current-password" required>
		</div>
		<div class="form-group">
			<label>新密码</label>
			<input type="password" id="new_password" placeholder="至少 12 位，含大小写字母、数字和特殊字符" autocomplete="new-password" required>
		</div>
		<div class="form-group">
			<label>确认新密码</label>
			<input type="password" id="confirm_password" placeholder="再次输入新密码" autocomplete="new-password" required>
		</div>
		<button type="submit" class="btn" id="change-btn">确认修改</button>
	</form>

	<div class="footer-note">
		<a href="#" onclick="doLogout(event)">退出登录</a>
	</div>
</div>

<script>
async function doChange(e) {
	e.preventDefault();
	const btn = document.getElementById('change-btn');
	const errMsg = document.getElementById('error-msg');

	const currentPassword = document.getElementById('current_password').value;
	const newPassword = document.getElementById('new_password').value;
	const confirmPassword = document.getElementById('confirm_password').value;

	errMsg.classList.remove('show');

	if (newPassword !== confirmPassword) {
		errMsg.textContent = '两次输入的新密码不一致';
		errMsg.classList.add('show');
		return;
	}

	btn.disabled = true;
	btn.textContent = '提交中...';

	try {
		const res = await fetch('/console/api/change-password', {
			method: 'POST',
			headers: { 'Content-Type': 'application/json' },
			body: JSON.stringify({ current_password: currentPassword, new_password: newPassword })
		});
		const data = await res.json();

		if (res.ok && data.success) {
			window.location.href = '/console/dashboard';
		} else {
			errMsg.textContent = data.error || '修改失败';
			errMsg.classList.add('show');
		}
	} catch (err) {
		errMsg.textContent = '网络错误，请重试';
		errMsg.classList.add('show');
	}

	btn.disabled = false;
	btn.textContent = '确认修改';
}

async function doLogout(e) {
	e.preventDefault();
	await fetch('/console/api/logout', { method: 'POST' });
	window.location.href = '/console/login';
}
</script>
</body>
</html>`
