// Register page logic
document.getElementById('registerForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    const data = {
        username: document.getElementById('username').value,
        nickname: document.getElementById('nickname').value,
        password: document.getElementById('password').value
    };
    const alertBox = document.getElementById('alert');
    alertBox.innerHTML = '<div class="alert alert-success">正在注册...</div>';
    try {
        const res = await fetch('/api/register', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify(data)
        });
        const result = await res.json();
        if (result.code === 200) {
            alertBox.innerHTML = '<div class="alert alert-success">注册成功，即将跳转登录...</div>';
            setTimeout(() => location.href = '/static/login.html', 1500);
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