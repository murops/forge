import { redirect } from "next/navigation";
import { Suspense } from "react";
import { getTemplates } from "@/lib/api";
import { getSession } from "@/lib/session";
import { NewRepositoryForm } from "./form";

export default function NewRepositoryPage() {
  return (
    <section className="mx-auto max-w-xl space-y-6">
      <div>
        <h1 className="text-2xl font-semibold tracking-tight">New repository</h1>
        <p className="mt-1 text-sm text-muted">
          The repository is created in the background, usually in less than a minute.
        </p>
      </div>
      <Suspense fallback={<p className="text-sm text-muted">Loading templates…</p>}>
        <Form />
      </Suspense>
    </section>
  );
}

async function Form() {
  if (!(await getSession())) {
    redirect("/");
  }
  return <NewRepositoryForm templates={await getTemplates()} />;
}
