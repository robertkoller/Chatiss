// app.jsx — Shell: window chrome, login→app flow, direction/theme switch, auto-reply.
const { useState: US, useEffect: UE, useRef: UR } = React

function Segmented({ value, onChange, options }) {
  return (
    <div className="seg">
      {options.map(o => (
        <button key={o.id} className={'seg-btn ' + (value === o.id ? 'seg-on' : '')} onClick={() => onChange(o.id)}>
          {o.label}
        </button>
      ))}
    </div>
  )
}

function hexToRgb(hex) {
  const h = hex.replace('#', '')
  const n = parseInt(h.length === 3 ? h.split('').map(c => c + c).join('') : h, 16)
  return `${(n >> 16) & 255},${(n >> 8) & 255},${n & 255}`
}
const SWATCHES = [
  { name: 'Theme default', hex: null },
  { name: 'Toxic green', hex: '#1fff7a' },
  { name: 'Amber', hex: '#ffb02e' },
  { name: 'Cyan', hex: '#38e0ff' },
  { name: 'Klein blue', hex: '#5b7bff' },
  { name: 'Magenta', hex: '#ff5cc8' },
  { name: 'Red alert', hex: '#ff5c5c' },
]

function Swatches({ accent, onPick }) {
  return (
    <div className="swatches" title="Accent color — click to recolor">
      {SWATCHES.map(s => (
        <button key={s.name} title={s.name}
          className={'sw ' + (!s.hex ? 'sw-default ' : '') + ((accent === s.hex) || (!accent && !s.hex) ? 'sw-on' : '')}
          style={s.hex ? { background: s.hex } : undefined}
          onClick={() => onPick(s.hex)} />
      ))}
    </div>
  )
}

function App() {
  const [dir, setDir] = US('phosphor')
  const [theme, setTheme] = US('dark')
  const [accent, setAccent] = US(null)
  const mix = (pct, c) => `color-mix(in oklab, ${accent}, ${c} ${pct}%)`
  let accentStyle
  if (accent) {
    accentStyle = {
      '--accent': accent,
      '--accent-strong': mix(32, 'white'),
      '--accent-ghost': `color-mix(in srgb, ${accent} 14%, transparent)`,
      '--online': accent,
      '--glow': hexToRgb(accent),
    }
    if (dir === 'phosphor') Object.assign(accentStyle, {
      '--text': mix(40, 'white'),
      '--text-muted': mix(16, 'black'),
      '--text-faint': mix(52, 'black'),
      '--border': mix(70, 'black'),
      '--border-2': mix(54, 'black'),
      '--bubble-out': mix(80, 'black'),
      '--bubble-out-text': mix(44, 'white'),
      '--titlebar-text': mix(38, 'black'),
    })
  }
  const TITLEBAR = { cipher: 'chatiss · secure session', haven: 'Chatiss', phosphor: 'chatiss@local — secure tty', brutal: 'CHATISS' }
  const [screen, setScreen] = US('login') // login | app
  const [me, setMe] = US('you')
  const [contacts, setContacts] = US(SAMPLE_CONTACTS)
  const [selected, setSelected] = US('mara.k')
  const [threads, setThreads] = US(SAMPLE_THREADS)
  const [typing, setTyping] = US(false)
  const replyTimer = UR(null)

  function login(username) { setMe(username || 'you'); setScreen('app') }
  function logout() { setScreen('login'); setSelected('mara.k') }

  function addContact(username) {
    setContacts(cs => [{ username, online: Math.random() > 0.5, preview: 'New contact added', t: 'now', unread: 0 }, ...cs])
    setThreads(t => ({ ...t, [username]: [] }))
    setSelected(username)
  }

  function selectContact(u) {
    setSelected(u)
    setContacts(cs => cs.map(c => c.username === u ? { ...c, unread: 0 } : c))
  }

  function sendMessage(text) {
    const peer = selected
    const ts = Math.floor(Date.now() / 1000)
    setThreads(t => ({ ...t, [peer]: [...(t[peer] || []), { text, outgoing: true, timestamp: ts }] }))
    setContacts(cs => cs.map(c => c.username === peer ? { ...c, preview: text, t: 'now' } : c))
    // simulated encrypted reply
    clearTimeout(replyTimer.current)
    replyTimer.current = setTimeout(() => {
      setTyping(true)
      setTimeout(() => {
        setTyping(false)
        const reply = AUTO_REPLIES[Math.floor(Math.random() * AUTO_REPLIES.length)]
        setThreads(t => ({ ...t, [peer]: [...(t[peer] || []), { text: reply, outgoing: false, timestamp: Math.floor(Date.now() / 1000) }] }))
        setContacts(cs => cs.map(c => c.username === peer ? { ...c, preview: reply, t: 'now' } : c))
      }, 1400)
    }, 700)
  }

  UE(() => () => clearTimeout(replyTimer.current), [])

  const ThemeIcon = theme === 'dark' ? IconSun : IconMoon

  return (
    <div className="stage">
      {/* prototype controls */}
      <div className="proto-bar">
        <div className="proto-left">
          <span className="proto-logo"><IconLock size={15} /></span>
          <span className="proto-title">Chatiss — redesign explorations</span>
        </div>
        <div className="proto-right">
          <Segmented value={dir} onChange={setDir} options={[{ id: 'phosphor', label: 'Phosphor' }, { id: 'brutal', label: 'Brutal' }, { id: 'cipher', label: 'Cipher' }, { id: 'haven', label: 'Haven' }]} />
          <span className="proto-div" />
          <Swatches accent={accent} onPick={setAccent} />
          <button className="proto-icon" onClick={() => setTheme(t => t === 'dark' ? 'light' : 'dark')} title="Toggle theme">
            <ThemeIcon size={16} />
          </button>
          {screen === 'app' && <button className="proto-text" onClick={logout}>Lock</button>}
        </div>
      </div>

      {/* the app window */}
      <div className="window-shell">
        <div className={'chatiss'} data-dir={dir} data-theme={theme} style={accentStyle}>
          <div className="titlebar">
            <div className="lights"><span className="l-r" /><span className="l-y" /><span className="l-g" /></div>
            <div className="titlebar-label">{TITLEBAR[dir]}</div>
            <div style={{ width: 52 }} />
          </div>

          <div className="app-body">
            {screen === 'login'
              ? <LoginScreen key={'login-' + dir} dir={dir} onLogin={login} />
              : (
                <div className="two-pane" key={'app-' + dir}>
                  <Sidebar dir={dir} contacts={contacts} selected={selected} myUsername={me}
                    onSelect={selectContact} onContactAdded={addContact} />
                  <MessageThread dir={dir} peerUsername={selected} messages={threads[selected] || []} typing={typing} onSend={sendMessage} />
                </div>
              )}
          </div>
        </div>
      </div>

      <p className="proto-hint">Switch direction &amp; theme above, and click a <b>swatch</b> to recolor the accent live. Try logging in, switching contacts, and sending a message.</p>
    </div>
  )
}

ReactDOM.createRoot(document.getElementById('root')).render(<App />)
