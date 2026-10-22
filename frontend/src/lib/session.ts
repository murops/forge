import "server-only";
import { connection } from "next/server";
import { auth } from "@/auth";

// auth() re-encrypts the cookie with a random IV, which Next does not allow while prerendering
export async function getSession() {
  await connection();
  const session = await auth();
  return session?.user && !session.error ? session : null;
}
