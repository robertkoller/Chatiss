// screens.jsx — Login, Sidebar, MessageThread. Config-driven across 4 directions.
const { useState: uS, useEffect: uE, useRef: uR } = React

/* per-direction identicon shape + copy */
const SHAPE = { cipher: ['square', 1], haven: ['round', 6], phosphor: ['square', 0], brutal: ['square', 2] }
const sh = d => SHAPE[d] || ['square', 2]
const monoHue = d => d === 'phosphor' ? 70 : null
const COPY = {
  cipher:   { tag: 'e2e encrypted · peer-to-peer', unlock: 'Unlock', contacts: '// contacts', verified: 'e2e · verified', foot: 'No servers. No metadata. Keys never leave this device.' },
  haven:    { tag: 'End-to-end encrypted messaging', unlock: 'Unlock', contacts: 'Contacts', verified: 'Encrypted · verified', foot: 'No servers. No metadata. Keys never leave this device.' },
  phosphor: { tag: 'SECURE CHANNEL // P2P', unlock: 'Open Sesame', contacts: 'CONTACTS', verified: 'LINK SECURE', foot: 'NO SERVERS · NO METADATA · KEYS LOCAL' },
  brutal:   { tag: 'PEER-TO-PEER. NO SERVERS.', unlock: 'GET IN', contacts: 'CONTACTS', verified: 'VERIFIED', foot: 'No servers. No metadata. Keys never leave this device.' },
}

function useHover() {
  const [h, setH] = uS(false)
  return [h, { onMouseEnter: () => setH(true), onMouseLeave: () => setH(false) }]
}

/* ============ LOGIN ============ */
function LoginScreen({ dir, onLogin }) {
  const [passphrase, setPassphrase] = uS('')
  const [username, setUsername] = uS('')
  const [hidden, setHidden] = uS(true)
  const [error, setError] = uS('')
  const [loading, setLoading] = uS('')
  const [focus, setFocus] = uS('')
  const [btnH, btnHover] = useHover()
  const c = COPY[dir]

  async function handleSubmit(e) {
    e.preventDefault()
    if (!passphrase || !username) return
    setLoading(true); setError('')
    await new Promise(r => setTimeout(r, 900))
    if (passphrase.length < 4) { setError('Passphrase too short — try a longer secret'); setLoading(false); return }
    setLoading(false); onLogin(username)
  }

  const disabled = loading || !passphrase || !username
  return (
    <div className="lg-wrap">
      <div className="lg-card lg-enter">
        {dir === 'brutal' && <span className="sticker">★ ZERO&nbsp;LOGS</span>}
        <div className="lg-brandrow">
          <div className="lg-mark"><IconLock size={dir === 'haven' ? 20 : 18} sw={2.2} /></div>
          <div>
            <h1 className="lg-title">Chatiss</h1>
            <p className="lg-sub">{c.tag}</p>
          </div>
        </div>

        {dir === 'phosphor' && <div className="term-line">$ chatiss --unlock<span className="cursor" /></div>}

        <form onSubmit={handleSubmit} className="lg-form">
          <label className="lbl">Passphrase</label>
          <div className={'field ' + (focus === 'p' ? 'field-focus' : '')}>
            <input type={hidden ? 'password' : 'text'} value={passphrase}
              onChange={e => setPassphrase(e.target.value)} onFocus={() => setFocus('p')} onBlur={() => setFocus('')}
              placeholder="Your secret passphrase" className="inp" autoFocus autoCapitalize="off" autoCorrect="off" spellCheck={false} />
            <button type="button" className="ghost-btn" onClick={() => setHidden(h => !h)}
              title={hidden ? 'Show' : 'Hide'}>{hidden ? <IconEyeOff size={16} /> : <IconEye size={16} />}</button>
          </div>

          <label className="lbl">Username</label>
          <div className={'field ' + (focus === 'u' ? 'field-focus' : '')}>
            <input type="text" value={username} onChange={e => setUsername(e.target.value)}
              onFocus={() => setFocus('u')} onBlur={() => setFocus('')}
              placeholder="How others will find you" className="inp" autoCapitalize="off" autoCorrect="off" spellCheck={false} />
          </div>

          {error && <div className="errbox"><p className="errtext">{error}</p></div>}

          <button type="submit" disabled={disabled} {...btnHover}
            className="primary-btn" style={{ opacity: disabled ? 0.5 : 1, transform: btnH && !disabled ? 'translateY(-1px)' : 'none' }}>
            {loading ? <span className="dots3"><i /><i /><i /></span> : error ? 'Try Again' : c.unlock}
          </button>
        </form>

        <div className="lg-foot"><IconShield size={13} /><span>{c.foot}</span></div>
      </div>
    </div>
  )
}

/* ============ SIDEBAR ============ */
function ContactRow({ dir, c, active, onSelect }) {
  const [h, hov] = useHover()
  const [shape, rad] = sh(dir)
  return (
    <button {...hov} onClick={() => onSelect(c.username)}
      className={'contact ' + (active ? 'contact-active' : '')}
      style={{ background: active ? 'var(--accent-ghost)' : h ? 'var(--surface-2)' : 'transparent' }}>
      <div className="av-wrap">
        <Identicon name={c.username} size={dir === 'haven' ? 34 : 32} shape={shape} radius={rad} mono={monoHue(dir)} />
        <span className="presence" style={{ background: c.online ? 'var(--online)' : 'var(--offline)' }} />
      </div>
      <div className="contact-mid">
        <div className="contact-top">
          <span className="contact-name">{c.username}</span>
          <span className="contact-time">{c.t}</span>
        </div>
        <div className="contact-bot">
          <span className="contact-prev">{c.preview}</span>
          {c.unread > 0 && <span className="badge">{c.unread}</span>}
        </div>
      </div>
    </button>
  )
}

function Sidebar({ dir, contacts, selected, myUsername, onSelect, onContactAdded }) {
  const [adding, setAdding] = uS(false)
  const [newUsername, setNewUsername] = uS('')
  const [addError, setAddError] = uS('')
  const [addLoading, setAddLoading] = uS(false)
  const [q, setQ] = uS('')
  const [addH, addHov] = useHover()
  const [shape, rad] = sh(dir)

  async function handleAdd(e) {
    e.preventDefault()
    if (!newUsername.trim()) return
    setAddLoading(true); setAddError('')
    await new Promise(r => setTimeout(r, 700))
    setAddLoading(false)
    if (contacts.some(c => c.username === newUsername.trim())) { setAddError('Already in your contacts'); return }
    setNewUsername(''); setAdding(false); onContactAdded(newUsername.trim())
  }

  const filtered = contacts.filter(c => c.username.toLowerCase().includes(q.toLowerCase()))

  return (
    <div className="sidebar">
      <div className="sb-head">
        <div className="sb-me">
          <Identicon name={myUsername} size={30} shape={shape} radius={rad} mono={monoHue(dir)} />
          <div className="sb-me-txt">
            <span className="sb-me-name">{myUsername}</span>
            <span className="sb-me-status"><span className="me-dot" />online</span>
          </div>
        </div>
        <button {...addHov} className="icon-btn" onClick={() => { setAdding(v => !v); setAddError('') }}
          style={{ transform: adding ? 'rotate(45deg)' : addH ? 'scale(1.08)' : 'none' }} title={adding ? 'Cancel' : 'Add contact'}>
          <IconPlus size={18} />
        </button>
      </div>

      <div className="sb-search">
        <IconSearch size={14} />
        <input value={q} onChange={e => setQ(e.target.value)} placeholder="Search" className="search-inp" />
      </div>

      <div className={'addform ' + (adding ? 'addform-open' : '')}>
        <form onSubmit={handleAdd} className="addform-inner">
          <input autoFocus={adding} value={newUsername} onChange={e => setNewUsername(e.target.value)} placeholder="Username to add" className="add-inp" />
          {addError && <p className="add-err">{addError}</p>}
          <button type="submit" disabled={addLoading || !newUsername.trim()} className="add-submit"
            style={{ opacity: addLoading || !newUsername.trim() ? 0.5 : 1 }}>{addLoading ? 'Looking up…' : 'Add contact'}</button>
        </form>
      </div>

      <div className="sb-list">
        <div className="sb-section">{COPY[dir].contacts}</div>
        {filtered.length === 0 && <p className="sb-empty">No contacts.<br />Press + to add one.</p>}
        {filtered.map(c => <ContactRow key={c.username} dir={dir} c={c} active={selected === c.username} onSelect={onSelect} />)}
      </div>
    </div>
  )
}

/* ============ MESSAGE THREAD ============ */
function Bubble({ dir, m, fresh }) {
  const [shape, rad] = sh(dir)
  return (
    <div className={'brow ' + (m.outgoing ? 'brow-out' : 'brow-in')}>
      {!m.outgoing && <Identicon name={m.peer} size={26} shape={shape} radius={Math.max(0, rad - 1)} mono={monoHue(dir)} />}
      <div className={'bubble ' + (m.outgoing ? 'bubble-out' : 'bubble-in') + (fresh ? ' bubble-fresh' : '')} style={{ opacity: m.pending ? 0.7 : 1 }}>
        {dir === 'phosphor' && <span className="bubble-tag">{m.outgoing ? 'you' : m.peer}&gt;</span>}
        <span className="btext">{m.text}</span>
        <span className="bmeta">
          {fmtTime(m.timestamp)}
          {m.outgoing && !m.pending && <IconCheckCheck size={13} style={{ marginLeft: 4, opacity: 0.8 }} />}
          {m.pending && <span className="queued">· queued</span>}
        </span>
      </div>
    </div>
  )
}

function MessageThread({ dir, peerUsername, messages, typing, onSend }) {
  const [input, setInput] = uS('')
  const [sending, setSending] = uS(false)
  const [sendError, setSendError] = uS('')
  const [freshIdx, setFreshIdx] = uS(-1)
  const bottomRef = uR(null)
  const [sendH, sendHov] = useHover()
  const [shape, rad] = sh(dir)

  uE(() => { bottomRef.current && (bottomRef.current.parentElement.scrollTop = bottomRef.current.parentElement.scrollHeight) }, [messages, typing])

  async function handleSend(e) {
    e.preventDefault()
    const text = input.trim()
    if (!text || sending) return
    setSending(true); setSendError('')
    setFreshIdx(messages.length)
    onSend(text)
    setInput(''); setSending(false)
  }

  if (!peerUsername) {
    return (
      <div className="thread thread-empty">
        <div className="empty-card lg-enter">
          <div className="empty-mark"><IconLock size={26} sw={1.6} /></div>
          <p className="empty-title">Select a conversation</p>
          <p className="empty-sub">Messages are end-to-end encrypted and stored only on your device.</p>
        </div>
      </div>
    )
  }

  const disabled = sending || !input.trim()
  return (
    <div className="thread">
      <div className="thread-head">
        <Identicon name={peerUsername} size={34} shape={shape} radius={rad} mono={monoHue(dir)} />
        <div className="th-mid">
          <span className="peer-name">{peerUsername}</span>
          <span className="peer-sub"><IconLock size={11} /> {COPY[dir].verified}</span>
        </div>
        <button className="icon-btn-sm" title="More"><IconDots size={18} /></button>
      </div>

      <div className="messages">
        {messages.length === 0 && <p className="no-msg">No messages yet. Say hello!</p>}
        {messages.map((m, i) => <Bubble key={i} dir={dir} m={{ ...m, peer: peerUsername }} fresh={i === freshIdx} />)}
        {typing && (
          <div className="brow brow-in">
            <Identicon name={peerUsername} size={26} shape={shape} radius={Math.max(0, rad - 1)} mono={monoHue(dir)} />
            <div className="bubble bubble-in typing"><span className="tdot" /><span className="tdot" /><span className="tdot" /></div>
          </div>
        )}
        <div ref={bottomRef} />
      </div>

      <form onSubmit={handleSend} className="composer">
        <div className="composer-field">
          <input value={input} onChange={e => setInput(e.target.value)} placeholder={`Message ${peerUsername}…`}
            className="msg-inp" autoFocus autoCapitalize="off" autoCorrect="off" spellCheck={false}
            onKeyDown={e => { if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); handleSend(e) } }} />
        </div>
        <button {...sendHov} type="submit" disabled={disabled} className="send-btn"
          style={{ opacity: disabled ? 0.4 : 1, transform: sendH && !disabled ? 'scale(1.07)' : 'none' }}>
          <IconSend size={18} />
        </button>
      </form>
      {sendError && <p className="send-err">{sendError}</p>}
    </div>
  )
}

Object.assign(window, { LoginScreen, Sidebar, MessageThread, useHover })
