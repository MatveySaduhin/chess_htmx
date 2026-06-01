const EASY_AUTH_ACCESS_TOKEN_KEY = "easy_auth_access_token";
const EASY_AUTH_REFRESH_TOKEN_KEY = "easy_auth_refresh_token";

function getAccessToken() {
    return localStorage.getItem(EASY_AUTH_ACCESS_TOKEN_KEY);
}

function authFetch(url, options = {}) {
    const accessToken = getAccessToken();
    const headers = new Headers(options.headers || {});
    if (accessToken) {
        headers.set("Authorization", `Bearer ${accessToken}`);
    }

    return fetch(url, {
        ...options,
        headers
    });
}

async function loadCurrentProfile() {
    if (!getAccessToken()) {
        return null;
    }

    const response = await authFetch("/api/profile/me");
    if (response.status === 401) {
        logoutLocal();
        return null;
    }
    if (!response.ok) {
        throw new Error("Failed to load chess profile");
    }

    const data = await response.json();
    return data.profile || null;
}

function storeEasyAuthTokens(tokenResponse) {
    localStorage.setItem(EASY_AUTH_ACCESS_TOKEN_KEY, tokenResponse.access_token);
    if (tokenResponse.refresh_token) {
        localStorage.setItem(EASY_AUTH_REFRESH_TOKEN_KEY, tokenResponse.refresh_token);
    }
}

function logoutLocal() {
    localStorage.removeItem(EASY_AUTH_ACCESS_TOKEN_KEY);
    localStorage.removeItem(EASY_AUTH_REFRESH_TOKEN_KEY);
}

function easyAuthErrorMessage(data, fallback) {
    return data.message || data.error || fallback;
}
