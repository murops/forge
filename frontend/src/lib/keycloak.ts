import "server-only";

export type Tokens = {
  accessToken: string;
  refreshToken?: string;
  expiresAt: number;
  idToken?: string;
};

const issuer = () => process.env.AUTH_KEYCLOAK_ISSUER!;

export async function refreshTokens(refreshToken: string): Promise<Tokens> {
  const res = await fetch(`${issuer()}/protocol/openid-connect/token`, {
    method: "POST",
    headers: { "Content-Type": "application/x-www-form-urlencoded" },
    body: new URLSearchParams({
      grant_type: "refresh_token",
      refresh_token: refreshToken,
      client_id: process.env.AUTH_KEYCLOAK_ID!,
      client_secret: process.env.AUTH_KEYCLOAK_SECRET!,
    }),
    cache: "no-store",
  });
  if (!res.ok) {
    throw new Error(`refresh failed: ${res.status}`);
  }
  const body = await res.json();
  return {
    accessToken: body.access_token,
    refreshToken: body.refresh_token ?? refreshToken,
    expiresAt: Math.floor(Date.now() / 1000) + body.expires_in,
    idToken: body.id_token,
  };
}

export function endSessionUrl(idToken: string | undefined, redirectTo: string) {
  const url = new URL(`${issuer()}/protocol/openid-connect/logout`);
  url.searchParams.set("post_logout_redirect_uri", redirectTo);
  url.searchParams.set("client_id", process.env.AUTH_KEYCLOAK_ID!);
  if (idToken) {
    url.searchParams.set("id_token_hint", idToken);
  }
  return url.toString();
}
