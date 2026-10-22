import "server-only";
import { headers } from "next/headers";
import { notFound, redirect } from "next/navigation";
import { getToken } from "next-auth/jwt";
import { isExpired } from "@/auth";
import { refreshTokens } from "@/lib/keycloak";
import type { NewRepository, Repository, Template } from "@/lib/types";

export class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
    public field?: string,
  ) {
    super(message);
  }
}

// read straight from the session cookie, the token is never sent to the browser
async function accessToken(): Promise<string> {
  const token = await getToken({
    req: { headers: await headers() },
    secret: process.env.AUTH_SECRET!,
    secureCookie: process.env.AUTH_URL?.startsWith("https://"),
  });
  if (!token?.accessToken) {
    redirect("/");
  }
  if (!isExpired(token)) {
    return token.accessToken;
  }

  let refreshed: string | undefined;
  if (token.refreshToken) {
    try {
      refreshed = (await refreshTokens(token.refreshToken)).accessToken;
    } catch {
      refreshed = undefined;
    }
  }
  if (!refreshed) {
    redirect("/");
  }
  return refreshed;
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(`${process.env.FORGE_API_URL}/api/v1${path}`, {
    ...init,
    headers: {
      Authorization: `Bearer ${await accessToken()}`,
      "Content-Type": "application/json",
    },
    cache: "no-store",
  });

  if (res.status === 401) {
    redirect("/");
  }
  if (!res.ok) {
    const body = await res.json().catch(() => ({}));
    throw new ApiError(res.status, body.error ?? res.statusText, body.field);
  }
  return res.json();
}

export async function getTemplates() {
  const body = await request<{ templates: Template[] }>("/templates");
  return body.templates;
}

export async function getRepositories() {
  const body = await request<{ repositories: Repository[] | null }>("/repositories");
  return body.repositories ?? [];
}

export async function getRepository(id: string) {
  try {
    return await request<Repository>(`/repositories/${encodeURIComponent(id)}`);
  } catch (err) {
    if (err instanceof ApiError && err.status === 404) {
      notFound();
    }
    throw err;
  }
}

export async function createRepository(input: NewRepository) {
  return request<Repository>("/repositories", {
    method: "POST",
    body: JSON.stringify(input),
  });
}
