// Security demo page logic - uses addEventListener (no inline handlers, CSP-safe)

document.addEventListener('DOMContentLoaded', () => {
    document.getElementById('btnSqlVuln').addEventListener('click', () => testSQL(false));
    document.getElementById('btnSqlSafe').addEventListener('click', () => testSQL(true));
    document.getElementById('btnXssVuln').addEventListener('click', () => testXSS(false));
    document.getElementById('btnXssSafe').addEventListener('click', () => testXSS(true));
    document.getElementById('btnDosVuln').addEventListener('click', () => testDoS(false));
    document.getElementById('btnDosSafe').addEventListener('click', () => testDoS(true));
    document.getElementById('btnTamper').addEventListener('click', tamperMessage);
    document.getElementById('btnCheck').addEventListener('click', integrityCheck);
});

// SQL Injection test
async function testSQL(safe) {
    const input = document.getElementById(safe ? 'sqlInputSafe' : 'sqlInput').value;
    const resultBox = document.getElementById(safe ? 'sqlSafeResult' : 'sqlVulnResult');
    resultBox.textContent = '正在请求...';

    const url = safe ? '/api/attack/sql/safe' : '/api/attack/sql/vulnerable';
    const res = await fetch(url + '?username=' + encodeURIComponent(input));
    const data = await res.json();

    resultBox.textContent = JSON.stringify(data, null, 2);

    if (!safe && data.data && data.data.results && data.data.results.length > 0) {
        resultBox.textContent += '\n\n⚠️ 攻击成功！漏洞接口返回了所有用户数据！';
    } else if (safe) {
        resultBox.textContent += '\n\n✅ 防御成功！参数化查询未受注入影响。';
    }
}

// XSS test
async function testXSS(safe) {
    const input = document.getElementById(safe ? 'xssInputSafe' : 'xssInput').value;
    const resultBox = document.getElementById(safe ? 'xssSafeResult' : 'xssVulnResult');
    const renderBox = document.getElementById(safe ? 'xssSafeRender' : 'xssVulnRender');
    resultBox.textContent = '正在请求...';

    const url = safe ? '/api/attack/xss/safe' : '/api/attack/xss/vulnerable';
    const res = await fetch(url, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ content: input })
    });
    const data = await res.json();

    resultBox.textContent = JSON.stringify(data, null, 2);

    if (!safe) {
        // VULNERABLE: using innerHTML - event-handler payloads (img onerror, svg onload) execute!
        // Note: <script> tags inserted via innerHTML do NOT execute (browser security feature),
        // but <img onerror>, <svg onload> etc. DO execute - demonstrating real XSS danger.
        renderBox.innerHTML = data.data ? data.data.raw_content : '';
        resultBox.textContent += '\n\n⚠️ 攻击成功！innerHTML渲染触发了事件型payload。\n';
        resultBox.textContent += '说明：浏览器不会执行innerHTML插入的<script>标签，\n';
        resultBox.textContent += '但<img onerror>、<svg onload>等事件型payload仍会触发弹窗！\n';
        resultBox.textContent += '请观察页面是否弹出了alert提示框。';
    } else {
        // SAFE: using textContent - script shown as text (XSS defense)
        const escaped = data.data ? data.data.escaped_content : '';
        renderBox.textContent = escaped;
        resultBox.textContent += '\n\n✅ 防御成功！HTML转义后脚本作为文本显示，不会执行。';
    }
}

// DoS test
async function testDoS(safe) {
    const count = parseInt(document.getElementById(safe ? 'dosCountSafe' : 'dosCount').value);
    const resultBox = document.getElementById(safe ? 'dosSafeResult' : 'dosVulnResult');
    resultBox.textContent = '正在发起 ' + count + ' 个并发请求...';

    const url = safe ? '/api/ping' : '/api/attack/dos/vulnerable';
    const startTime = Date.now();
    let success = 0;
    let failed = 0;
    let rateLimited = 0;

    const promises = [];
    for (let i = 0; i < count; i++) {
        promises.push(
            fetch(url, {
                headers: safe ? { 'Authorization': 'Bearer ' + (localStorage.getItem('token') || '') } : {}
            }).then(r => {
                if (r.status === 429) {
                    rateLimited++;
                    return null;
                }
                success++;
                return r.json();
            }).catch(() => {
                failed++;
            })
        );
    }

    await Promise.all(promises);
    const elapsed = Date.now() - startTime;

    resultBox.textContent = '测试完成！\n';
    resultBox.textContent += '总请求数: ' + count + '\n';
    resultBox.textContent += '成功: ' + success + '\n';
    resultBox.textContent += '被限流(429): ' + rateLimited + '\n';
    resultBox.textContent += '失败: ' + failed + '\n';
    resultBox.textContent += '耗时: ' + elapsed + 'ms\n\n';

    if (!safe) {
        resultBox.textContent += '⚠️ 漏洞接口无限制，所有请求都被处理，服务器资源被消耗！\n';
        resultBox.textContent += '在真实场景中，大量此类请求会导致服务不可用。';
    } else {
        if (rateLimited > 0) {
            resultBox.textContent += '✅ 防御生效！速率限制中间件拦截了 ' + rateLimited + ' 个请求（返回429）。\n';
            resultBox.textContent += '超过限制的请求被拒绝，保护了服务器资源。';
        } else {
            resultBox.textContent += '当前请求数未超过限制（120次/分钟）。增加请求数可触发限流。';
        }
    }
}

// Tamper attack demo - simulates direct DB modification
async function tamperMessage() {
    const id = document.getElementById('tamperId').value;
    const content = document.getElementById('tamperContent').value;
    const resultBox = document.getElementById('tamperResult');
    resultBox.textContent = '正在模拟篡改数据库...';

    const res = await fetch('/api/attack/tamper/' + id, {
        method: 'POST',
        headers: {
            'Content-Type': 'application/json',
            'Authorization': 'Bearer ' + (localStorage.getItem('token') || '')
        },
        body: JSON.stringify({ content: content })
    });
    const data = await res.json();
    resultBox.textContent = JSON.stringify(data, null, 2);
    if (data.code === 200) {
        resultBox.textContent += '\n\n⚠️ 已模拟直接修改数据库！curr_hash 未更新。\n现在点"执行完整性校验"即可检测到篡改。';
    }
}

// Integrity check - hash chain verification
async function integrityCheck() {
    const resultBox = document.getElementById('checkResult');
    resultBox.textContent = '正在扫描全部哈希链...';

    const res = await fetch('/api/integrity/check', {
        headers: { 'Authorization': 'Bearer ' + (localStorage.getItem('token') || '') }
    });
    const data = await res.json();
    if (data.code !== 200) {
        resultBox.textContent = '校验失败: ' + JSON.stringify(data);
        return;
    }
    const d = data.data;
    let txt = '扫描完成！\n';
    txt += '私信: ' + d.total_messages + ' 条 | 群消息: ' + d.total_group_msgs + ' 条 | 审计日志: ' + d.total_audit_logs + ' 条\n';
    txt += '状态: ' + d.status + '\n';
    txt += '发现异常: ' + d.alert_count + ' 处\n\n';
    if (d.alert_count > 0) {
        txt += '🚨 篡改详情：\n';
        d.alerts.forEach(a => {
            txt += '- [' + a.target_type + ' #' + a.target_id + '] ' + a.reason + '\n';
            txt += '  期望: ' + (a.expected || '').substring(0, 32) + '...\n';
            txt += '  实际: ' + (a.actual || '').substring(0, 32) + '...\n\n';
        });
        txt += '✅ 哈希链成功定位了被篡改的记录！管理员已收到实时告警。';
    } else {
        txt += '✅ 所有哈希链完整，未发现篡改。';
    }
    resultBox.textContent = txt;
}