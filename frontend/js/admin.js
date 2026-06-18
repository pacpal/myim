// Tamper-proof audit center logic
if (!checkAuth()) throw new Error('Not authenticated');

document.addEventListener('DOMContentLoaded', () => {
    document.getElementById('btnLogout').addEventListener('click', logout);
    document.getElementById('tabAlerts').addEventListener('click', () => switchTab('alerts'));
    document.getElementById('tabLogs').addEventListener('click', () => switchTab('logs'));
    document.getElementById('tabCheck').addEventListener('click', () => switchTab('check'));
    document.getElementById('btnCheck').addEventListener('click', runIntegrityCheck);
    document.getElementById('paneAlerts').addEventListener('click', handleAlertClick);

    loadAlerts();
    loadLogs();
    connectSSE();
});

function switchTab(tab) {
    document.querySelectorAll('.tab').forEach(t => t.classList.remove('active'));
    document.getElementById('paneAlerts').style.display = 'none';
    document.getElementById('paneLogs').style.display = 'none';
    document.getElementById('paneCheck').style.display = 'none';
    if (tab === 'alerts') {
        document.getElementById('tabAlerts').classList.add('active');
        document.getElementById('paneAlerts').style.display = 'block';
    } else if (tab === 'logs') {
        document.getElementById('tabLogs').classList.add('active');
        document.getElementById('paneLogs').style.display = 'block';
    } else {
        document.getElementById('tabCheck').classList.add('active');
        document.getElementById('paneCheck').style.display = 'block';
    }
}

function handleAlertClick(e) {
    const btn = e.target.closest('[data-action="resolve"]');
    if (!btn) return;
    resolveAlert(parseInt(btn.dataset.id));
}

async function loadAlerts() {
    const pane = document.getElementById('paneAlerts');
    let res;
    try {
        res = await apiGet('/api/admin/integrity/alerts');
    } catch (e) {
        pane.innerHTML = '<div class="empty">⚠️ 网络错误，无法加载篡改告警</div>';
        return;
    }
    if (!res || res.code !== 200) {
        const msg = (res && res.message) ? res.message : '加载失败';
        pane.innerHTML = '<div class="empty">⚠️ ' + escapeHtml(msg) + '（请确认使用管理员账号登录）</div>';
        document.getElementById('statAlerts').textContent = 0;
        document.getElementById('statNew').textContent = 0;
        return;
    }
    if (!res.data || res.data.length === 0) {
        pane.innerHTML = '<div class="empty">暂无篡改告警，系统数据完整 ✅</div>';
        document.getElementById('statAlerts').textContent = 0;
        document.getElementById('statNew').textContent = 0;
        return;
    }
    document.getElementById('statAlerts').textContent = res.data.length;
    document.getElementById('statNew').textContent = res.data.filter(a => !a.handled).length;
    pane.innerHTML = res.data.map(a => {
        const cls = a.handled ? 'resolved' : '';
        const status = a.handled ? '<span style="color:#2ecc71">已处理</span>' : '<span style="color:#e74c3c">● 新告警</span>';
        const btn = a.handled ? '' : '<button class="btn btn-sm" data-action="resolve" data-id="' + a.id + '">标记已处理</button>';
        return '<div class="alert-item ' + cls + '">' +
            '<strong>🚨 ' + escapeHtml(a.reason) + '</strong> ' + status + '<br>' +
            '<span>目标: ' + escapeHtml(a.target_type) + ' #' + a.target_id + '</span><br>' +
            '<span class="hash">期望哈希: ' + escapeHtml(a.expected_hash) + '</span><br>' +
            '<span class="hash">实际哈希: ' + escapeHtml(a.actual_hash) + '</span><br>' +
            '<span style="color:#888; font-size:12px;">' + escapeHtml(a.created_at) + '</span> ' + btn +
            '</div>';
    }).join('');
}

async function loadLogs() {
    const pane = document.getElementById('paneLogs');
    let res;
    try {
        res = await apiGet('/api/admin/audit/logs');
    } catch (e) {
        pane.innerHTML = '<div class="empty">⚠️ 网络错误，无法加载审计日志</div>';
        return;
    }
    if (!res || res.code !== 200) {
        const msg = (res && res.message) ? res.message : '加载失败';
        pane.innerHTML = '<div class="empty">⚠️ ' + escapeHtml(msg) + '（请确认使用管理员账号登录）</div>';
        document.getElementById('statLogs').textContent = 0;
        return;
    }
    if (!res.data || res.data.length === 0) {
        pane.innerHTML = '<div class="empty">暂无审计日志</div>';
        document.getElementById('statLogs').textContent = 0;
        return;
    }
    document.getElementById('statLogs').textContent = res.data.length;
    pane.innerHTML = res.data.map(l => {
        const color = l.action === 'tamper' ? '#e74c3c' : (l.action === 'integrity_check' ? '#f39c12' : '#667eea');
        return '<div class="log-item" style="border-left-color:' + color + '">' +
            '<strong>[' + escapeHtml(l.action) + ']</strong> ' + escapeHtml(l.detail) + '<br>' +
            '<span style="color:#888; font-size:12px;">' + escapeHtml(l.actor_name || '匿名') + ' | IP:' + escapeHtml(l.ip) + ' | ' + escapeHtml(l.created_at) + '</span><br>' +
            '<span class="hash">链哈希: ' + escapeHtml(l.hash) + '</span>' +
            '</div>';
    }).join('');
}

async function runIntegrityCheck() {
    const box = document.getElementById('checkResult');
    box.innerHTML = '<div class="empty">正在扫描全部哈希链...</div>';
    const res = await apiGet('/api/integrity/check');
    if (!res || res.code !== 200) {
        box.innerHTML = '<div class="empty">校验失败</div>';
        return;
    }
    const d = res.data;
    document.getElementById('statMessages').textContent = d.total_messages;
    if (d.alert_count === 0) {
        box.innerHTML = '<div class="alert-item" style="border-left-color:#2ecc71">' +
            '<strong>✅ 完整性校验通过</strong><br>' +
            '扫描了 ' + d.total_messages + ' 条私信、' + d.total_group_msgs + ' 条群消息、' + d.total_audit_logs + ' 条审计日志<br>' +
            '未发现任何篡改，哈希链完整。</div>';
    } else {
        box.innerHTML = '<div class="alert-item">' +
            '<strong>🚨 发现 ' + d.alert_count + ' 处异常！</strong></div>' +
            d.alerts.map(a => '<div class="alert-item">' +
                '<strong>' + escapeHtml(a.reason) + '</strong><br>' +
                '<span>目标: ' + escapeHtml(a.target_type) + ' #' + a.target_id + '</span><br>' +
                '<span class="hash">期望: ' + escapeHtml(a.expected) + '</span><br>' +
                '<span class="hash">实际: ' + escapeHtml(a.actual) + '</span></div>').join('');
        loadAlerts();
    }
}

function showToast(msg) {
    const t = document.createElement('div');
    t.className = 'toast';
    t.textContent = msg;
    document.body.appendChild(t);
    setTimeout(() => t.remove(), 5000);
}

function connectSSE() {
    const token = getToken();
    const es = new EventSource('/api/sse?token=' + encodeURIComponent(token));
    es.addEventListener('integrity_alert', (e) => {
        const alerts = JSON.parse(e.data);
        showToast('🚨 检测到篡改！发现 ' + alerts.length + ' 处异常');
        loadAlerts();
    });
    es.onerror = () => {
        es.close();
        setTimeout(() => { if (getToken()) connectSSE(); }, 3000);
    };
}

async function resolveAlert(id) {
    await apiPut('/api/admin/integrity/alerts/' + id + '/resolve', {});
    loadAlerts();
}

async function logout() {
    await apiPost('/api/logout');
    localStorage.clear();
    location.href = '/static/login.html';
}

function escapeHtml(str) {
    if (!str) return '';
    return String(str)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
}