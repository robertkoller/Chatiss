// Wails runtime — injected by the desktop shell at startup.
// In dev mode these come from the Wails dev server; in prod they're embedded.

interface WailsRuntime {
  EventsOn(eventName: string, callback: (...data: unknown[]) => void): () => void
  EventsOff(eventName: string, ...additionalEventNames: string[]): void
  EventsEmit(eventName: string, ...data: unknown[]): void
  WindowMinimise(): void
  WindowMaximise(): void
  WindowClose(): void
  Quit(): void
}

declare global {
  interface Window {
    runtime: WailsRuntime
    go: Record<string, Record<string, Record<string, (...args: unknown[]) => Promise<unknown>>>>
  }
}

export const EventsOn = (eventName: string, callback: (...data: unknown[]) => void) =>
  window.runtime.EventsOn(eventName, callback)

export const EventsOff = (eventName: string) =>
  window.runtime.EventsOff(eventName)
