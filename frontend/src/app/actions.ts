"use server";

import { headers } from "next/headers";
import { redirect } from "next/navigation";
import { getToken } from "next-auth/jwt";
import { signIn, signOut } from "@/auth";
import { ApiError, createRepository } from "@/lib/api";
import { endSessionUrl } from "@/lib/keycloak";
import { validateDescription, validateName } from "@/lib/validation";

export async function signInAction() {
  await signIn("keycloak", { redirectTo: "/" });
}

// also ends the Keycloak session, otherwise the next sign in would be silent
export async function signOutAction() {
  const token = await getToken({
    req: { headers: await headers() },
    secret: process.env.AUTH_SECRET!,
    secureCookie: process.env.AUTH_URL?.startsWith("https://"),
  });
  await signOut({ redirect: false });
  redirect(endSessionUrl(token?.idToken, process.env.AUTH_URL ?? "http://localhost:3000"));
}

export type FormState = {
  values: { name: string; description: string; template: string; private: boolean; with_lambda: boolean };
  errors: Partial<Record<"name" | "description" | "template" | "with_lambda" | "form", string>>;
};

export async function createRepositoryAction(_: FormState, form: FormData): Promise<FormState> {
  const values = {
    name: String(form.get("name") ?? "").trim(),
    description: String(form.get("description") ?? "").trim(),
    template: String(form.get("template") ?? ""),
    private: form.get("private") === "on",
    with_lambda: form.get("with_lambda") === "on",
  };

  const errors: FormState["errors"] = {
    name: validateName(values.name),
    description: validateDescription(values.description),
    template: values.template ? undefined : "Pick a template.",
  };
  if (Object.values(errors).some(Boolean)) {
    return { values, errors };
  }

  let id: string;
  try {
    id = (await createRepository(values)).id;
  } catch (err) {
    if (err instanceof ApiError && (err.status === 409 || err.status === 422)) {
      const field = err.field && err.field in values ? err.field : "form";
      return { values, errors: { [field]: err.message } };
    }
    throw err;
  }
  redirect(`/repositories/${id}`);
}
