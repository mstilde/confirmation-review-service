export function getToken(): string | null {
  if (typeof window === "undefined") return null;
  return localStorage.getItem("cr_token");
}

export function setToken(token: string) {
  localStorage.setItem("cr_token", token);
}

export function clearToken() {
  localStorage.removeItem("cr_token");
}

export function isAuthenticated(): boolean {
  return true;
}

export async function login(email: string, password: string) {
  const res = await fetch("/api/auth/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ email, password }),
  });
  const data = await res.json();
  if (!res.ok) throw new Error(data.error || "Error de autenticación");
  setToken(data.token);
  return data;
}

export async function api(path: string, options: RequestInit = {}) {
  const headers: Record<string, string> = {
    "Content-Type": "application/json",
    ...(options.headers as Record<string, string>),
  };

  const res = await fetch(path, { ...options, headers });
  const data = await res.json().catch(() => ({}));
  if (!res.ok) {
    throw new Error(data.error || `HTTP ${res.status}`);
  }
  return data;
}
