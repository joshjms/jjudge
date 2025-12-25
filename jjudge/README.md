# JJudge

## Backend API setup
- Copy `.env.local.example` to `.env.local` and adjust `NEXT_PUBLIC_API_BASE_URL` if your backend URL changes (defaults to `http://localhost:8080`).
- `src/lib/api.ts` exposes a tiny wrapper around `fetch` that automatically targets the configured base URL, handles JSON, and raises an `ApiError` on non-OK responses.

### Example usage
```ts
import { api } from "@/lib/api";

type Problem = { id: string; title: string };

async function loadProblems() {
	const data = await api.get<Problem[]>("/problems");
	return data;
}

async function createProblem(payload: { title: string }) {
	return api.post<Problem>("/problems", payload);
}
```
