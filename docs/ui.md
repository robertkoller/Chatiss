# UI & Desktop App

## Stack

| Layer | Technology |
|---|---|
| Desktop shell | [Wails v2](https://wails.io) |
| UI framework | React 18 + TypeScript |
| Build tool | Vite |
| Go backend | `app/` package |

## What Wails Does

Wails compiles your Go code and React app into a single native `.app` binary. On macOS it uses **WKWebView** (the same engine as Safari) to render the React UI — there is no bundled browser. The Go code runs natively as the backend process.

React calls Go functions via a generated JavaScript bridge (`window.go`). Go pushes events to React via `runtime.EventsEmit`.

## wails.json

`wails.json` is the project config file for the Wails CLI. **Do not delete it and do not add it to `.gitignore`** — it needs to be committed alongside the code. It tells Wails how to install and build the frontend.

```json
{
  "frontend:install": "npm install",
  "frontend:build": "npm run build",
  "frontend:dev:watcher": "npm run dev",
  "frontend:dev:serverUrl": "auto"
}
```

## Running in Development

```bash
wails dev
```

This compiles Go, starts the Wails dev server, and opens the app window. Changes to `.go` files trigger a Go recompile. Changes to `.tsx`/`.ts`/`.css` files hot-reload via Vite without a full recompile.

## Building for Production

```bash
wails build
```

Produces `build/bin/Chatiss.app` — a self-contained macOS application.

## Go ↔ React Bridge

**Go methods exposed to React** (in `app/app.go`):

| Method | Returns | Description |
|---|---|---|
| `Login(passphrase, username)` | `string` (error or `""`) | Start the background service |
| `GetContacts()` | `[]UIContact` | List contacts with online status |
| `AddContact(username)` | `string` (error or `""`) | Look up + save a new contact |
| `GetMessages(username)` | `[]UIMessage` | Message history for a contact |
| `SendMessage(username, text)` | `string` (error or `""`) | Send (live or via mailbox) |

**Events emitted from Go** (received in React via `EventsOn`):

| Event | Payload | Description |
|---|---|---|
| `message:received` | `UIMessage` | New incoming message |
| `contact:online` | `{username: string}` | Peer connected |
| `contact:offline` | `{username: string}` | Peer disconnected |

## Frontend Structure

```
frontend/src/
├── App.tsx                     # Root: login gate, contact state, event wiring
├── main.tsx                    # React DOM mount
├── index.css                   # Global CSS variables + resets
├── components/
│   ├── LoginScreen.tsx         # Passphrase + username form
│   ├── Sidebar.tsx             # Contact list + add-contact form
│   └── MessageThread.tsx       # Chat bubbles + send input
└── wailsjs/
    ├── runtime/runtime.ts      # Wails event helpers (EventsOn, EventsOff)
    └── go/app/App.ts           # Typed wrappers around window.go calls
```
