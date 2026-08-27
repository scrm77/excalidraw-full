# Modernization of draw.meatbags.ru

## Outcome

Keep the private multi-canvas product and owner API while replacing the old editor core with the current upstream Excalidraw editor.

## Compatibility boundary

- Keep the Go server, SQLite schema, GitHub allowlist, JWT authentication, and `/api/v2/kv` owner API.
- Keep editable canvas JSON compatible with the existing database.
- Keep two explicit storage modes only: this browser (IndexedDB) and the private server.
- Do not configure or expose the OpenAI proxy as part of this change.
- Preserve the current production image and database backup as the rollback boundary.

## Implementation

1. Merge current `excalidraw/excalidraw` into the custom `multi-canvas` frontend.
2. Keep upstream editor packages unchanged and port the canvas catalog into the upstream sidebar.
3. Remove unused browser-side Cloudflare KV and Amazon S3 adapters.
4. Use the current Mermaid converter and Mermaid runtime, including the upstream XSS fix.
5. Build and test the frontend and Go backend before production deployment.
6. Back up and integrity-check SQLite before switching the production container.

## Update policy

The monthly GitHub Actions check compares the pinned editor with official Excalidraw and opens or refreshes one review issue when updates exist. Updates are reviewed and tested; they are never deployed automatically.

## Acceptance checks

- The app loads and the editor can create, edit, export, and reopen a scene.
- GitHub login remains restricted by the configured allowlist.
- Browser-only canvases survive a reload in the same browser.
- Server canvases list, save, reopen, rename, and delete.
- The owner API can list, read, and update a test canvas with readback verification.
- Mermaid import accepts a normal diagram and rejects unsafe content.
- The old image and the pre-deploy SQLite copy are sufficient for rollback.
