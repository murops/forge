"use client";

import { useActionState, useState } from "react";
import { createRepositoryAction, type FormState } from "@/app/actions";
import type { Template } from "@/lib/types";

export function NewRepositoryForm({ templates }: { templates: Template[] }) {
  const initial: FormState = {
    values: { name: "", description: "", template: templates[0]?.slug ?? "", private: false, with_lambda: false },
    errors: {},
  };
  const [state, action, pending] = useActionState(createRepositoryAction, initial);
  const [template, setTemplate] = useState(state.values.template);
  const selected = templates.find((t) => t.slug === template);

  return (
    <form action={action} className="card space-y-5 p-6">
      <Field label="Name" error={state.errors.name}>
        <input
          name="name"
          defaultValue={state.values.name}
          required
          placeholder="payments-api"
          autoComplete="off"
          className="input font-mono"
        />
      </Field>

      <Field label="Description" error={state.errors.description}>
        <input name="description" defaultValue={state.values.description} maxLength={350} className="input" />
      </Field>

      <fieldset className="space-y-2">
        <legend className="text-sm font-medium">Template</legend>
        {templates.map((t) => (
          <label
            key={t.slug}
            className="flex cursor-pointer gap-3 rounded-md border border-line p-3 has-checked:border-fg"
          >
            <input
              type="radio"
              name="template"
              value={t.slug}
              checked={template === t.slug}
              onChange={() => setTemplate(t.slug)}
              className="mt-1"
            />
            <span>
              <span className="block text-sm font-medium">{t.name}</span>
              <span className="block text-sm text-muted">{t.description}</span>
            </span>
          </label>
        ))}
        {state.errors.template && <p className="text-sm text-red-600 first-letter:uppercase">{state.errors.template}</p>}
      </fieldset>

      <div className="space-y-2">
        <label className="flex items-center gap-2 text-sm">
          <input type="checkbox" name="private" defaultChecked={state.values.private} />
          Private repository
        </label>
        {selected?.supports_lambda && (
          <label className="flex items-center gap-2 text-sm">
            <input type="checkbox" name="with_lambda" defaultChecked={state.values.with_lambda} />
            Create the AWS Lambda function and ECR repository
          </label>
        )}
        {state.errors.with_lambda && <p className="text-sm text-red-600 first-letter:uppercase">{state.errors.with_lambda}</p>}
      </div>

      {state.errors.form && <p className="rounded-md bg-red-500/10 p-3 text-sm text-red-600 first-letter:uppercase">{state.errors.form}</p>}

      <button className="btn-primary w-full justify-center" disabled={pending}>
        {pending ? "Creating…" : "Create repository"}
      </button>
    </form>
  );
}

function Field({ label, error, children }: { label: string; error?: string; children: React.ReactNode }) {
  return (
    <label className="block space-y-1.5">
      <span className="text-sm font-medium">{label}</span>
      {children}
      {error && <span className="block text-sm text-red-600 first-letter:uppercase">{error}</span>}
    </label>
  );
}
