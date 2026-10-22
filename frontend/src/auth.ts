import NextAuth from "next-auth";
import type { JWT } from "next-auth/jwt";
import Keycloak from "next-auth/providers/keycloak";
import { refreshTokens } from "@/lib/keycloak";

declare module "next-auth" {
  interface Session {
    error?: "RefreshTokenError";
  }
}

declare module "next-auth/jwt" {
  interface JWT {
    accessToken?: string;
    refreshToken?: string;
    expiresAt?: number;
    idToken?: string;
    error?: "RefreshTokenError";
  }
}

// renew a bit before Keycloak considers the token expired
const EXPIRY_MARGIN_SECONDS = 30;

export function isExpired(token: JWT) {
  return !token.expiresAt || Date.now() / 1000 > token.expiresAt - EXPIRY_MARGIN_SECONDS;
}

export const { handlers, auth, signIn, signOut } = NextAuth({
  providers: [Keycloak],
  session: { strategy: "jwt" },
  callbacks: {
    async jwt({ token, account }) {
      if (account) {
        return {
          ...token,
          accessToken: account.access_token,
          refreshToken: account.refresh_token,
          expiresAt: account.expires_at,
          idToken: account.id_token,
        };
      }
      if (!isExpired(token) || !token.refreshToken) {
        return token;
      }
      try {
        return { ...token, ...(await refreshTokens(token.refreshToken)), error: undefined };
      } catch {
        return { ...token, error: "RefreshTokenError" };
      }
    },
    // tokens stay in the encrypted cookie, the browser only gets name and email
    session({ session, token }) {
      return { ...session, error: token.error };
    },
  },
});
