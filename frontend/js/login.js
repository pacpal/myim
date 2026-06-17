// Login page logic
document.getElementById('loginForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    const data = {
        username: document.getElementById('username').value,
        password: document.getElementById('password').value
    };
    const alertBox = document.getElementById('alert');
    alertBox.innerHTML = '<div class="alert alert-success">正在登录...</div>';
    try {
        const res = await fetch('/api/login', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });
        const result = await res.json();
        if (result.code === 200) {
            localStorage.setItem('token', result.data.token);
            localStorage.setItem('user_id', result.data.user_id);
            localStorage.setItem('nickname', result.data.nickname);
            localStorage.setItem('role', result.data.role || 0);
            alertBox.innerHTML = '<div class="alert alert-success">登录成功，正在进入聊天...</div>';
            setTimeout(() => location.href = '/static/chat.html', 1000);
        } else {
            alertBox.innerHTML = '<div class="alert alert-error">' + escapeHtml(result.message) + '</div>';
        }
    } catch (err) {
        alertBox.innerHTML = '<div class="alert alert-error">网络错误：' + escapeHtml(err.message) + '</div>';
    }
});

function escapeHtml(str) {
    if (!str) return '';
    return String(str)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
}