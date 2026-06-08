// Auto-generated Wails Go bindings for app.App
// These proxy calls through window.go which is injected by the Wails runtime.

export interface UIContact {
  username: string
  online: boolean
}

export interface UIMessage {
  from: string
  text: string
  timestamp: number
  outgoing: boolean
  pending: boolean
}

function call<T>(method: string, ...args: unknown[]): Promise<T> {
  const fn = window.go['app']['App'][method] as (...a: unknown[]) => Promise<T>
  return fn(...args)
}

export const Login = (passphrase: string, username: string): Promise<string> =>
  call('Login', passphrase, username)

export const GetContacts = (): Promise<UIContact[]> =>
  call('GetContacts')

export const AddContact = (username: string): Promise<string> =>
  call('AddContact', username)

export const GetMessages = (username: string): Promise<UIMessage[]> =>
  call('GetMessages', username)

export const Connect = (username: string): Promise<void> =>
  call('Connect', username)

export const SendMessage = (username: string, text: string): Promise<string> =>
  call('SendMessage', username, text)
