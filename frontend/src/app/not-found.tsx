import Link from "next/link";

export default function NotFound() {
  return (
    <section className="py-16 text-center">
      <h1 className="text-2xl font-semibold">Not found</h1>
      <p className="mt-2 text-sm text-muted">This repository does not exist or was requested by someone else.</p>
      <Link href="/" className="btn-secondary mt-6">
        Back to repositories
      </Link>
    </section>
  );
}
