import Link from "next/link";
import { Suspense } from "react";
import { UserMenu } from "@/components/user-menu";

export function Header() {
  return (
    <header className="border-b border-line">
      <div className="mx-auto flex h-16 max-w-5xl items-center justify-between px-4">
        <nav className="flex items-center gap-6">
          <Link href="/" className="text-lg font-semibold tracking-tight">
            Forge
          </Link>
          <Link href="/" className="text-sm text-muted hover:text-fg">
            Repositories
          </Link>
          <Link href="/repositories/new" className="text-sm text-muted hover:text-fg">
            New repository
          </Link>
        </nav>
        <Suspense fallback={<div className="h-9 w-20" />}>
          <UserMenu />
        </Suspense>
      </div>
    </header>
  );
}
