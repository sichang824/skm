import { Link } from "react-router-dom";

export function NotFoundPage() {
  return (
    <div className="min-h-screen bg-zinc-950 text-zinc-50">
      <div className="mx-auto flex max-w-3xl flex-col gap-6 px-6 py-16">
        <header className="flex flex-col gap-2">
          <h1 className="text-3xl font-semibold tracking-tight">Not Found</h1>
          <p className="text-zinc-300">
            The page you requested does not exist.
          </p>
        </header>

        <div>
          <Link
            to="/"
            className="inline-flex rounded-xl bg-white px-4 py-2 text-sm font-medium text-zinc-900 hover:bg-zinc-100 active:bg-zinc-200"
          >
            Back to Home
          </Link>
        </div>
      </div>
    </div>
  );
}

