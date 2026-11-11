const PRODUCTION_API_URL = "https://edwinslab.onrender.com";
const RAW_BASE_URL =
  import.meta.env.VITE_API_BASE_URL || (import.meta.env.PROD ? PRODUCTION_API_URL : "");
const API_BASE_URL = typeof RAW_BASE_URL === "string" ? RAW_BASE_URL.trim().replace(/\/+$/, "") : "";

const TOKEN_KEY = "edwinslab.session.token";
const DEFAULT_TIMEOUT = 20000;

let memoryToken = null;

function canUseSessionStorage() {
  try {
    return typeof window !== "undefined" && Boolean(window.sessionStorage);
  } catch {
    return false;
  }
}

export const tokenStore = {
  get() {
    if (memoryToken) return memoryToken;
    if (!canUseSessionStorage()) return null;
    try {
      memoryToken = window.sessionStorage.getItem(TOKEN_KEY);
      return memoryToken;
    } catch {
      return null;
    }
  },
  set(token) {
    memoryToken = token;
    if (!canUseSessionStorage()) return;
    try {
      window.sessionStorage.setItem(TOKEN_KEY, token);
    } catch {
      memoryToken = token;
    }
  },
  clear() {
    memoryToken = null;
    if (!canUseSessionStorage()) return;
    try {
      window.sessionStorage.removeItem(TOKEN_KEY);
    } catch {
      memoryToken = null;
    }
  },
};

export class ApiError extends Error {
  constructor(message, { status = 0, data = null, path = "", code = "api_error" } = {}) {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.data = data;
    this.path = path;
    this.code = code;
  }

  get details() {
    if (!this.data) return [];
    if (Array.isArray(this.data.errors)) return this.data.errors;
    if (Array.isArray(this.data.details)) return this.data.details;
    return [];
  }
}

export function isApiConfigured() {
  return API_BASE_URL.length > 0;
}

export function getApiBaseUrl() {
  return API_BASE_URL;
}

function buildUrl(path, query) {
  const base = `${API_BASE_URL}${path.startsWith("/") ? path : `/${path}`}`;
  if (!query) return base;

  const url = new URL(base);
  Object.entries(query).forEach(([key, value]) => {
    if (value === undefined || value === null || value === "") return;
    if (Array.isArray(value)) {
      value.forEach((item) => {
        if (item !== undefined && item !== null && item !== "") {
          url.searchParams.append(`${key}[]`, String(item));
        }
      });
      return;
    }
    url.searchParams.set(key, String(value));
  });
  return url.toString();
}

function extractMessage(data, status) {
  if (data && typeof data === "object") {
    if (typeof data.message === "string" && data.message.trim()) return data.message;
    if (typeof data.error === "string" && data.error.trim()) return data.error;
    if (typeof data.errors === "string" && data.errors.trim()) return data.errors;
    if (Array.isArray(data.errors) && data.errors.length) return data.errors.join(", ");
    if (typeof data.details === "string" && data.details.trim()) return data.details;
  }
  if (status === 401) return "Your session is invalid or has expired.";
  if (status === 403) return "You do not have permission to do that.";
  if (status === 404) return "The requested resource was not found.";
  if (status >= 500) return "The server could not complete this request.";
  return `Request failed${status ? ` (${status})` : ""}.`;
}

function dispatch(name, detail) {
  if (typeof window === "undefined") return;
  window.dispatchEvent(new CustomEvent(name, { detail }));
}

export async function request(path, options = {}) {
  const {
    method = "GET",
    body,
    query,
    auth = true,
    signal,
    timeout = DEFAULT_TIMEOUT,
    headers: extraHeaders,
  } = options;

  if (!API_BASE_URL) {
    throw new ApiError(
      "The app is not connected to the API. Set VITE_API_BASE_URL in your environment.",
      { code: "missing_configuration", path },
    );
  }

  const headers = {
    Accept: "application/json",
    ...extraHeaders,
  };

  const isFormData = typeof FormData !== "undefined" && body instanceof FormData;
  if (body !== undefined && !isFormData) {
    headers["Content-Type"] = "application/json";
  }

  const token = auth ? tokenStore.get() : null;
  if (token) headers.Authorization = `Bearer ${token}`;

  const controller = new AbortController();
  const timeoutId = window.setTimeout(() => controller.abort(), timeout);

  if (signal) {
    if (signal.aborted) controller.abort();
    else signal.addEventListener("abort", () => controller.abort(), { once: true });
  }

  let response;
  try {
    response = await fetch(buildUrl(path, query), {
      method,
      headers,
      body: body === undefined ? undefined : isFormData ? body : JSON.stringify(body),
      signal: controller.signal,
      credentials: "omit",
    });
  } catch {
    if (controller.signal.aborted) {
      throw new ApiError("The request timed out. Check your connection and try again.", {
        code: "timeout",
        path,
      });
    }
    throw new ApiError("Unable to reach the Edwins Lab server. Check your connection and try again.", {
      code: "network_error",
      path,
    });
  } finally {
    window.clearTimeout(timeoutId);
  }

  let data = null;
  const text = await response.text();
  if (text) {
    try {
      data = JSON.parse(text);
    } catch {
      data = { message: text };
    }
  }

  if (!response.ok) {
    if (response.status === 401 && auth) {
      tokenStore.clear();
      dispatch("lab:unauthorized");
    }
    if (response.status === 403 && data?.must_change_password) {
      dispatch("lab:password-change-required");
    }

    throw new ApiError(extractMessage(data, response.status), {
      status: response.status,
      data,
      path,
    });
  }

  return data;
}

export const api = {
  get: (path, options) => request(path, { ...options, method: "GET" }),
  post: (path, body, options) => request(path, { ...options, method: "POST", body }),
  patch: (path, body, options) => request(path, { ...options, method: "PATCH", body }),
  delete: (path, options) => request(path, { ...options, method: "DELETE" }),
};
