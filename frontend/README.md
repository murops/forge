# Forge frontend

Next.js app where developers sign in with SSO, pick a template and follow the repository creation.

- Auth.js with the Keycloak provider. The Keycloak tokens stay in the encrypted session cookie and are only read on the server, the browser never sees them.
- Every call to the Forge API happens on the server (Server Components and Server Actions) with the user's access token, refreshed when it expires.
- Pages poll with `router.refresh()` while a request is still queued or provisioning.

## Running

The easiest way is the compose stack in the repository root, which also starts Keycloak and the API. To work on the UI alone:

```bash
npm ci
cp .env.example .env.local
npm run dev
```

`npm test`, `npm run lint` and `npm run typecheck` run the unit tests, ESLint and the TypeScript checks.
