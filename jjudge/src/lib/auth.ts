"use client";

import { useSyncExternalStore } from "react";

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

const readFromStorage = (): StoredAuth => {
	if (typeof window === "undefined") return { token: null, user: null };
	const raw = window.localStorage.getItem(STORAGE_KEY);
	if (!raw) return { token: null, user: null };
	try {
		const parsed = JSON.parse(raw);
		return {
			token: typeof parsed.token === "string" ? parsed.token : null,
			user: parsed.user ?? null,
		};
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
	authState = value;
	cachedSnapshot = value;
	writeToStorage(value);
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
	return useSyncExternalStore(subscribeAuth, getAuthSnapshot, () => authState);
}
