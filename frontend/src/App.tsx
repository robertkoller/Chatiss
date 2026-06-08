import { useState, useEffect, useCallback } from 'react'
import { EventsOn } from './wailsjs/runtime/runtime'
import { GetContacts, Connect, UIContact, UIMessage } from './wailsjs/go/app/App'
import LoginScreen from './components/LoginScreen'
import Sidebar from './components/Sidebar'
import MessageThread from './components/MessageThread'

const DIR_KEY    = 'chatiss-dir'
const THEME_KEY  = 'chatiss-theme'
const ACCENT_KEY = 'chatiss-accent'

function loadPref(key: string, fallback: string): string {
  return localStorage.getItem(key) ?? fallback
}
function loadAccent(): string | null {
  const v = localStorage.getItem(ACCENT_KEY)
  return v || null
}

function hexToRgb(hex: string): string {
  const h = hex.replace('#', '')
  const full = h.length === 3 ? h.split('').map(c => c + c).join('') : h
  const n = parseInt(full, 16)
  return `${(n >> 16) & 255},${(n >> 8) & 255},${n & 255}`
}

function buildAccentStyle(accent: string | null, dir: string): React.CSSProperties {
  if (!accent) { return {} }
  const mix = (pct: number, c: string) => `color-mix(in oklab, ${accent}, ${c} ${pct}%)`
  const vars: Record<string, string> = {
    '--accent':       accent,
    '--accent-strong': mix(32, 'white'),
    '--accent-ghost': `color-mix(in srgb, ${accent} 14%, transparent)`,
    '--online':       accent,
    '--glow':         hexToRgb(accent),
  }
  if (dir === 'phosphor') {
    Object.assign(vars, {
      '--text':            mix(40, 'white'),
      '--text-muted':      mix(16, 'black'),
      '--text-faint':      mix(52, 'black'),
      '--border':          mix(70, 'black'),
      '--border-2':        mix(54, 'black'),
      '--bubble-out':      mix(80, 'black'),
      '--bubble-out-text': mix(44, 'white'),
      '--titlebar-text':   mix(38, 'black'),
    })
  }
  return vars as React.CSSProperties
}

export default function App() {
  const [loggedIn, setLoggedIn]         = useState(false)
  const [myUsername, setMyUsername]     = useState('')
  const [contacts, setContacts]         = useState<UIContact[]>([])
  const [selected, setSelected]         = useState<string | null>(null)
  const [incomingTick, setIncomingTick] = useState(0)

  const [dir, setDir]     = useState<string>(() => loadPref(DIR_KEY,   'phosphor'))
  const [theme, setTheme] = useState<string>(() => loadPref(THEME_KEY, 'dark'))
  const [accent, setAccent] = useState<string | null>(loadAccent)

  const [defaultDir, setDefaultDir]     = useState<string>(() => loadPref(DIR_KEY,   'phosphor'))
  const [defaultTheme, setDefaultTheme] = useState<string>(() => loadPref(THEME_KEY, 'dark'))
  const [defaultAccent, setDefaultAccent] = useState<string | null>(loadAccent)

  function handleSetDefault() {
    localStorage.setItem(DIR_KEY,   dir)
    localStorage.setItem(THEME_KEY, theme)
    if (accent) { localStorage.setItem(ACCENT_KEY, accent) } else { localStorage.removeItem(ACCENT_KEY) }
    setDefaultDir(dir)
    setDefaultTheme(theme)
    setDefaultAccent(accent)
  }

  const refreshContacts = useCallback(async () => {
    try {
      const list = await GetContacts()
      setContacts(list ?? [])
    } catch (e) {
      console.error('GetContacts failed:', e)
    }
  }, [])

  useEffect(() => {
    if (!loggedIn) { return }
    refreshContacts()

    const offMessage = EventsOn('message:received', (data: unknown) => {
      const msg = data as UIMessage
      setIncomingTick(t => t + 1)
      if (msg.from) { refreshContacts() }
    })
    const offAdded   = EventsOn('contact:added',   () => { refreshContacts() })
    const offOnline  = EventsOn('contact:online',  (data: unknown) => {
      const d = data as { username: string }
      setContacts(prev => prev.map(c => c.username === d.username ? { ...c, online: true  } : c))
    })
    const offOffline = EventsOn('contact:offline', (data: unknown) => {
      const d = data as { username: string }
      setContacts(prev => prev.map(c => c.username === d.username ? { ...c, online: false } : c))
    })

    return () => { offMessage(); offAdded(); offOnline(); offOffline() }
  }, [loggedIn, refreshContacts])

  const accentStyle = buildAccentStyle(accent, dir)

  return (
    <div className="chatiss" data-dir={dir} data-theme={theme} style={accentStyle}>
      <div className="app-body">
        {!loggedIn
          ? (
            <LoginScreen
              onLogin={(username) => { setMyUsername(username); setLoggedIn(true) }}
            />
          )
          : (
            <div className="two-pane">
              <Sidebar
                contacts={contacts}
                selected={selected}
                myUsername={myUsername}
                onSelect={(username) => { setSelected(username); Connect(username) }}
                onContactAdded={refreshContacts}
                dir={dir}
                theme={theme}
                accent={accent}
                defaultDir={defaultDir}
                defaultTheme={defaultTheme}
                defaultAccent={defaultAccent}
                onDirChange={setDir}
                onThemeChange={setTheme}
                onAccentChange={setAccent}
                onSetDefault={handleSetDefault}
              />
              <MessageThread peerUsername={selected} incomingTick={incomingTick} />
            </div>
          )
        }
      </div>
    </div>
  )
}
