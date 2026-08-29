package web

// Landing page for the open-source project
var landingPageHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width, initial-scale=1.0">
<title>Server Health Monitor — 开源服务器运维与自动修复平台</title>
<meta name="description" content="面向生产环境的通用服务器运维平台，提供环境识别、健康监控、风险分析、告警和受控修复。">
<style>
* { margin: 0; padding: 0; box-sizing: border-box; }
body {
	font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", "Noto Sans CJK SC", sans-serif;
	background: #0a0a1a;
	color: #e2e8f0;
	overflow-x: hidden;
}

/* Hero Section */
.hero {
	position: relative;
	min-height: 100vh;
	display: flex;
	align-items: center;
	justify-content: center;
	text-align: center;
	padding: 60px 20px;
	overflow: hidden;
}
.hero-bg {
	position: absolute;
	top: 0; left: 0; right: 0; bottom: 0;
	background: 
		radial-gradient(ellipse at 20% 50%, rgba(99,102,241,0.15) 0%, transparent 50%),
		radial-gradient(ellipse at 80% 50%, rgba(139,92,246,0.15) 0%, transparent 50%),
		radial-gradient(ellipse at 50% 100%, rgba(59,130,246,0.1) 0%, transparent 50%);
}
.hero-grid {
	position: absolute;
	top: 0; left: 0; right: 0; bottom: 0;
	background-image: 
		linear-gradient(rgba(99,102,241,0.05) 1px, transparent 1px),
		linear-gradient(90deg, rgba(99,102,241,0.05) 1px, transparent 1px);
	background-size: 50px 50px;
	mask-image: radial-gradient(ellipse at center, black 30%, transparent 70%);
	-webkit-mask-image: radial-gradient(ellipse at center, black 30%, transparent 70%);
}
.hero-content {
	position: relative;
	z-index: 10;
	max-width: 800px;
}
.hero-badge {
	display: inline-flex;
	align-items: center;
	gap: 8px;
	background: rgba(99,102,241,0.15);
	border: 1px solid rgba(99,102,241,0.3);
	border-radius: 999px;
	padding: 8px 18px;
	font-size: 13px;
	color: #a5b4fc;
	margin-bottom: 24px;
}
.hero-badge .dot {
	width: 8px; height: 8px;
	background: #4ade80;
	border-radius: 50%;
	animation: pulse 2s infinite;
}
@keyframes pulse {
	0%, 100% { opacity: 1; transform: scale(1); }
	50% { opacity: 0.6; transform: scale(1.2); }
}
.hero h1 {
	font-size: clamp(36px, 6vw, 64px);
	font-weight: 800;
	line-height: 1.1;
	margin-bottom: 20px;
	background: linear-gradient(135deg, #fff 0%, #a5b4fc 50%, #818cf8 100%);
	-webkit-background-clip: text;
	-webkit-text-fill-color: transparent;
	background-clip: text;
}
.hero p.lead {
	font-size: clamp(16px, 2vw, 20px);
	color: #94a3b8;
	margin-bottom: 36px;
	line-height: 1.7;
	max-width: 600px;
	margin-left: auto;
	margin-right: auto;
}
.hero-cta {
	display: flex;
	gap: 14px;
	justify-content: center;
	flex-wrap: wrap;
	margin-bottom: 48px;
}
.btn {
	display: inline-flex;
	align-items: center;
	gap: 8px;
	padding: 14px 28px;
	border-radius: 10px;
	font-size: 15px;
	font-weight: 600;
	text-decoration: none;
	transition: all 0.2s;
	cursor: pointer;
	border: none;
}
.btn-primary {
	background: linear-gradient(135deg, #6366f1 0%, #8b5cf6 100%);
	color: #fff;
	box-shadow: 0 4px 20px rgba(99,102,241,0.4);
}
.btn-primary:hover {
	transform: translateY(-2px);
	box-shadow: 0 8px 30px rgba(99,102,241,0.5);
}
.btn-secondary {
	background: rgba(255,255,255,0.05);
	border: 1px solid rgba(255,255,255,0.1);
	color: #e2e8f0;
}
.btn-secondary:hover {
	background: rgba(255,255,255,0.1);
	border-color: rgba(255,255,255,0.2);
	transform: translateY(-2px);
}

/* Stats bar */
.hero-stats {
	display: flex;
	justify-content: center;
	gap: 48px;
	flex-wrap: wrap;
}
.stat-item {
	text-align: center;
}
.stat-value {
	font-size: 32px;
	font-weight: 700;
	color: #fff;
}
.stat-label {
	font-size: 13px;
	color: #64748b;
	margin-top: 4px;
}

/* Features Section */
.section {
	padding: 100px 20px;
	max-width: 1200px;
	margin: 0 auto;
}
.section-header {
	text-align: center;
	margin-bottom: 60px;
}
.section-tag {
	display: inline-block;
	background: rgba(99,102,241,0.15);
	color: #a5b4fc;
	padding: 6px 14px;
	border-radius: 999px;
	font-size: 12px;
	font-weight: 600;
	letter-spacing: 1px;
	text-transform: uppercase;
	margin-bottom: 16px;
}
.section-header h2 {
	font-size: clamp(28px, 4vw, 42px);
	font-weight: 700;
	color: #fff;
	margin-bottom: 16px;
}
.section-header p {
	font-size: 17px;
	color: #94a3b8;
	max-width: 600px;
	margin: 0 auto;
	line-height: 1.7;
}

.features-grid {
	display: grid;
	grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
	gap: 20px;
}
.feature-card {
	background: rgba(30,41,59,0.5);
	border: 1px solid rgba(71,85,105,0.5);
	border-radius: 16px;
	padding: 32px;
	transition: all 0.3s;
	position: relative;
	overflow: hidden;
}
.feature-card::before {
	content: '';
	position: absolute;
	top: 0; left: 0; right: 0;
	height: 2px;
	background: linear-gradient(90deg, transparent, #6366f1, transparent);
	opacity: 0;
	transition: opacity 0.3s;
}
.feature-card:hover {
	transform: translateY(-4px);
	border-color: rgba(99,102,241,0.5);
	background: rgba(30,41,59,0.8);
}
.feature-card:hover::before {
	opacity: 1;
}
.feature-icon {
	width: 56px; height: 56px;
	background: linear-gradient(135deg, rgba(99,102,241,0.2), rgba(139,92,246,0.2));
	border-radius: 14px;
	display: flex;
	align-items: center;
	justify-content: center;
	font-size: 28px;
	margin-bottom: 20px;
}
.feature-card h3 {
	font-size: 20px;
	font-weight: 600;
	color: #fff;
	margin-bottom: 12px;
}
.feature-card p {
	color: #94a3b8;
	font-size: 14px;
	line-height: 1.7;
}

/* Architecture Section */
.arch-section {
	background: linear-gradient(180deg, transparent, rgba(99,102,241,0.05), transparent);
}
.arch-diagram {
	background: rgba(15,23,42,0.8);
	border: 1px solid rgba(71,85,105,0.5);
	border-radius: 16px;
	padding: 40px;
	margin-top: 40px;
	overflow-x: auto;
}
.arch-flow {
	display: flex;
	align-items: center;
	justify-content: center;
	gap: 16px;
	flex-wrap: wrap;
}
.arch-node {
	background: rgba(30,41,59,0.9);
	border: 1px solid rgba(99,102,241,0.3);
	border-radius: 12px;
	padding: 20px 24px;
	text-align: center;
	min-width: 140px;
}
.arch-node-icon {
	font-size: 32px;
	margin-bottom: 8px;
}
.arch-node-title {
	font-size: 14px;
	font-weight: 600;
	color: #fff;
	margin-bottom: 4px;
}
.arch-node-desc {
	font-size: 11px;
	color: #64748b;
}
.arch-arrow {
	font-size: 24px;
	color: #6366f1;
}
.arch-layer {
	text-align: center;
	margin: 30px 0 10px;
	font-size: 12px;
	color: #64748b;
	text-transform: uppercase;
	letter-spacing: 2px;
}

/* CTA Section */
.cta-section {
	text-align: center;
	padding: 100px 20px;
}
.cta-box {
	max-width: 600px;
	margin: 0 auto;
	background: linear-gradient(135deg, rgba(99,102,241,0.15), rgba(139,92,246,0.15));
	border: 1px solid rgba(99,102,241,0.3);
	border-radius: 20px;
	padding: 50px 40px;
}
.cta-box h2 {
	font-size: 28px;
	font-weight: 700;
	color: #fff;
	margin-bottom: 16px;
}
.cta-box p {
	color: #94a3b8;
	margin-bottom: 28px;
	line-height: 1.7;
}

/* Footer */
.footer {
	border-top: 1px solid rgba(71,85,105,0.3);
	padding: 30px 20px;
	text-align: center;
	color: #64748b;
	font-size: 13px;
}
.footer-links {
	display: flex;
	justify-content: center;
	gap: 24px;
	margin-bottom: 16px;
	flex-wrap: wrap;
}
.footer-links a {
	color: #94a3b8;
	text-decoration: none;
	transition: color 0.2s;
}
.footer-links a:hover { color: #fff; }

/* Nav */
.navbar {
	position: fixed;
	top: 0; left: 0; right: 0;
	z-index: 100;
	padding: 16px 32px;
	display: flex;
	justify-content: space-between;
	align-items: center;
	background: rgba(10,10,26,0.8);
	backdrop-filter: blur(10px);
	border-bottom: 1px solid rgba(71,85,105,0.2);
}
.nav-logo {
	display: flex;
	align-items: center;
	gap: 10px;
	font-weight: 700;
	font-size: 18px;
	color: #fff;
}
.nav-logo .icon { font-size: 24px; }
.nav-links {
	display: flex;
	gap: 24px;
	align-items: center;
}
.nav-links a {
	color: #94a3b8;
	text-decoration: none;
	font-size: 14px;
	transition: color 0.2s;
}
.nav-links a:hover { color: #fff; }
.nav-cta {
	padding: 8px 18px;
	background: linear-gradient(135deg, #6366f1, #8b5cf6);
	color: #fff !important;
	border-radius: 8px;
	font-weight: 500;
}

/* Responsive */
@media (max-width: 768px) {
	.nav-links a:not(.nav-cta) { display: none; }
	.hero-stats { gap: 24px; }
	.stat-value { font-size: 24px; }
	.section { padding: 60px 20px; }
	.arch-flow { flex-direction: column; }
	.arch-arrow { transform: rotate(90deg); }
}
</style>
</head>
<body>

<nav class="navbar">
	<div class="nav-logo">
		<span class="icon">🛡️</span>
		<span>Server Health Monitor</span>
	</div>
	<div class="nav-links">
		<a href="#features">功能</a>
		<a href="#architecture">架构</a>
		<a href="#security">安全</a>
		<a href="/console/login">控制台</a>
		<a href="/public/status">公开状态</a>
		<a href="https://github.com/Oxen112774/ServerHealthMonitor" class="nav-cta">⭐ GitHub</a>
	</div>
</nav>

<section class="hero">
	<div class="hero-bg"></div>
	<div class="hero-grid"></div>
	<div class="hero-content">
		<div class="hero-badge">
			<span class="dot"></span>
			开源 · Go 语言 · 生产就绪
		</div>
		<h1>服务器监控<br>与自动修复系统</h1>
		<p class="lead">
			面向服务器、容器和企业应用的生产级运维方案。<br>
			Go 语言编写，环境自适配，风险分析，告警与受控修复。
		</p>
		<div class="hero-cta">
			<a href="/console/login" class="btn btn-primary">
				🚀 进入控制台
			</a>
			<a href="#features" class="btn btn-secondary">
				了解更多
			</a>
		</div>
		<div class="hero-stats">
			<div class="stat-item">
				<div class="stat-value">&lt;20MB</div>
				<div class="stat-label">内存占用</div>
			</div>
			<div class="stat-item">
				<div class="stat-value">5s</div>
				<div class="stat-label">检测间隔</div>
			</div>
			<div class="stat-item">
				<div class="stat-value">2FA</div>
				<div class="stat-label">双重认证</div>
			</div>
			<div class="stat-item">
				<div class="stat-value">100%</div>
				<div class="stat-label">开源免费</div>
			</div>
		</div>
	</div>
</section>

<section class="section" id="features">
	<div class="section-header">
		<span class="section-tag">Core Features</span>
		<h2>为什么选择我们</h2>
		<p>从环境盘点到风险处理，覆盖服务器与应用运维全流程</p>
	</div>
	<div class="features-grid">
		<div class="feature-card">
			<div class="feature-icon">⚡</div>
			<h3>极致性能</h3>
			<p>Go 语言编写，单二进制零依赖，内存占用不到 20MB，对服务器资源几乎零影响。</p>
		</div>
		<div class="feature-card">
			<div class="feature-icon">🔄</div>
			<h3>自动修复</h3>
			<p>连续异常 N 次后自动重启服务，带冷却机制防止循环重启，守护你的游戏服。</p>
		</div>
		<div class="feature-card">
			<div class="feature-icon">📱</div>
			<h3>微信告警</h3>
			<p>集成 ServerChan 方糖，服务异常直接推送到微信，随时随地第一时间知晓。</p>
		</div>
		<div class="feature-card">
			<div class="feature-icon">📊</div>
			<h3>Prometheus</h3>
			<p>标准 /metrics 端点，无缝对接 Prometheus + Grafana，专业级可视化大屏。</p>
		</div>
		<div class="feature-card">
			<div class="feature-icon">🔐</div>
			<h3>企业级安全</h3>
			<p>密码 + API 密钥双因素认证，3 次错误永久锁定，完整审计日志可追溯。</p>
		</div>
		<div class="feature-card">
			<div class="feature-icon">🎛️</div>
			<h3>集中管理</h3>
			<p>统一管理多台服务器，一键部署 Agent，告别手动登录每台机器。</p>
		</div>
		<div class="feature-card">
			<div class="feature-icon">🐳</div>
			<h3>Docker 部署</h3>
			<p>Prometheus + Grafana 整套 Docker Compose 一键启动，开箱即用。</p>
		</div>
		<div class="feature-card">
			<div class="feature-icon">🤖</div>
			<h3>CI/CD</h3>
			<p>GitHub Actions 自动构建部署，代码 push 即上线，Python 脚本秒级传输。</p>
		</div>
		<div class="feature-card">
			<div class="feature-icon">🌐</div>
			<h3>跨平台</h3>
			<p>支持 Linux / Windows / macOS，Agent 跑在服务器，控制台跑在哪都行。</p>
		</div>
	</div>
</section>

<section class="section arch-section" id="architecture">
	<div class="section-header">
		<span class="section-tag">Architecture</span>
		<h2>系统架构</h2>
		<p>清晰的分层设计，每个组件独立可替换</p>
	</div>
	<div class="arch-diagram">
		<div class="arch-layer">用户层</div>
		<div class="arch-flow">
			<div class="arch-node">
				<div class="arch-node-icon">💻</div>
				<div class="arch-node-title">Web 浏览器</div>
				<div class="arch-node-desc">电脑 / 手机 / 平板</div>
			</div>
			<div class="arch-arrow">→</div>
			<div class="arch-node">
				<div class="arch-node-icon">📱</div>
				<div class="arch-node-title">微信</div>
				<div class="arch-node-desc">ServerChan 告警</div>
			</div>
		</div>
		
		<div class="arch-layer">控制层</div>
		<div class="arch-flow">
			<div class="arch-node">
				<div class="arch-node-icon">🎛️</div>
				<div class="arch-node-title">管理控制台</div>
				<div class="arch-node-desc">用户/服务器/审计</div>
			</div>
			<div class="arch-arrow">→</div>
			<div class="arch-node">
				<div class="arch-node-icon">📈</div>
				<div class="arch-node-title">Prometheus</div>
				<div class="arch-node-desc">指标采集存储</div>
			</div>
			<div class="arch-arrow">→</div>
			<div class="arch-node">
				<div class="arch-node-icon">📊</div>
				<div class="arch-node-title">Grafana</div>
				<div class="arch-node-desc">可视化大屏</div>
			</div>
		</div>

		<div class="arch-layer">Agent 层（每台服务器）</div>
		<div class="arch-flow">
			<div class="arch-node">
				<div class="arch-node-icon">🔍</div>
				<div class="arch-node-title">指标采集</div>
				<div class="arch-node-desc">CPU/内存/磁盘</div>
			</div>
			<div class="arch-arrow">→</div>
			<div class="arch-node">
				<div class="arch-node-icon">❤️</div>
				<div class="arch-node-title">健康检测</div>
				<div class="arch-node-desc">5秒级检测</div>
			</div>
			<div class="arch-arrow">→</div>
			<div class="arch-node">
				<div class="arch-node-icon">🔧</div>
				<div class="arch-node-title">自动修复</div>
				<div class="arch-node-desc">systemctl restart</div>
			</div>
			<div class="arch-arrow">→</div>
			<div class="arch-node">
				<div class="arch-node-icon">🔔</div>
				<div class="arch-node-title">告警通知</div>
				<div class="arch-node-desc">方糖/Webhook</div>
			</div>
		</div>

		<div class="arch-layer">业务层</div>
		<div class="arch-flow">
			<div class="arch-node">
				<div class="arch-node-icon">🎮</div>
				<div class="arch-node-title">SCP:SL 服务</div>
				<div class="arch-node-desc">scpsl-*.service</div>
			</div>
		</div>
	</div>
</section>

<section class="section" id="security">
	<div class="section-header">
		<span class="section-tag">Security</span>
		<h2>安全设计</h2>
		<p>从认证到部署，全方位保护你的基础设施</p>
	</div>
	<div class="features-grid">
		<div class="feature-card">
			<div class="feature-icon">🔑</div>
			<h3>双因素认证</h3>
			<p>密码 + API 密钥双重验证，即使密码泄露也无法登录。</p>
		</div>
		<div class="feature-card">
			<div class="feature-icon">🚫</div>
			<h3>暴力破解防护</h3>
			<p>连续 3 次错误永久锁定账户，只能由管理员手动解锁。</p>
		</div>
		<div class="feature-card">
			<div class="feature-icon">🛡️</div>
			<h3>密码安全</h3>
			<p>bcrypt 算法哈希存储密码，API 密钥使用 SHA-256 哈希。</p>
		</div>
		<div class="feature-card">
			<div class="feature-icon">📋</div>
			<h3>审计日志</h3>
			<p>所有操作完整记录，谁在什么时候做了什么，一目了然。</p>
		</div>
		<div class="feature-card">
			<div class="feature-icon">🏰</div>
			<h3>沙箱隔离</h3>
			<p>systemd 安全加固，内存/CPU 限制，服务运行在隔离环境。</p>
		</div>
		<div class="feature-card">
			<div class="feature-icon">🚦</div>
			<h3>限流防护</h3>
			<p>登录接口速率限制，防止暴力破解和 DoS 攻击。</p>
		</div>
	</div>
</section>

<section class="cta-section">
	<div class="cta-box">
		<h2>🚀 开始使用</h2>
		<p>开源免费，MIT 协议，<br>立即部署你的第一台监控服务器</p>
		<div style="display:flex;gap:12px;justify-content:center;flex-wrap:wrap">
			<a href="/console/login" class="btn btn-primary">进入控制台</a>
			<a href="https://github.com/Oxen112774/ServerHealthMonitor" class="btn btn-secondary">⭐ GitHub 仓库</a>
		</div>
	</div>
</section>

<footer class="footer">
	<div class="footer-links">
		<a href="#features">功能</a>
		<a href="#architecture">架构</a>
		<a href="#security">安全</a>
		<a href="/console/login">控制台</a>
		<a href="/public/status">公开状态</a>
		<a href="https://github.com/Oxen112774/ServerHealthMonitor">GitHub</a>
	</div>
	<p>© 2026 SCP:SL Monitor · Open Source · MIT License</p>
</footer>

<script>
// Smooth scroll
document.querySelectorAll('a[href^="#"]').forEach(anchor => {
	anchor.addEventListener('click', function(e) {
		const target = document.querySelector(this.getAttribute('href'));
		if (target) {
			e.preventDefault();
			target.scrollIntoView({ behavior: 'smooth' });
		}
	});
});

// Navbar background on scroll
window.addEventListener('scroll', function() {
	const nav = document.querySelector('.navbar');
	if (window.scrollY > 50) {
		nav.style.background = 'rgba(10,10,26,0.95)';
	} else {
		nav.style.background = 'rgba(10,10,26,0.8)';
	}
});
</script>

</body>
</html>`
