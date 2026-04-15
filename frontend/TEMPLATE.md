# Frontend Template (React + TailwindCSS v4 + Vite)

This folder is intended to be **copied** and used as a starter template for new frontend projects.

## What you get

- React (latest)
- Vite (latest)
- TypeScript
- TailwindCSS v4 (via `@tailwindcss/vite`)
- URL routing via React Router (`react-router-dom`)
- Vitest + Testing Library (jsdom)
- A simple `Makefile` wrapper for common commands

## Use this template (copy workflow)

1. Copy the whole folder.

```bash
cp -R blog/frontend /path/to/new-project
cd /path/to/new-project
```

1. Remove local artifacts (if you copied from a working directory).

```bash
rm -rf node_modules dist
```

1. Update project metadata.

- Edit `package.json`:
  - `name`
  - `version`
  - (optional) `private`
- Search/replace the text in `src/App.tsx` as needed.

1. Install and run.

```bash
make install
make dev
```

## Day-to-day commands

```bash
make           # list targets
make install   # npm install
make dev       # vite dev server
make test      # vitest (run mode)
make build     # production build
make lint      # eslint
make preview   # preview build
make clean     # remove node_modules/dist
```

## Project structure (key files)

- `vite.config.ts`
  - Enables Tailwind v4 via `@tailwindcss/vite`
  - Configures Vitest (`test.environment = "jsdom"`, `setupFiles = "./vitest.setup.ts"`)
- `src/index.css`
  - Tailwind entrypoint: `@import "tailwindcss";`
- `src/vite-env.d.ts`
  - Vite client types + `@testing-library/jest-dom` types
- `vitest.setup.ts`
  - Testing Library DOM matchers: `@testing-library/jest-dom/vitest`
- `src/App.spec.tsx`
  - Routing tests (`/`, `/about`, and Not Found fallback)
- `src/main.tsx`
  - Wraps the app with `BrowserRouter`
- `src/App.tsx`
  - Route table using `Routes` / `Route`
- `src/pages/*`
  - Example pages (Home/About/NotFound)

## SPA routing in production (important)

When using `BrowserRouter`, your production server must serve `index.html` for unknown routes
(e.g. `/about`) so the client-side router can take over.

Vite dev/preview already handles this for you. For custom deployments, configure a fallback:

- Nginx: try_files → `/index.html`
- Netlify: `_redirects` → `/* /index.html 200`
- Vercel: rewrite all routes to `/`

## How this template was created (from scratch)

If you want to recreate it from zero (or upgrade the template), these are the commands used:

1. Scaffold with Vite (React + TS).

```bash
npm create vite@latest frontend -- --template react-ts
cd frontend
npm install
```

1. Add Tailwind v4 for Vite.

```bash
npm install -D tailwindcss@latest @tailwindcss/vite@latest
```

Then:

- Add `tailwindcss()` to `plugins` in `vite.config.ts`
- Replace `src/index.css` with:

```css
@import "tailwindcss";
```

1. Add tests (Vitest + Testing Library).

```bash
npm install -D vitest@latest jsdom@latest \
  @testing-library/react@latest @testing-library/jest-dom@latest @testing-library/user-event@latest
```

Then:

- Add `test` scripts in `package.json`
- Add `test` config in `vite.config.ts` (jsdom + `setupFiles`)
- Create `vitest.setup.ts` importing `@testing-library/jest-dom/vitest`
- Add `src/vite-env.d.ts` to include matcher types

## Keeping dependencies up to date

- Update within compatible ranges:

```bash
npm update
```

- Bump to latest major versions (review changes carefully):

```bash
npx npm-check-updates -u
npm install
```

## Notes

- This template intentionally uses Tailwind v4's Vite plugin approach. It does **not** rely on the older `tailwindcss init` workflow.
