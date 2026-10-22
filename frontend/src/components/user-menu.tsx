import { signInAction, signOutAction } from "@/app/actions";
import { getSession } from "@/lib/session";

export async function UserMenu() {
  const session = await getSession();

  if (!session) {
    return (
      <form action={signInAction}>
        <button className="btn-primary">Sign in</button>
      </form>
    );
  }

  return (
    <div className="flex items-center gap-4">
      <span className="hidden text-sm text-muted sm:inline">{session.user?.email ?? session.user?.name}</span>
      <form action={signOutAction}>
        <button className="btn-secondary">Sign out</button>
      </form>
    </div>
  );
}
