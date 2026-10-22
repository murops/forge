import Link from "next/link";
import { Suspense } from "react";
import { AutoRefresh } from "@/components/auto-refresh";
import { LocalTime } from "@/components/local-time";
import { StatusBadge } from "@/components/status-badge";
import { signInAction } from "@/app/actions";
import { getRepositories } from "@/lib/api";
import { getSession } from "@/lib/session";
import { isFinished } from "@/lib/status";

export default function Home() {
  return (
    <Suspense fallback={<p className="text-sm text-muted">Loading…</p>}>
      <Dashboard />
    </Suspense>
  );
}

async function Dashboard() {
  if (!(await getSession())) {
    return <Landing />;
  }

  const repositories = await getRepositories();
  return (
    <section className="space-y-6">
      <div className="flex items-end justify-between gap-4">
        <div>
          <h1 className="text-2xl font-semibold tracking-tight">Your repositories</h1>
          <p className="mt-1 text-sm text-muted">Repositories you requested through Forge.</p>
        </div>
        <Link href="/repositories/new" className="btn-primary">
          New repository
        </Link>
      </div>

      {repositories.length === 0 ? (
        <div className="card p-10 text-center text-sm text-muted">
          Nothing here yet. Create your first repository from a template.
        </div>
      ) : (
        <ul className="card divide-y divide-line">
          {repositories.map((repo) => (
            <li key={repo.id} className="flex items-center justify-between gap-4 px-5 py-4">
              <div className="min-w-0">
                <Link href={`/repositories/${repo.id}`} className="font-medium hover:underline">
                  {repo.name}
                </Link>
                <p className="mt-0.5 truncate text-sm text-muted">
                  {repo.template} · <LocalTime iso={repo.created_at} />
                </p>
              </div>
              <StatusBadge status={repo.status} />
            </li>
          ))}
        </ul>
      )}
      {repositories.some((r) => !isFinished(r.status)) && <AutoRefresh />}
    </section>
  );
}

function Landing() {
  return (
    <section className="mx-auto max-w-2xl py-16 text-center">
      <h1 className="text-4xl font-semibold tracking-tight">New repositories, ready to ship</h1>
      <p className="mt-4 text-muted">
        Pick a template and Forge creates the GitHub repository with CI, security scans, SonarCloud and
        protected <code>main</code>, <code>homol</code> and <code>desen</code> branches.
      </p>
      <form action={signInAction} className="mt-8">
        <button className="btn-primary">Sign in with SSO</button>
      </form>
    </section>
  );
}
