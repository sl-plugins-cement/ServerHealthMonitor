package web

var dashboardHTML = `<!DOCTYPE html>
<html lang="zh-CN">
<head>
<meta charset="UTF-8">
<meta name="viewport" content="width=device-width,initial-scale=1.0,maximum-scale=1.0,user-scalable=no">
<meta name="theme-color" content="#4a90d9">
<meta name="apple-mobile-web-app-capable" content="yes">
<meta name="apple-mobile-web-app-status-bar-style" content="default">
<title>服务器运维监控</title>
<style>
:root{
--bg:#f0f2f5;--card:#fff;--primary:#4a90d9;--primary-dark:#357abd;--primary-light:#e8f1fb;
--success:#52c41a;--success-light:#f6ffed;--warning:#faad14;--warning-light:#fffbe6;
--error:#ff4d4f;--error-light:#fff2f0;--info:#1890ff;--info-light:#e6f7ff;
--text:#1a1a2e;--text2:#8c8c8c;--text3:#bfbfbf;--border:#e8e8e8;--border-light:#f0f0f0;
--radius:16px;--radius-sm:10px;
--shadow:0 2px 12px rgba(0,0,0,.06);--shadow-h:0 8px 28px rgba(0,0,0,.12);
--ts:.35s cubic-bezier(.4,0,.2,1);
--font:-apple-system,"PingFang SC","Microsoft YaHei","Segoe UI",sans-serif;
}
[data-theme="dark"]{
--bg:#0d1117;--card:#161b22;--primary:#4a90d9;--primary-dark:#357abd;--primary-light:#1a2332;
--success:#52c41a;--success-light:#16261a;--warning:#faad14;--warning-light:#2a2118;
--error:#ff4d4f;--error-light:#2a1213;--info:#1890ff;--info-light:#0d1f2e;
--text:#e6edf3;--text2:#8b949e;--text3:#6e7681;--border:#30363d;--border-light:#21262d;
--shadow:0 2px 12px rgba(0,0,0,.3);--shadow-h:0 8px 28px rgba(0,0,0,.4);
}
*{margin:0;padding:0;box-sizing:border-box}
body{font-family:var(--font);background:var(--bg);color:var(--text);min-height:100vh;transition:background var(--ts),color var(--ts);-webkit-font-smoothing:antialiased}
::-webkit-scrollbar{width:6px;height:6px}
::-webkit-scrollbar-track{background:transparent}
::-webkit-scrollbar-thumb{background:var(--border);border-radius:3px}

.header{background:linear-gradient(135deg,#4a90d9 0%,#357abd 50%,#2c6cb0 100%);color:#fff;padding:0;position:sticky;top:0;z-index:100;box-shadow:0 2px 16px rgba(74,144,217,.25)}
.header-inner{max-width:1280px;margin:0 auto;padding:14px 20px;display:flex;align-items:center;justify-content:space-between;gap:12px}
.header-left{display:flex;align-items:center;gap:14px;min-width:0}
.header-logo{width:44px;height:44px;background:rgba(255,255,255,.18);border-radius:12px;display:flex;align-items:center;justify-content:center;font-size:24px;flex-shrink:0}
.header-title{font-size:17px;font-weight:700;letter-spacing:.3px;white-space:nowrap}
.header-sub{font-size:11px;opacity:.8;margin-top:2px;white-space:nowrap;overflow:hidden;text-overflow:ellipsis}
.header-right{display:flex;align-items:center;gap:8px;flex-shrink:0}
.conn-pill{display:inline-flex;align-items:center;gap:6px;font-size:11px;background:rgba(255,255,255,.15);padding:5px 12px;border-radius:20px;white-space:nowrap}
.conn-dot{width:8px;height:8px;border-radius:50%;background:#52c41a;box-shadow:0 0 6px rgba(82,196,26,.6);animation:pulse 2s infinite}
.conn-dot.off{background:#ff4d4f;box-shadow:0 0 6px rgba(255,77,79,.6);animation:none}
@keyframes pulse{0%,100%{opacity:1}50%{opacity:.4}}
.btn-icon{width:38px;height:38px;border:none;background:rgba(255,255,255,.15);color:#fff;border-radius:10px;cursor:pointer;font-size:17px;transition:all var(--ts);display:flex;align-items:center;justify-content:center}
.btn-icon:hover{background:rgba(255,255,255,.28);transform:translateY(-1px)}

.container{max-width:1280px;margin:0 auto;padding:20px;padding-bottom:80px}
.section-title{font-size:13px;font-weight:700;color:var(--text2);margin:24px 0 12px;text-transform:uppercase;letter-spacing:.8px;display:flex;align-items:center;gap:8px}
.section-title .bar{width:3px;height:16px;background:var(--primary);border-radius:2px}

.metrics-grid{display:grid;grid-template-columns:repeat(auto-fill,minmax(280px,1fr));gap:16px}
.metric-card{background:var(--card);border-radius:var(--radius);padding:20px;box-shadow:var(--shadow);border:1px solid var(--border);transition:all var(--ts);position:relative;overflow:hidden}
.metric-card::before{content:'';position:absolute;top:0;left:0;right:0;height:3px;background:var(--metric-color,var(--primary));opacity:.8}
.metric-card:hover{box-shadow:var(--shadow-h);transform:translateY(-2px)}
.metric-head{display:flex;align-items:center;justify-content:space-between;margin-bottom:14px}
.metric-label{font-size:13px;color:var(--text2);font-weight:600}
.metric-icon{width:36px;height:36px;border-radius:10px;display:flex;align-items:center;justify-content:center;font-size:18px;background:var(--metric-bg,var(--primary-light));color:var(--metric-color,var(--primary))}
.metric-body{display:flex;align-items:baseline;gap:4px;margin-bottom:12px}
.metric-value{font-size:34px;font-weight:800;line-height:1;letter-spacing:-.5px}
.metric-unit{font-size:15px;font-weight:500;color:var(--text2)}
.metric-sub{font-size:12px;color:var(--text3);margin-left:8px;font-weight:400}
.progress-bar{height:8px;background:var(--border-light);border-radius:4px;overflow:hidden;margin-bottom:8px}
.progress-fill{height:100%;border-radius:4px;transition:width .8s cubic-bezier(.4,0,.2,1),background-color .3s;width:0}
.progress-fill.ok{background:linear-gradient(90deg,#52c41a,#73d13d)}
.progress-fill.warn{background:linear-gradient(90deg,#faad14,#ffc53d)}
.progress-fill.crit{background:linear-gradient(90deg,#ff4d4f,#ff7875)}
.sparkline{width:100%;height:36px;margin-top:4px}
.metric-detail{font-size:11px;color:var(--text3);margin-top:6px;display:flex;justify-content:space-between}

.instances{display:flex;flex-direction:column;gap:12px}
.inst-card{background:var(--card);border-radius:var(--radius);padding:16px 20px;box-shadow:var(--shadow);border:1px solid var(--border);transition:all var(--ts);display:flex;align-items:center;gap:16px}
.inst-card:hover{box-shadow:var(--shadow-h)}
.inst-dot{width:12px;height:12px;border-radius:50%;flex-shrink:0;transition:transform var(--ts)}
.inst-card:hover .inst-dot{transform:scale(1.2)}
.inst-dot.active{background:var(--success);box-shadow:0 0 10px rgba(82,196,26,.5)}
.inst-dot.inactive{background:var(--error);box-shadow:0 0 10px rgba(255,77,79,.5)}
.inst-dot.unknown{background:var(--text3)}
.inst-info{flex:1;min-width:0}
.inst-name{font-weight:700;font-size:15px}
.inst-detail{font-size:12px;color:var(--text2);margin-top:3px}
.inst-badges{display:flex;gap:6px;flex-wrap:wrap;justify-content:flex-end}
.badge{font-size:11px;padding:3px 10px;border-radius:6px;font-weight:600;letter-spacing:.3px}
.badge.green{background:var(--success-light);color:var(--success);border:1px solid var(--success)}
.badge.red{background:var(--error-light);color:var(--error);border:1px solid var(--error)}
.badge.gray{background:var(--border-light);color:var(--text2);border:1px solid var(--border)}

.alerts-panel{background:var(--card);border-radius:var(--radius);box-shadow:var(--shadow);border:1px solid var(--border);overflow:hidden}
.alerts-head{padding:16px 20px;border-bottom:1px solid var(--border);font-weight:700;font-size:14px;display:flex;align-items:center;justify-content:space-between}
.alerts-list{max-height:340px;overflow-y:auto}
.alert-item{padding:14px 20px;border-bottom:1px solid var(--border-light);display:flex;gap:12px;align-items:flex-start;transition:background var(--ts)}
.alert-item:hover{background:var(--border-light)}
.alert-item:last-child{border-bottom:none}
.alert-icon{width:28px;height:28px;border-radius:8px;display:flex;align-items:center;justify-content:center;font-size:14px;flex-shrink:0}
.alert-icon.warning{background:var(--warning-light);color:var(--warning)}
.alert-icon.critical{background:var(--error-light);color:var(--error)}
.alert-icon.recovery{background:var(--success-light);color:var(--success)}
.alert-body{flex:1;min-width:0}
.alert-title{font-size:13px;font-weight:700;margin-bottom:3px}
.alert-msg{font-size:12px;color:var(--text2);line-height:1.5}
.alert-time{font-size:11px;color:var(--text3);white-space:nowrap;margin-top:2px}
.no-data{padding:48px 20px;text-align:center;color:var(--text3);font-size:14px}
.no-data .emoji{font-size:32px;display:block;margin-bottom:8px;opacity:.5}

.toasts{position:fixed;top:80px;right:20px;z-index:9999;display:flex;flex-direction:column;gap:10px;max-width:380px;pointer-events:none}
.toast{background:var(--card);border-radius:var(--radius);padding:14px 16px;box-shadow:0 8px 32px rgba(0,0,0,.15);display:flex;gap:10px;align-items:flex-start;animation:slideIn .4s cubic-bezier(.4,0,.2,1);border-left:4px solid var(--warning);pointer-events:auto}
[data-theme="dark"] .toast{box-shadow:0 8px 32px rgba(0,0,0,.4)}
.toast.critical{border-left-color:var(--error)}
.toast.recovery{border-left-color:var(--success)}
.toast.removing{animation:slideOut .3s cubic-bezier(.4,0,.2,1) forwards}
.toast-body{flex:1;min-width:0}
.toast-title{font-size:13px;font-weight:700;margin-bottom:3px}
.toast-msg{font-size:12px;color:var(--text2);line-height:1.4}
.toast-close{background:none;border:none;color:var(--text3);cursor:pointer;font-size:18px;padding:0;line-height:1;opacity:.5;transition:opacity var(--ts)}
.toast-close:hover{opacity:1}
@keyframes slideIn{from{transform:translateX(120%);opacity:0}to{transform:translateX(0);opacity:1}}
@keyframes slideOut{to{transform:translateX(120%);opacity:0}}

.modal-overlay{position:fixed;inset:0;background:rgba(0,0,0,.45);z-index:5000;display:none;align-items:center;justify-content:center;animation:fadeIn .2s;backdrop-filter:blur(4px)}
.modal-overlay.show{display:flex}
@keyframes fadeIn{from{opacity:0}to{opacity:1}}
.modal{background:var(--card);border-radius:20px;padding:28px;width:90%;max-width:440px;box-shadow:0 16px 48px rgba(0,0,0,.25);animation:modalIn .35s cubic-bezier(.4,0,.2,1);max-height:90vh;overflow-y:auto}
@keyframes modalIn{from{transform:translateY(30px) scale(.95);opacity:0}to{transform:translateY(0) scale(1);opacity:1}}
.modal-title{font-size:18px;font-weight:700;margin-bottom:6px;display:flex;align-items:center;gap:8px}
.modal-desc{font-size:12px;color:var(--text2);margin-bottom:20px}
.setting-row{margin-bottom:18px}
.setting-label{font-size:12px;color:var(--text2);margin-bottom:8px;display:flex;justify-content:space-between;font-weight:600}
.setting-input{width:100%;padding:10px 14px;border:1px solid var(--border);border-radius:10px;font-size:14px;font-family:var(--font);background:var(--card);color:var(--text);transition:border-color var(--ts)}
.setting-input:focus{outline:none;border-color:var(--primary);box-shadow:0 0 0 3px rgba(74,144,217,.12)}
.setting-slider{width:100%;-webkit-appearance:none;height:6px;background:var(--border-light);border-radius:3px;outline:none}
.setting-slider::-webkit-slider-thumb{-webkit-appearance:none;width:20px;height:20px;border-radius:50%;background:var(--primary);cursor:pointer;box-shadow:0 2px 6px rgba(0,0,0,.2);transition:transform var(--ts)}
.setting-slider::-webkit-slider-thumb:hover{transform:scale(1.15)}
.toggle{position:relative;width:46px;height:26px;background:var(--border);border-radius:13px;cursor:pointer;transition:background var(--ts);flex-shrink:0}
.toggle.on{background:var(--primary)}
.toggle::after{content:'';position:absolute;top:3px;left:3px;width:20px;height:20px;background:#fff;border-radius:50%;transition:transform var(--ts);box-shadow:0 2px 4px rgba(0,0,0,.15)}
.toggle.on::after{transform:translateX(20px)}
.toggle-row{display:flex;align-items:center;justify-content:space-between}
.modal-actions{display:flex;gap:10px;justify-content:flex-end;margin-top:24px;padding-top:20px;border-top:1px solid var(--border-light)}
.btn{padding:10px 24px;border:none;border-radius:10px;font-size:14px;font-family:var(--font);cursor:pointer;transition:all var(--ts);font-weight:600}
.btn-primary{background:var(--primary);color:#fff}
.btn-primary:hover{background:var(--primary-dark);transform:translateY(-1px)}
.btn-secondary{background:var(--border-light);color:var(--text)}
.btn-secondary:hover{background:var(--border)}

.loading{display:flex;flex-direction:column;align-items:center;justify-content:center;min-height:200px;color:var(--text2);gap:12px}
.spinner{width:36px;height:36px;border:3px solid var(--border-light);border-top-color:var(--primary);border-radius:50%;animation:spin .8s linear infinite}
@keyframes spin{to{transform:rotate(360deg)}}

.footer{text-align:center;padding:20px;color:var(--text3);font-size:11px;line-height:1.6}
.footer a{color:var(--primary);text-decoration:none}

.bottom-nav{display:none;position:fixed;bottom:0;left:0;right:0;background:var(--card);border-top:1px solid var(--border);padding:6px 0;z-index:100;box-shadow:0 -2px 12px rgba(0,0,0,.06)}
.bottom-nav-inner{display:flex;justify-content:space-around;max-width:480px;margin:0 auto}
.nav-item{display:flex;flex-direction:column;align-items:center;gap:2px;padding:6px 16px;border:none;background:none;cursor:pointer;color:var(--text3);font-size:10px;transition:color var(--ts)}
.nav-item.active{color:var(--primary)}
.nav-item .nav-icon{font-size:20px}

.error-banner{background:var(--error-light);border:1px solid var(--error);border-radius:var(--radius);padding:16px 20px;margin-bottom:20px;display:flex;align-items:center;gap:12px;animation:fadeIn .3s}
.error-banner .icon{font-size:24px}
.error-banner .body{flex:1}
.error-banner .title{font-weight:700;font-size:14px;color:var(--error)}
.error-banner .msg{font-size:12px;color:var(--text2);margin-top:2px}
.retry-btn{padding:6px 16px;background:var(--error);color:#fff;border:none;border-radius:8px;cursor:pointer;font-size:12px;font-weight:600;transition:opacity var(--ts)}
.retry-btn:hover{opacity:.85}

@media(max-width:768px){
.header-inner{padding:12px 14px}
.header-logo{width:38px;height:38px;font-size:20px}
.header-title{font-size:15px}
.container{padding:14px;padding-bottom:70px}
.metrics-grid{grid-template-columns:1fr;gap:12px}
.metric-card{padding:16px}
.metric-value{font-size:28px}
.section-title{font-size:12px;margin:18px 0 10px}
.inst-card{padding:14px;gap:12px}
.inst-badges{flex-direction:column;align-items:flex-end}
.toasts{right:10px;left:10px;max-width:none;top:70px}
.bottom-nav{display:block}
.modal{padding:22px;border-radius:16px}
}
@media(max-width:380px){
.header-title{font-size:13px}
.metric-value{font-size:24px}
}
</style>
</head>
<body>
<div class="header">
  <div class="header-inner">
    <div class="header-left">
      <div class="header-logo">&#x1F6E0;</div>
      <div style="min-width:0">
        <div class="header-title">服务器运维监控</div>
        <div class="header-sub" id="serverInfo">正在加载...</div>
      </div>
    </div>
    <div class="header-right">
      <div class="conn-pill">
        <span class="conn-dot" id="connDot"></span>
        <span id="connText">连接中</span>
      </div>
      <button class="btn-icon" onclick="toggleTheme()" id="themeBtn" title="切换主题">&#x1F313;</button>
      <button class="btn-icon" onclick="toggleSettings()" title="设置">&#x2699;</button>
    </div>
  </div>
</div>

<div class="container">
  <div class="error-banner" id="errorBanner" style="display:none">
    <div class="icon">&#x26A0;</div>
    <div class="body">
      <div class="title">无法连接到服务器</div>
      <div class="msg" id="errorMsg">请检查网络连接</div>
    </div>
    <button class="retry-btn" onclick="fetchData()">重试</button>
  </div>

  <div class="section-title"><span class="bar"></span>系统资源</div>
  <div class="metrics-grid" id="metricsGrid">
    <div class="loading"><div class="spinner"></div><span>加载中...</span></div>
  </div>

  <div class="section-title"><span class="bar"></span>服务实例</div>
  <div class="instances" id="instancesList">
    <div class="loading"><div class="spinner"></div><span>加载中...</span></div>
  </div>

  <div class="section-title"><span class="bar"></span>告警记录</div>
  <div class="alerts-panel">
    <div class="alerts-head">
      <span>&#x1F514; 告警历史</span>
      <span id="alertCount" style="font-weight:400;font-size:12px;color:var(--text2)">0 条</span>
    </div>
    <div class="alerts-list" id="alertsList">
      <div class="no-data"><span class="emoji">&#x2705;</span>暂无告警记录</div>
    </div>
  </div>

  <div class="footer">
    Server Health Monitor &mdash; Go Edition<br>
    <a href="/metrics" target="_blank">Prometheus Metrics</a> &middot;
    <a href="https://github.com/Oxen112774/ServerHealthMonitor" target="_blank">GitHub</a>
  </div>
</div>

<div class="toasts" id="toasts"></div>

<div class="modal-overlay" id="settingsModal">
  <div class="modal">
    <div class="modal-title">&#x2699; 监控设置</div>
    <div class="modal-desc">调整刷新频率和告警阈值，设置自动保存</div>
    <div class="setting-row">
      <div class="setting-label"><span>刷新间隔</span><span id="intervalVal">3 秒</span></div>
      <input type="range" class="setting-slider" id="intervalSlider" min="1" max="15" value="3">
    </div>
    <div class="setting-row">
      <div class="setting-label"><span>CPU 告警阈值</span><span id="thrCpuVal">90%</span></div>
      <input type="range" class="setting-slider" id="thrCpu" min="50" max="100" value="90">
    </div>
    <div class="setting-row">
      <div class="setting-label"><span>内存告警阈值</span><span id="thrMemVal">90%</span></div>
      <input type="range" class="setting-slider" id="thrMem" min="50" max="100" value="90">
    </div>
    <div class="setting-row">
      <div class="setting-label"><span>磁盘告警阈值</span><span id="thrDiskVal">85%</span></div>
      <input type="range" class="setting-slider" id="thrDisk" min="50" max="100" value="85">
    </div>
    <div class="setting-row toggle-row">
      <div class="setting-label" style="margin:0"><span>浏览器推送通知</span></div>
      <div class="toggle" id="toggleNotif" onclick="this.classList.toggle('on')"></div>
    </div>
    <div class="setting-row toggle-row">
      <div class="setting-label" style="margin:0"><span>声音提醒</span></div>
      <div class="toggle" id="toggleSound" onclick="this.classList.toggle('on')"></div>
    </div>
    <div class="modal-actions">
      <button class="btn btn-secondary" onclick="toggleSettings()">关闭</button>
      <button class="btn btn-primary" onclick="saveSettings()">保存并应用</button>
    </div>
  </div>
</div>

<div class="bottom-nav">
  <div class="bottom-nav-inner">
    <button class="nav-item active" onclick="scrollToSection('metricsGrid',this)">
      <span class="nav-icon">&#x1F4CA;</span>资源
    </button>
    <button class="nav-item" onclick="scrollToSection('instancesList',this)">
      <span class="nav-icon">&#x1F3AE;</span>实例
    </button>
    <button class="nav-item" onclick="scrollToSection('alertsList',this)">
      <span class="nav-icon">&#x1F514;</span>告警
    </button>
    <button class="nav-item" onclick="toggleSettings()">
      <span class="nav-icon">&#x2699;</span>设置
    </button>
  </div>
</div>

<script>
let refreshTimer=null, lastAlerts=[], seenAlertIds=new Set(), settings={};

function loadSettings(){
  try{settings=JSON.parse(localStorage.getItem('scpsl_monitor_settings')||'{}');}catch(e){settings={};}
  settings=Object.assign({interval:3,cpu:90,mem:90,disk:85,load:4,notif:false,sound:false,theme:'light'},settings);
  applyTheme();
}
function saveSettingsL(){localStorage.setItem('scpsl_monitor_settings',JSON.stringify(settings));}
function applySettings(){
  if(refreshTimer)clearInterval(refreshTimer);
  refreshTimer=setInterval(fetchData,settings.interval*1000);
}
function applyTheme(){
  document.documentElement.setAttribute('data-theme',settings.theme);
  document.getElementById('themeBtn').innerHTML=settings.theme==='dark'?'&#x1F315;':'&#x1F313;';
}
function toggleTheme(){settings.theme=settings.theme==='dark'?'light':'dark';saveSettingsL();applyTheme();}
function toggleSettings(){
  const m=document.getElementById('settingsModal');
  m.classList.toggle('show');
  if(m.classList.contains('show')){
    document.getElementById('intervalSlider').value=settings.interval;
    document.getElementById('intervalVal').textContent=settings.interval+' 秒';
    document.getElementById('thrCpu').value=settings.cpu;
    document.getElementById('thrCpuVal').textContent=settings.cpu+'%';
    document.getElementById('thrMem').value=settings.mem;
    document.getElementById('thrMemVal').textContent=settings.mem+'%';
    document.getElementById('thrDisk').value=settings.disk;
    document.getElementById('thrDiskVal').textContent=settings.disk+'%';
    document.getElementById('toggleNotif').classList.toggle('on',settings.notif);
    document.getElementById('toggleSound').classList.toggle('on',settings.sound);
  }
}
function saveSettings(){
  settings.interval=parseInt(document.getElementById('intervalSlider').value);
  settings.cpu=parseInt(document.getElementById('thrCpu').value);
  settings.mem=parseInt(document.getElementById('thrMem').value);
  settings.disk=parseInt(document.getElementById('thrDisk').value);
  settings.notif=document.getElementById('toggleNotif').classList.contains('on');
  settings.sound=document.getElementById('toggleSound').classList.contains('on');
  saveSettingsL();applySettings();
  document.getElementById('settingsModal').classList.remove('show');
}
document.getElementById('intervalSlider').addEventListener('input',function(){document.getElementById('intervalVal').textContent=this.value+' 秒';});
document.getElementById('thrCpu').addEventListener('input',function(){document.getElementById('thrCpuVal').textContent=this.value+'%';});
document.getElementById('thrMem').addEventListener('input',function(){document.getElementById('thrMemVal').textContent=this.value+'%';});
document.getElementById('thrDisk').addEventListener('input',function(){document.getElementById('thrDiskVal').textContent=this.value+'%';});

function fmtBytes(n){
  if(!n||n<0)return '0 B';
  const u=['B','KB','MB','GB','TB'];let i=0;
  while(n>=1024&&i<u.length-1){n/=1024;i++;}
  return n.toFixed(1)+' '+u[i];
}
function fmtUptime(s){
  if(!s||s<0)return '0 秒';
  if(s<60)return Math.floor(s)+' 秒';
  if(s<3600)return Math.floor(s/60)+' 分'+Math.floor(s%60)+' 秒';
  if(s<86400)return Math.floor(s/3600)+' 小时'+Math.floor((s%3600)/60)+' 分';
  return Math.floor(s/86400)+' 天'+Math.floor((s%86400)/3600)+' 小时';
}
function levelClass(val,threshold){
  if(val>=threshold)return'crit';
  if(val>=threshold*0.8)return'warn';
  return'ok';
}
function sparkline(data,color){
  if(!data||data.length<2)return '<svg class="sparkline" viewBox="0 0 100 36"></svg>';
  const w=100,h=36,pad=2;
  const max=Math.max.apply(null,data);
  const min=Math.min.apply(null,data);
  const range=(max-min)||1;
  const pts=data.map(function(v,i){
    const x=pad+(i/(data.length-1))*(w-pad*2);
    const y=h-pad-((v-min)/range)*(h-pad*2);
    return x.toFixed(1)+','+y.toFixed(1);
  }).join(' ');
  const areaPts=pad+','+(h-pad)+' '+pts+' '+(w-pad)+','+(h-pad);
  return '<svg class="sparkline" viewBox="0 0 100 36" preserveAspectRatio="none">'+
    '<polygon points="'+areaPts+'" fill="'+color+'" opacity="0.12"/>'+
    '<polyline points="'+pts+'" fill="none" stroke="'+color+'" stroke-width="1.8" stroke-linejoin="round" stroke-linecap="round"/></svg>';
}
function scrollToSection(id,btn){
  document.querySelectorAll('.nav-item').forEach(function(el){el.classList.remove('active');});
  if(btn)btn.classList.add('active');
  document.getElementById(id).scrollIntoView({behavior:'smooth',block:'start'});
}

async function fetchData(){
  try{
    const res=await fetch('/api/status',{signal:AbortSignal.timeout(8000)});
    if(!res.ok)throw new Error('HTTP '+res.status);
    const data=await res.json();
    updateUI(data);
    setConn(true);
  }catch(e){
    setConn(false);
  }
}
function setConn(online){
  const dot=document.getElementById('connDot');
  const txt=document.getElementById('connText');
  const banner=document.getElementById('errorBanner');
  if(online){dot.className='conn-dot';txt.textContent='已连接';banner.style.display='none';}
  else{dot.className='conn-dot off';txt.textContent='已断开';banner.style.display='flex';}
}

function updateUI(data){
  const srv=data.server||{};
  const m=data.metrics||{};
  document.getElementById('serverInfo').textContent=
    (srv.hostname||'server')+' | 运行 '+fmtUptime(srv.uptime||0)+' | CPU x'+(srv.cpu_count||1);

  const grid=document.getElementById('metricsGrid');
  const cpuVal=m.cpu?m.cpu.percent:0;
  const memVal=m.memory?m.memory.percent:0;
  const memUsed=m.memory?m.memory.used:0;
  const memTotal=m.memory?m.memory.total:0;
  const diskVal=m.disk?m.disk.percent:0;
  const diskUsed=m.disk?m.disk.used:0;
  const diskTotal=m.disk?m.disk.total:0;
  const loadVal=m.load?m.load['1min']:0;
  const load5=m.load?m.load['5min']:0;
  const load15=m.load?m.load['15min']:0;
  const loadNorm=srv.cpu_count?loadVal/srv.cpu_count:0;
  const netVal=m.network||'disabled';
  const procVal=m.process_count||0;

  const hist=data.history||{cpu:[],memory:[],disk:[]};

  grid.innerHTML=
    card('CPU','&#x2697;',cpuVal,'%',settings.cpu,hist.cpu||[],'#4a90d9','')+
    card('内存','&#x1F4BE;',memVal,'%',settings.mem,hist.memory||[],'#722ed1',fmtBytes(memUsed)+' / '+fmtBytes(memTotal))+
    card('磁盘','&#x1F4BF;',diskVal,'%',settings.disk,hist.disk||[],'#fa8c16',fmtBytes(diskUsed)+' / '+fmtBytes(diskTotal))+
    card('系统负载','&#x26A1;',loadNorm,'',settings.load||4,[],'#13c2c2','1m:'+loadVal.toFixed(2)+' 5m:'+load5.toFixed(2)+' 15m:'+load15.toFixed(2))+
    cardNet('网络连通','&#x1F310;',netVal)+
    card('进程数','&#x1F916;',procVal,'',settings.proc||500,[],'#eb2f96','');

  const insts=data.instances||[];
  const il=document.getElementById('instancesList');
  if(!insts.length){
    il.innerHTML='<div class="no-data"><span class="emoji">&#x1F50D;</span>未检测到 SCP:SL 服务实例</div>';
  }else{
    il.innerHTML=insts.map(function(i){
      const cls=i.state==='active'?'active':(i.state==='inactive'||i.state==='failed'?'inactive':'unknown');
      const udpBadge=i.udp==='listening'?'<span class="badge green">UDP \u2713</span>':
        i.udp==='not-listening'?'<span class="badge red">UDP \u2717</span>':'<span class="badge gray">'+i.udp+'</span>';
      const stateBadge=i.state==='active'?'<span class="badge green">'+i.state+'</span>':
        '<span class="badge red">'+i.state+'</span>';
      return '<div class="inst-card"><div class="inst-dot '+cls+'"></div><div class="inst-info">'+
        '<div class="inst-name">'+i.service+'</div>'+
        '<div class="inst-detail">端口 '+i.port+' | 进程数: '+i.processes+'</div></div>'+
        '<div class="inst-badges">'+stateBadge+udpBadge+'</div></div>';
    }).join('');
  }

  const alerts=data.alerts||[];
  document.getElementById('alertCount').textContent=alerts.length+' 条';
  const al=document.getElementById('alertsList');
  if(!alerts.length){
    al.innerHTML='<div class="no-data"><span class="emoji">&#x2705;</span>暂无告警记录</div>';
  }else{
    al.innerHTML=alerts.map(function(a){
      const icon=a.type==='critical'?'\uD83D\uDD34':(a.type==='recovery'?'\u2705':'\u26A0');
      const iconBg=a.type==='critical'?'critical':(a.type==='recovery'?'recovery':'warning');
      return '<div class="alert-item"><div class="alert-icon '+iconBg+'">'+icon+'</div>'+
        '<div class="alert-body"><div class="alert-title">'+a.title+'</div>'+
        '<div class="alert-msg">'+a.message+'</div>'+
        '<div class="alert-time">'+(a.time||'')+'</div></div></div>';
    }).join('');
  }

  for(let i=0;i<alerts.length;i++){
    const a=alerts[i];
    const aid=a.time+'|'+a.title;
    if(!seenAlertIds.has(aid)){
      seenAlertIds.add(aid);
      showToast(a);
      if(settings.notif&&'Notification'in window&&Notification.permission==='granted'){
        new Notification(a.title,{body:a.message});
      }
    }
  }
}

function card(label,icon,val,unit,threshold,hist,color,sub){
  const cls=levelClass(val,threshold);
  const pct=Math.min(val,100);
  return '<div class="metric-card" style="--metric-color:'+color+';--metric-bg:'+color+'22">'+
    '<div class="metric-head"><span class="metric-label">'+label+'</span>'+
    '<span class="metric-icon">'+icon+'</span></div>'+
    '<div class="metric-body"><span class="metric-value">'+(typeof val==='number'?val.toFixed(1):val)+
    '</span><span class="metric-unit">'+unit+'</span>'+
    (sub?'<span class="metric-sub">'+sub+'</span>':'')+'</div>'+
    '<div class="progress-bar"><div class="progress-fill '+cls+'" style="width:'+pct+'%"></div></div>'+
    sparkline(hist,color)+
    '<div class="metric-detail"><span>实时</span><span>阈值 '+threshold+'</span></div></div>';
}

function cardNet(label,icon,val){
  const color=val==='reachable'?'#52c41a':(val==='unreachable'?'#ff4d4f':'#8c8c8c');
  const text=val==='reachable'?'正常':(val==='unreachable'?'不可达':'未启用');
  return '<div class="metric-card" style="--metric-color:'+color+';--metric-bg:'+color+'22">'+
    '<div class="metric-head"><span class="metric-label">'+label+'</span>'+
    '<span class="metric-icon">'+icon+'</span></div>'+
    '<div class="metric-body"><span class="metric-value" style="color:'+color+'">'+text+'</span></div>'+
    '<div class="metric-detail"><span>'+val+'</span><span>&nbsp;</span></div></div>';
}

function showToast(a){
  const tc=document.getElementById('toasts');
  const div=document.createElement('div');
  div.className='toast '+(a.type||'warning');
  div.innerHTML='<div class="toast-body"><div class="toast-title">'+a.title+'</div>'+
    '<div class="toast-msg">'+a.message+'</div></div>'+
    '<button class="toast-close" onclick="this.parentElement.remove()">&times;</button>';
  tc.appendChild(div);
  if(settings.sound){
    try{
      const ctx=new (window.AudioContext||window.webkitAudioContext)();
      const osc=ctx.createOscillator();
      osc.connect(ctx.destination);
      osc.frequency.value=800;osc.start();
      setTimeout(function(){osc.stop();},200);
    }catch(e){}
  }
  setTimeout(function(){
    div.classList.add('removing');
    setTimeout(function(){div.remove();},300);
  },10000);
}

loadSettings();
applySettings();
fetchData();
if('Notification'in window&&Notification.permission!=='granted'){
  setTimeout(function(){Notification.requestPermission();},3000);
}
document.getElementById('settingsModal').addEventListener('click',function(e){
  if(e.target===this)this.classList.remove('show');
});
</script>
</body>
</html>`
