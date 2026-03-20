"use client";

import { useEffect, useSyncExternalStore } from "react";

type StoredAuth = {
	token: string | null;
	user?: {
		name?: string | null;
		email?: string | null;
		username?: string | null;
		role?: string | null;
	} | null;
};

const STORAGE_KEY = "jj_auth_state";

let authState: StoredAuth = { token: null, user: null };
const listeners = new Set<() => void>();
let cachedSnapshot: StoredAuth | null = null;

let expiryTimer: ReturnType<typeof setTimeout> | null = null;

const decodeJwtPayload = (token: string): Record<string, unknown> | null => {
	const parts = token.split(".");
	if (parts.length < 2) return null;
	const payload = parts[1];
	if (!payload) return null;
	try {
		const normalized = payload.replace(/-/g, "+").replace(/_/g, "/");
		const padded = normalized.padEnd(Math.ceil(normalized.length / 4) * 4, "=");
		const decoded = atob(padded);
		const parsed = JSON.parse(decoded);
		if (parsed && typeof parsed === "object") return parsed as Record<string, unknown>;
		return null;
	} catch {
		return null;
	}
};

const getTokenExpiryMs = (token: string): number | null => {
	const payload = decodeJwtPayload(token);
	if (!payload) return null;
	const rawExp = payload.exp;
	if (typeof rawExp === "number" && Number.isFinite(rawExp)) {
		return rawExp > 1_000_000_000_000 ? rawExp : rawExp * 1000;
	}
	if (typeof rawExp === "string") {
		const numeric = Number(rawExp);
		if (!Number.isFinite(numeric)) return null;
		return numeric > 1_000_000_000_000 ? numeric : numeric * 1000;
	}
	return null;
};

const isTokenExpired = (token: string, now = Date.now()): boolean => {
	const expiry = getTokenExpiryMs(token);
	if (!expiry) return false;
	return now >= expiry;
};

const normalizeAuth = (value: StoredAuth): StoredAuth => {
	const token = typeof value.token === "string" ? value.token : null;
	const user = value.user ?? null;
	if (token && isTokenExpired(token)) {
		return { token: null, user: null };
	}
	return { token, user };
};

const scheduleExpiryCheck = (token: string | null) => {
	if (expiryTimer) {
		clearTimeout(expiryTimer);
		expiryTimer = null;
	}
	if (!token) return;
	const expiry = getTokenExpiryMs(token);
	if (!expiry) return;
	const delay = Math.max(expiry - Date.now(), 0);
	expiryTimer = setTimeout(() => {
		clearAuth();
	}, delay);
};

const readFromStorage = (): StoredAuth => {
	if (typeof window === "undefined") return { token: null, user: null };
	const raw = window.localStorage.getItem(STORAGE_KEY);
	if (!raw) return { token: null, user: null };
	try {
		const parsed = JSON.parse(raw);
		const normalized = normalizeAuth({
			token: typeof parsed.token === "string" ? parsed.token : null,
			user: parsed.user ?? null,
		});
		if (!normalized.token && (parsed?.token || parsed?.user)) {
			writeToStorage(normalized);
		}
		return normalized;
	} catch {
		return { token: null, user: null };
	}
};

const writeToStorage = (value: StoredAuth) => {
	if (typeof window === "undefined") return;
	window.localStorage.setItem(STORAGE_KEY, JSON.stringify(value));
};

const notify = () => {
	for (const cb of listeners) cb();
};

export const setAuth = (value: StoredAuth) => {
	const normalized = normalizeAuth(value);
	authState = normalized;
	cachedSnapshot = normalized;
	writeToStorage(normalized);
	scheduleExpiryCheck(normalized.token);
	notify();
};

export const clearAuth = () => {
	setAuth({ token: null, user: null });
};

export const getAuthSnapshot = (): StoredAuth => {
	if (typeof window === "undefined") return authState;
	if (cachedSnapshot !== null) return cachedSnapshot;

	const snapshot = readFromStorage();
	authState = snapshot;
	cachedSnapshot = snapshot;

	return snapshot;
};

export const subscribeAuth = (cb: () => void) => {
	listeners.add(cb);
	return () => listeners.delete(cb);
};

export function useAuth() {
	const snapshot = useSyncExternalStore(subscribeAuth, getAuthSnapshot, () => authState);

	useEffect(() => {
		scheduleExpiryCheck(snapshot.token);
	}, [snapshot.token]);

	return snapshot;
}
