import Link from "next/link";
import { Suspense } from "react";
import { AutoRefresh } from "@/components/auto-refresh";
import { LocalTime } from "@/components/local-time";
import { StatusBadge } from "@/components/status-badge";
import { getRepository } from "@/lib/api";
import { isFinished } from "@/lib/status";

export default function RepositoryPage({ params }: PageProps<"/repositories/[id]">) {
  return (
    <section className="mx-auto max-w-2xl space-y-6">
      <Link href="/" className="text-sm text-muted hover:text-fg">
        ← Repositories
      </Link>
      <Suspense fallback={<p className="text-sm text-muted">Loading…</p>}>
        <Details params={params} />
      </Suspense>
    </section>
  );
}

async function Details({ params }: { params: Promise<{ id: string }> }) {
  const { id } = await params;
  const repo = await getRepository(id);

  return (
    <div className="card divide-y divide-line">
      <div className="flex items-start justify-between gap-4 p-6">
        <div>
          <h1 className="font-mono text-xl font-semibold">{repo.name}</h1>
          {repo.description && <p className="mt-1 text-sm text-muted">{repo.description}</p>}
        </div>
        <StatusBadge status={repo.status} />
      </div>

      <dl className="grid grid-cols-2 gap-x-6 gap-y-4 p-6 text-sm">
        <Item label="Template">{repo.template}</Item>
        <Item label="Visibility">{repo.private ? "Private" : "Public"}</Item>
        <Item label="AWS Lambda">{repo.with_lambda ? "Requested" : "No"}</Item>
        <Item label="Requested">
          <LocalTime iso={repo.created_at} />
        </Item>
      </dl>

      {repo.status === "succeeded" && repo.repo_url && (
        <div className="p-6">
          <a href={repo.repo_url} target="_blank" rel="noreferrer" className="btn-primary">
            Open on GitHub
          </a>
        </div>
      )}
      {repo.status === "failed" && repo.error && (
        <div className="p-6">
          <p className="mb-2 text-sm font-medium text-red-600">Provisioning failed after the retries</p>
          <pre className="overflow-x-auto rounded-md bg-red-500/10 p-3 text-xs whitespace-pre-wrap">{repo.error}</pre>
        </div>
      )}
      {!isFinished(repo.status) && (
        <p className="p-6 text-sm text-muted">This page updates automatically.</p>
      )}
      {!isFinished(repo.status) && <AutoRefresh />}
    </div>
  );
}

function Item({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div>
      <dt className="text-muted">{label}</dt>
      <dd className="mt-0.5 font-medium">{children}</dd>
    </div>
  );
}
