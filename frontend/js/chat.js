// Chat page logic - CSP-safe (no inline handlers, uses addEventListener + event delegation)
if (!checkAuth()) throw new Error('Not authenticated');

let currentTab = 'friends';
let currentChat = null; // { type: 'friend'|'group', id: int, name: string }
let sseSource = null;

// Initialize on DOM ready
document.addEventListener('DOMContentLoaded', () => {
    document.getElementById('myNickname').textContent = localStorage.getItem('nickname') || '用户';
    document.getElementById('myAvatar').textContent = (localStorage.getItem('nickname') || 'U').charAt(0).toUpperCase();

    // Show audit center link for admins
    if (parseInt(localStorage.getItem('role') || '0') === 1) {
        const adminLink = document.getElementById('adminLink');
        if (adminLink) adminLink.style.display = 'block';
    }

    // Static event listeners
    document.getElementById('btnLogout').addEventListener('click', logout);
    document.getElementById('tabFriends').addEventListener('click', () => switchTab('friends'));
    document.getElementById('tabGroups').addEventListener('click', () => switchTab('groups'));
    document.getElementById('tabRequests').addEventListener('click', () => switchTab('requests'));
    document.getElementById('btnAddFriend').addEventListener('click', showAddModal);
    const btnOpenGroup = document.getElementById('btnOpenGroupModal');
    if (btnOpenGroup) btnOpenGroup.addEventListener('click', showGroupModal);
    document.getElementById('btnCloseAddModal').addEventListener('click', () => closeModal('addModal'));
    document.getElementById('btnCancelGroup').addEventListener('click', () => closeModal('groupModal'));
    document.getElementById('btnCreateGroup').addEventListener('click', createGroup);
    document.getElementById('searchKeyword').addEventListener('input', searchUsers);

    // Event delegation for dynamically generated list items
    document.getElementById('listContent').addEventListener('click', handleListClick);
    document.getElementById('searchResults').addEventListener('click', handleSearchClick);
    document.getElementById('chatContent').addEventListener('click', handleChatClick);
    document.getElementById('chatContent').addEventListener('keydown', handleChatKeydown);

    loadFriends();
    connectSSE();
});

// Event delegation handler for list items
function handleListClick(e) {
    const item = e.target.closest('[data-action]');
    if (!item) return;
    const action = item.dataset.action;
    if (action === 'openChat') {
        openChat(item.dataset.type, parseInt(item.dataset.id), item.dataset.name);
    } else if (action === 'acceptRequest') {
        acceptRequest(parseInt(item.dataset.id));
    } else if (action === 'rejectRequest') {
        rejectRequest(parseInt(item.dataset.id));
    } else if (action === 'showGroupModal') {
        showGroupModal();
    }
}

// Event delegation for search results
function handleSearchClick(e) {
    const item = e.target.closest('[data-action="sendRequest"]');
    if (!item) return;
    sendFriendRequest(parseInt(item.dataset.id));
}

// Event delegation for chat area buttons
function handleChatClick(e) {
    const item = e.target.closest('[data-action]');
    if (!item) return;
    const action = item.dataset.action;
    if (action === 'sendMessage') {
        sendMessage();
    } else if (action === 'deleteFriend') {
        deleteFriend(parseInt(item.dataset.id));
    } else if (action === 'viewMembers') {
        viewGroupMembers(parseInt(item.dataset.id));
    }
}

function handleChatKeydown(e) {
    if (e.target.id === 'msgInput' && e.key === 'Enter') {
        sendMessage();
    }
}

// SSE connection with auto-reconnect (weak network optimization)
let sseAuthFailCount = 0;
function connectSSE() {
    const token = getToken();
    if (!token) {
        location.href = '/static/login.html';
        return;
    }
    // 关键：确保只存在一个 EventSource，避免同时多重连
    if (sseSource) {
        try { sseSource.close(); } catch (e) {}
        sseSource = null;
    }
    sseSource = new EventSource('/api/sse?token=' + encodeURIComponent(token));

    sseSource.addEventListener('message', (e) => {
        sseAuthFailCount = 0;
        handleIncomingMessage(JSON.parse(e.data));
    });
    sseSource.addEventListener('group_message', (e) => {
        handleIncomingGroupMessage(JSON.parse(e.data));
    });
    sseSource.addEventListener('offline_message', (e) => {
        handleIncomingMessage(JSON.parse(e.data));
    });
    sseSource.addEventListener('friend_request', (e) => {
        const req = JSON.parse(e.data);
        alert('收到来自 ' + req.from_nickname + ' 的好友请求');
        if (currentTab === 'requests') loadRequests();
    });
    sseSource.addEventListener('presence', (e) => {
        if (currentTab === 'friends') loadFriends();
    });
    sseSource.onerror = () => {
        // EventSource 原生已自带自动重连，只有当服务端主动拒绝（CLOSED）时才需要手动介入
        if (sseSource && sseSource.readyState === EventSource.CLOSED) {
            sseAuthFailCount++;
            // 用轻量接口验证 token，若是 401 则跳登录页；否则手动重连一次
            if (sseAuthFailCount >= 2) {
                fetch('/api/ping', { headers: { 'Authorization': 'Bearer ' + token } })
                    .then(r => r.json())
                    .then(data => {
                        if (data && data.code === 401) {
                            localStorage.clear();
                            location.href = '/static/login.html';
                        } else {
                            sseAuthFailCount = 0;
                            setTimeout(() => { if (getToken()) connectSSE(); }, 3000);
                        }
                    })
                    .catch(() => {
                        setTimeout(() => { if (getToken()) connectSSE(); }, 3000);
                    });
            } else {
                setTimeout(() => { if (getToken()) connectSSE(); }, 3000);
            }
        }
        // 其余情况（OPEN 状态下的短暂网络抖动）由 EventSource 原生自动重连，不需要做任何事
    };
}

function handleIncomingMessage(msg) {
    if (currentChat && currentChat.type === 'friend' && currentChat.id === msg.from_user_id) {
        appendMessage('received', msg.from_name, msg.content, msg.created_at);
    } else {
        updateListPreview('friend', msg.from_user_id, msg.content);
    }
}

function handleIncomingGroupMessage(msg) {
    if (currentChat && currentChat.type === 'group' && currentChat.id === msg.group_id) {
        appendMessage('received', msg.from_name, msg.content, msg.created_at);
    } else {
        updateListPreview('group', msg.group_id, msg.content);
    }
}

function updateListPreview(type, id, content) {
    const item = document.querySelector('[data-' + type + '-id="' + id + '"] .last-msg');
    if (item) item.textContent = content;
}

// Tab switching
function switchTab(tab) {
    currentTab = tab;
    document.querySelectorAll('.tab').forEach(t => t.classList.toggle('active', t.dataset.tab === tab));
    if (tab === 'friends') loadFriends();
    else if (tab === 'groups') loadGroups();
    else if (tab === 'requests') loadRequests();
}

// Load friends list
async function loadFriends() {
    const res = await apiGet('/api/friends');
    const content = document.getElementById('listContent');
    if (!res || res.code !== 200 || !res.data || res.data.length === 0) {
        content.innerHTML = '<div class="empty-state">暂无好友<br><small>点击下方添加好友</small></div>';
        return;
    }
    content.innerHTML = res.data.map(f => {
        const name = escapeHtml(f.remark || f.nickname || f.username);
        const displayName = escapeHtml(f.nickname || f.username);
        return '<div class="list-item" data-action="openChat" data-type="friend" data-id="' + f.friend_id + '" data-name="' + displayName + '">' +
            '<div class="avatar">' + escapeHtml((f.nickname || f.username).charAt(0).toUpperCase()) + '</div>' +
            '<div class="info"><div class="name">' + name + '</div>' +
            '<div class="last-msg">' + (f.status ? '在线' : '离线') + '</div></div>' +
            '<div class="status-dot ' + (f.status ? 'online' : '') + '"></div></div>';
    }).join('');
}

// Load groups list
async function loadGroups() {
    const res = await apiGet('/api/groups');
    const content = document.getElementById('listContent');
    // 用 data-action 走事件委托，避免 innerHTML 破坏 addEventListener
    let html = '<div style="padding:12px;"><button class="btn btn-sm btn-primary" style="width:100%" data-action="showGroupModal">+ 创建群组</button></div>';

    if (!res || res.code !== 200 || !res.data || res.data.length === 0) {
        html += '<div class="empty-state">暂无群组</div>';
        content.innerHTML = html;
        return;
    }
    html += res.data.map(g => {
        return '<div class="list-item" data-action="openChat" data-type="group" data-id="' + g.id + '" data-name="' + escapeHtml(g.name) + '">' +
            '<div class="avatar">' + escapeHtml(g.name.charAt(0).toUpperCase()) + '</div>' +
            '<div class="info"><div class="name">' + escapeHtml(g.name) + '</div>' +
            '<div class="last-msg">群主: ' + escapeHtml(g.owner_name) + '</div></div></div>';
    }).join('');
    content.innerHTML = html;
}

// Load friend requests
async function loadRequests() {
    const res = await apiGet('/api/friends/requests');
    const content = document.getElementById('listContent');
    if (!res || res.code !== 200 || !res.data || res.data.length === 0) {
        content.innerHTML = '<div class="empty-state">暂无好友请求</div>';
        return;
    }
    content.innerHTML = res.data.map(r => {
        const name = escapeHtml(r.from_nickname || r.from_username);
        return '<div class="list-item">' +
            '<div class="avatar">' + escapeHtml((r.from_nickname || r.from_username).charAt(0).toUpperCase()) + '</div>' +
            '<div class="info"><div class="name">' + name + '</div>' +
            '<div class="last-msg">' + escapeHtml(r.message || '请求添加你为好友') + '</div></div>' +
            '<div style="display:flex; gap:6px;">' +
            '<button class="btn btn-sm btn-success" data-action="acceptRequest" data-id="' + r.id + '">接受</button>' +
            '<button class="btn btn-sm" data-action="rejectRequest" data-id="' + r.id + '">拒绝</button>' +
            '</div></div>';
    }).join('');
}

// Open chat
async function openChat(type, id, name) {
    currentChat = { type, id, name };
    document.querySelectorAll('.list-item').forEach(i => i.classList.remove('active'));
    const attr = type === 'friend' ? '[data-id="' + id + '"]' : '[data-id="' + id + '"]';
    const item = document.querySelector('[data-action="openChat"]' + attr);
    if (item) item.classList.add('active');

    const chatContent = document.getElementById('chatContent');
    let headerBtn = '';
    if (type === 'friend') {
        headerBtn = '<button class="btn btn-sm" data-action="deleteFriend" data-id="' + id + '">删除好友</button>';
    } else {
        headerBtn = '<button class="btn btn-sm" data-action="viewMembers" data-id="' + id + '">群成员</button>';
    }

    chatContent.innerHTML =
        '<div class="chat-header"><h3>' + escapeHtml(name) + '</h3>' + headerBtn + '</div>' +
        '<div class="messages" id="messages"></div>' +
        '<div class="message-input">' +
        '<input type="text" id="msgInput" placeholder="输入消息...">' +
        '<button data-action="sendMessage">发送</button></div>';

    // Load history
    if (type === 'friend') {
        const res = await apiGet('/api/messages/' + id + '?limit=50');
        if (res && res.code === 200 && res.data) {
            res.data.forEach(m => {
                const dir = m.from_user_id == localStorage.getItem('user_id') ? 'sent' : 'received';
                appendMessage(dir, m.from_name, m.content, m.created_at);
            });
        }
    } else {
        const res = await apiGet('/api/groups/' + id + '/messages?limit=50');
        if (res && res.code === 200 && res.data) {
            res.data.forEach(m => {
                const dir = m.from_user_id == localStorage.getItem('user_id') ? 'sent' : 'received';
                appendMessage(dir, m.from_name, m.content, m.created_at);
            });
        }
    }
    scrollMessages();
}

// Send message
async function sendMessage() {
    const input = document.getElementById('msgInput');
    if (!input) return;
    const content = input.value.trim();
    if (!content || !currentChat) return;

    input.value = '';
    appendMessage('sent', localStorage.getItem('nickname'), content, new Date().toLocaleString());

    if (currentChat.type === 'friend') {
        await apiPost('/api/messages', { to_user_id: currentChat.id, content: content, msg_type: 0 });
    } else {
        await apiPost('/api/groups/' + currentChat.id + '/messages', { content: content, msg_type: 0 });
    }
    scrollMessages();
}

// Append message to chat (XSS safe - uses textContent)
function appendMessage(dir, name, content, time) {
    const messages = document.getElementById('messages');
    if (!messages) return;
    const div = document.createElement('div');
    div.className = 'message ' + dir;
    const bubble = document.createElement('div');
    bubble.className = 'bubble';
    bubble.textContent = content; // Safe: textContent prevents XSS
    const meta = document.createElement('div');
    meta.className = 'meta';
    meta.textContent = (dir === 'received' ? name + ' · ' : '') + time;
    div.appendChild(bubble);
    div.appendChild(meta);
    messages.appendChild(div);
    scrollMessages();
}

function scrollMessages() {
    const messages = document.getElementById('messages');
    if (messages) messages.scrollTop = messages.scrollHeight;
}

// Friend request actions
async function acceptRequest(id) {
    await apiPut('/api/friends/requests/' + id + '/accept');
    loadRequests();
}
async function rejectRequest(id) {
    await apiPut('/api/friends/requests/' + id + '/reject');
    loadRequests();
}
async function deleteFriend(id) {
    if (!confirm('确定删除该好友？')) return;
    await apiDelete('/api/friends/' + id);
    loadFriends();
    document.getElementById('chatContent').innerHTML = '<div class="empty-state">选择一个好友或群组开始聊天</div>';
    currentChat = null;
}

// Search users
function showAddModal() {
    document.getElementById('addModal').classList.add('active');
    document.getElementById('searchResults').innerHTML = '';
    document.getElementById('searchKeyword').value = '';
}
function showGroupModal() {
    document.getElementById('groupModal').classList.add('active');
}

function closeModal(id) {
    document.getElementById(id).classList.remove('active');
}

async function searchUsers() {
    const keyword = document.getElementById('searchKeyword').value.trim();
    if (!keyword) {
        document.getElementById('searchResults').innerHTML = '';
        return;
    }
    const res = await apiGet('/api/users/search?keyword=' + encodeURIComponent(keyword));
    const box = document.getElementById('searchResults');
    if (!res || res.code !== 200 || !res.data || res.data.length === 0) {
        box.innerHTML = '<div style="padding:10px; color:#999;">未找到用户</div>';
        return;
    }
    box.innerHTML = res.data.map(u => {
        return '<div class="list-item">' +
            '<div class="avatar">' + escapeHtml(u.nickname.charAt(0).toUpperCase()) + '</div>' +
            '<div class="info"><div class="name">' + escapeHtml(u.nickname) + '</div>' +
            '<div class="last-msg">@' + escapeHtml(u.username) + '</div></div>' +
            '<button class="btn btn-sm btn-primary" data-action="sendRequest" data-id="' + u.id + '">添加</button></div>';
    }).join('');
}

async function sendFriendRequest(userId) {
    const res = await apiPost('/api/friends/request', { to_user_id: userId, message: '请求添加你为好友' });
    if (res && res.code === 200) {
        alert('好友请求已发送');
    } else {
        alert(res ? res.message : '发送失败');
    }
}

async function createGroup() {
    const name = document.getElementById('groupName').value.trim();
    const desc = document.getElementById('groupDesc').value.trim();
    if (!name) { alert('请输入群名称'); return; }
    const res = await apiPost('/api/groups', { name: name, description: desc });
    if (res && res.code === 200) {
        alert('群组创建成功');
        closeModal('groupModal');
        switchTab('groups');
    } else {
        alert(res ? res.message : '创建失败');
    }
}

async function viewGroupMembers(groupId) {
    const res = await apiGet('/api/groups/' + groupId + '/members');
    if (res && res.code === 200 && res.data) {
        const list = res.data.map(m => m.nickname + ' (@' + m.username + ') ' + (m.role === 2 ? '[群主]' : m.role === 1 ? '[管理员]' : '')).join('\n');
        alert('群成员：\n' + list);
    }
}

// Logout
async function logout() {
    await apiPost('/api/logout');
    localStorage.clear();
    location.href = '/static/login.html';
}

// HTML escape utility (XSS defense on frontend)
function escapeHtml(str) {
    if (!str) return '';
    return String(str)
        .replace(/&/g, '&amp;')
        .replace(/</g, '&lt;')
        .replace(/>/g, '&gt;')
        .replace(/"/g, '&quot;')
        .replace(/'/g, '&#39;');
}