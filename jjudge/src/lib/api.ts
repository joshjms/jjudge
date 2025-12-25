type QueryValue = string | number | boolean | null | undefined;

type ApiRequestOptions = {
	method?: "GET" | "POST" | "PUT" | "PATCH" | "DELETE";
	query?: Record<string, QueryValue>;
	body?: unknown;
	headers?: HeadersInit;
	signal?: AbortSignal;
	cache?: RequestCache;
};

export class ApiError extends Error {
	status: number;
	data: unknown;

	constructor(status: number, message: string, data: unknown) {
		super(message);
		this.name = "ApiError";
		this.status = status;
		this.data = data;
	}
}

const API_BASE_URL =
	(process.env.NEXT_PUBLIC_API_BASE_URL || "http://localhost:8080").replace(/\/$/, "");

const buildUrl = (path: string, query?: Record<string, QueryValue>) => {
	const normalizedPath = path.startsWith("/") ? path.slice(1) : path;
	const url = new URL(normalizedPath, `${API_BASE_URL}/`);

	if (query) {
		Object.entries(query).forEach(([key, value]) => {
			if (value === undefined || value === null) return;
			url.searchParams.append(key, String(value));
		});
	}

	return url.toString();
};

const serializeBody = (body: unknown) => {
	if (body === undefined || body === null) return undefined;
	if (typeof FormData !== "undefined" && body instanceof FormData) return body;
	return JSON.stringify(body);
};

async function request<TResponse>(
	path: string,
	{ method = "GET", query, body, headers, signal, cache = "no-store" }: ApiRequestOptions = {},
) {
	const serializedBody = serializeBody(body);
	const isFormData =
		typeof FormData !== "undefined" && serializedBody instanceof FormData && body !== undefined;

	const response = await fetch(buildUrl(path, query), {
		method,
		body: serializedBody,
		headers: {
			Accept: "application/json",
			...(!isFormData && serializedBody ? { "Content-Type": "application/json" } : {}),
			...headers,
		},
		signal,
		cache,
	});

	const contentType = response.headers.get("content-type");
	const isJson = contentType?.includes("application/json");
	const payload = isJson ? await response.json().catch(() => null) : await response.text();

	if (!response.ok) {
		throw new ApiError(response.status, "Request to backend failed", payload);
	}

	return payload as TResponse;
}

export const api = {
	get: <TResponse>(path: string, options?: Omit<ApiRequestOptions, "method" | "body">) =>
		request<TResponse>(path, { ...options, method: "GET" }),
	post: <TResponse>(
		path: string,
		body?: ApiRequestOptions["body"],
		options?: Omit<ApiRequestOptions, "method" | "body">,
	) => request<TResponse>(path, { ...options, method: "POST", body }),
	put: <TResponse>(
		path: string,
		body?: ApiRequestOptions["body"],
		options?: Omit<ApiRequestOptions, "method" | "body">,
	) => request<TResponse>(path, { ...options, method: "PUT", body }),
	patch: <TResponse>(
		path: string,
		body?: ApiRequestOptions["body"],
		options?: Omit<ApiRequestOptions, "method" | "body">,
	) => request<TResponse>(path, { ...options, method: "PATCH", body }),
	delete: <TResponse>(path: string, options?: Omit<ApiRequestOptions, "method">) =>
		request<TResponse>(path, { ...options, method: "DELETE" }),
};

export const getApiBaseUrl = () => API_BASE_URL;
