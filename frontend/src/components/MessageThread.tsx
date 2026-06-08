import { useState, useEffect, useRef } from 'react'
import { GetMessages, SendMessage, UIMessage } from '../wailsjs/go/app/App'
import Identicon from './Identicon'

interface Props {
  peerUsername: string | null
  incomingTick: number
}

const IconSend = () => (
  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <path d="M12 19V5M5 12l7-7 7 7" />
  </svg>
)
const IconLock = () => (
  <svg width="11" height="11" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <rect x="4.5" y="11" width="15" height="9" rx="2" /><path d="M8 11V8a4 4 0 0 1 8 0v3" />
  </svg>
)
const IconDots = () => (
  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <circle cx="5" cy="12" r="1.4" /><circle cx="12" cy="12" r="1.4" /><circle cx="19" cy="12" r="1.4" />
  </svg>
)
const IconCheckCheck = () => (
  <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <path d="M2 13l4 4 9-9M13 17l1 1 8-8" />
  </svg>
)
const IconRefresh = () => (
  <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round">
    <polyline points="23 4 23 10 17 10" /><polyline points="1 20 1 14 7 14" />
    <path d="M3.51 9a9 9 0 0114.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0020.49 15" />
  </svg>
)

const MONO_HUE = 70

interface BubbleProps {
  m: UIMessage & { peer: string }
  fresh: boolean
}

function Bubble({ m, fresh }: BubbleProps) {
  return (
    <div className={'brow ' + (m.outgoing ? 'brow-out' : 'brow-in')}>
      {!m.outgoing && (
        <Identicon name={m.peer} size={26} mono={MONO_HUE} />
      )}
      <div className={
        'bubble ' + (m.outgoing ? 'bubble-out' : 'bubble-in') + (fresh ? ' bubble-fresh' : '')
      } style={{ opacity: m.pending ? 0.7 : 1 }}>
        <span className="btext">{m.text}</span>
        <span className="bmeta">
          {fmtTime(m.timestamp)}
          {m.outgoing && !m.pending && (
            <span style={{ marginLeft: 4, opacity: 0.8, display: 'flex' }}>
              <IconCheckCheck />
            </span>
          )}
          {m.pending && <span className="queued">· queued</span>}
        </span>
      </div>
    </div>
  )
}

export default function MessageThread({ peerUsername, incomingTick }: Props) {
  const [messages, setMessages] = useState<UIMessage[]>([])
  const [loadError, setLoadError] = useState('')
  const [input, setInput] = useState('')
  const [sending, setSending] = useState(false)
  const [sendError, setSendError] = useState('')
  const [freshIdx, setFreshIdx] = useState(-1)
  const bottomRef = useRef<HTMLDivElement>(null)

  async function loadMessages() {
    if (!peerUsername) { setMessages([]); setLoadError(''); return }
    try {
      const msgs = await GetMessages(peerUsername)
      setMessages(msgs ?? [])
      setLoadError('')
    } catch (e) {
      setLoadError('Failed to load messages')
      console.error(e)
    }
  }

  // eslint-disable-next-line react-hooks/exhaustive-deps
  useEffect(() => { loadMessages() }, [peerUsername, incomingTick])

  useEffect(() => {
    if (bottomRef.current) {
      bottomRef.current.parentElement!.scrollTop = bottomRef.current.parentElement!.scrollHeight
    }
  }, [messages])

  async function handleSend(e: React.FormEvent) {
    e.preventDefault()
    const text = input.trim()
    if (!text || !peerUsername || sending) { return }
    setSending(true)
    setSendError('')
    setFreshIdx(messages.length)
    try {
      const err = await SendMessage(peerUsername, text)
      if (err) {
        setSendError(err)
      } else {
        setInput('')
        await loadMessages()
      }
    } catch (e) {
      setSendError('Send failed — try again')
      console.error(e)
    } finally {
      setSending(false)
    }
  }

  if (!peerUsername) {
    return (
      <div className="thread thread-empty">
        <div className="empty-card lg-enter">
          <div className="empty-mark">
            <IconLock />
          </div>
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
        <Identicon name={peerUsername} size={34} mono={MONO_HUE} />
        <div className="th-mid">
          <span className="peer-name">{peerUsername}</span>
          <span className="peer-sub"><IconLock /> LINK SECURE</span>
        </div>
        <button className="icon-btn-sm" title="More"><IconDots /></button>
      </div>

      <div className="messages">
        {loadError && (
          <div className="load-error">
            <span style={{ flex: 1 }}>{loadError}</span>
            <button onClick={loadMessages} className="retry-btn">
              <IconRefresh />
              <span>RETRY</span>
            </button>
          </div>
        )}
        {!loadError && messages.length === 0 && (
          <p className="no-msg">No messages yet. Say hello!</p>
        )}
        {messages.map((m, i) => (
          <Bubble key={i} m={{ ...m, peer: peerUsername }} fresh={i === freshIdx} />
        ))}
        <div ref={bottomRef} />
      </div>

      <form onSubmit={handleSend} className="composer">
        <div className="composer-field">
          <input
            value={input}
            onChange={e => setInput(e.target.value)}
            placeholder={`Message ${peerUsername}…`}
            className="msg-inp"
            autoFocus
            autoCapitalize="off"
            autoCorrect="off"
            spellCheck={false}
            onKeyDown={e => {
              if (e.key === 'Enter' && !e.shiftKey) { e.preventDefault(); handleSend(e) }
            }}
          />
        </div>
        <button
          type="submit"
          disabled={disabled}
          className="send-btn"
          style={{ opacity: disabled ? 0.4 : 1 }}
        >
          <IconSend />
        </button>
      </form>
      {sendError && <p className="send-err">{sendError}</p>}
    </div>
  )
}

function fmtTime(ts: number): string {
  const d = new Date(ts * 1000)
  const n = new Date()
  if (d.toDateString() === n.toDateString()) {
    return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  }
  return d.toLocaleDateString([], { month: 'short', day: 'numeric' })
    + ' ' + d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}
