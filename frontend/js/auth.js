// Auth utilities shared across pages
const API_BASE = '';

function getToken() {
    return localStorage.getItem('token');
}

function checkAuth() {
    const token = getToken();
    if (!token) {
        location.href = '/static/login.html';
        return false;
    }
    return true;
}

function authHeaders() {
    return {
        'Content-Type': 'application/json',
        'Authorization': 'Bearer ' + getToken()
    };
}

async function apiGet(url) {
    const res = await fetch(url, { headers: { 'Authorization': 'Bearer ' + getToken() } });
    return handleResponse(res);
}

async function apiPost(url, body) {
    const res = await fetch(url, {
        method: 'POST',
        headers: authHeaders(),
        body: JSON.stringify(body || {})
    });
    return handleResponse(res);
}

async function apiPut(url, body) {
    const res = await fetch(url, {
        method: 'PUT',
        headers: authHeaders(),
        body: JSON.stringify(body || {})
    });
    return handleResponse(res);
}

async function apiDelete(url) {
    const res = await fetch(url, {
        method: 'DELETE',
        headers: { 'Authorization': 'Bearer ' + getToken() }
    });
    return handleResponse(res);
}

async function handleResponse(res) {
    const data = await res.json();
    if (data.code === 401) {
        localStorage.clear();
        location.href = '/static/login.html';
        return null;
    }
    return data;
}