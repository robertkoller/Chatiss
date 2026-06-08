import { useState, useRef } from 'react'
import { AddContact, UIContact } from '../wailsjs/go/app/App'
import Identicon from './Identicon'

interface Props {
  contacts: UIContact[]
  selected: string | null
  myUsername: string
  onSelect: (username: string) => void
  onContactAdded: () => void
  dir: string
  theme: string
  accent: string | null
  defaultDir: string
  defaultTheme: string
  defaultAccent: string | null
  onDirChange: (dir: string) => void
  onThemeChange: (theme: string) => void
  onAccentChange: (accent: string | null) => void
  onSetDefault: () => void
}

const IconPlus = () => (
  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <path d="M12 5v14M5 12h14" />
  </svg>
)
const IconSearch = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <circle cx="11" cy="11" r="7" /><path d="M21 21l-4-4" />
  </svg>
)
const IconSettings = () => (
  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <circle cx="12" cy="12" r="3" />
    <path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 0 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 0 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 0 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 0 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 0 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 0 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 0 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 0 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z" />
  </svg>
)
const IconSun = () => (
  <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <circle cx="12" cy="12" r="4" /><path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4" />
  </svg>
)
const IconMoon = () => (
  <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <path d="M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8Z" />
  </svg>
)

const MONO_HUE = 70

const DIRS = [
  { id: 'phosphor', label: 'Phosphor' },
  { id: 'brutal',   label: 'Brutal'   },
  { id: 'cipher',   label: 'Cipher'   },
  { id: 'haven',    label: 'Haven'    },
]

const SWATCHES: { name: string; hex: string | null }[] = [
  { name: 'Theme default', hex: null       },
  { name: 'Toxic green',   hex: '#1fff7a'  },
  { name: 'Amber',         hex: '#ffb02e'  },
  { name: 'Cyan',          hex: '#38e0ff'  },
  { name: 'Klein blue',    hex: '#5b7bff'  },
  { name: 'Magenta',       hex: '#ff5cc8'  },
  { name: 'Red alert',     hex: '#ff5c5c'  },
]

interface ContactRowProps {
  c: UIContact
  active: boolean
  onSelect: (username: string) => void
}

function ContactRow({ c, active, onSelect }: ContactRowProps) {
  return (
    <button
      onClick={() => onSelect(c.username)}
      className={'contact ' + (active ? 'contact-active' : '')}
      style={{ background: active ? 'var(--accent-ghost)' : 'transparent' }}
    >
      <div className="av-wrap">
        <Identicon name={c.username} size={32} mono={MONO_HUE} />
        <span className="presence" style={{ background: c.online ? 'var(--online)' : 'var(--offline)' }} />
      </div>
      <div className="contact-mid">
        <div className="contact-top">
          <span className="contact-name">{c.username}</span>
        </div>
        <div className="contact-bot">
          <span className="contact-prev">{c.online ? 'online' : 'offline'}</span>
        </div>
      </div>
    </button>
  )
}

export default function Sidebar({
  contacts, selected, myUsername, onSelect, onContactAdded,
  dir, theme, accent, defaultDir, defaultTheme, defaultAccent,
  onDirChange, onThemeChange, onAccentChange, onSetDefault,
}: Props) {
  const [adding, setAdding]           = useState(false)
  const [newUsername, setNewUsername] = useState('')
  const [addError, setAddError]       = useState('')
  const [addLoading, setAddLoading]   = useState(false)
  const [q, setQ]                     = useState('')
  const [settingsOpen, setSettingsOpen] = useState(false)
  const colorInputRef = useRef<HTMLInputElement>(null)

  async function handleAdd(e: React.FormEvent) {
    e.preventDefault()
    if (!newUsername.trim()) { return }
    setAddLoading(true)
    setAddError('')
    const err = await AddContact(newUsername.trim())
    setAddLoading(false)
    if (err) {
      setAddError(err)
    } else {
      setNewUsername('')
      setAdding(false)
      onContactAdded()
    }
  }

  const filtered = contacts.filter(c =>
    c.username.toLowerCase().includes(q.toLowerCase())
  )

  const isCustomAccent = accent !== null && !SWATCHES.some(s => s.hex === accent)
  const isDefault = dir === defaultDir && theme === defaultTheme && accent === defaultAccent

  return (
    <div className="sidebar">

      {/* Header */}
      <div className="sb-head">
        <div className="sb-me">
          <Identicon name={myUsername} size={30} mono={MONO_HUE} />
          <div className="sb-me-txt">
            <span className="sb-me-name">{myUsername}</span>
            <span className="sb-me-status"><span className="me-dot" />online</span>
          </div>
        </div>
        <button
          className="icon-btn"
          onClick={() => { setAdding(v => !v); setAddError('') }}
          style={{ transform: adding ? 'rotate(45deg)' : 'none' }}
          title={adding ? 'Cancel' : 'Add contact'}
        >
          <IconPlus />
        </button>
      </div>

      {/* Search */}
      <div className="sb-search">
        <IconSearch />
        <input
          value={q}
          onChange={e => setQ(e.target.value)}
          placeholder="Search"
          className="search-inp"
        />
      </div>

      {/* Add contact form (animated expand) */}
      <div className={'addform ' + (adding ? 'addform-open' : '')}>
        <form onSubmit={handleAdd} className="addform-inner">
          <input
            autoFocus={adding}
            value={newUsername}
            onChange={e => setNewUsername(e.target.value)}
            placeholder="Username to add"
            className="add-inp"
            autoCapitalize="off"
            autoCorrect="off"
            spellCheck={false}
          />
          {addError && <p className="add-err">{addError}</p>}
          <button
            type="submit"
            disabled={addLoading || !newUsername.trim()}
            className="add-submit"
            style={{ opacity: addLoading || !newUsername.trim() ? 0.5 : 1 }}
          >
            {addLoading ? 'Looking up…' : 'Add contact'}
          </button>
        </form>
      </div>

      {/* Contact list */}
      <div className="sb-list">
        <div className="sb-section">CONTACTS</div>
        {filtered.length === 0 && (
          <p className="sb-empty">No contacts.<br />Press + to add one.</p>
        )}
        {filtered.map(c => (
          <ContactRow key={c.username} c={c} active={selected === c.username} onSelect={onSelect} />
        ))}
      </div>

      {/* Settings panel (expands above the bar) */}
      <div className={'sb-settings-panel ' + (settingsOpen ? 'sb-settings-panel-open' : '')}>
        <div className="sb-settings-inner">

          <p className="settings-section">DESIGN STYLE</p>
          <div className="dir-grid">
            {DIRS.map(d => (
              <button
                key={d.id}
                className={'dir-btn ' + (dir === d.id ? 'dir-btn-active' : '')}
                onClick={() => onDirChange(d.id)}
              >
                {d.label}
                {defaultDir === d.id && defaultTheme === theme && defaultAccent === accent && (
                  <span className="dir-btn-star">★</span>
                )}
              </button>
            ))}
          </div>

          <p className="settings-section">THEME</p>
          <div className="theme-row">
            <button
              className={'theme-btn ' + (theme === 'dark' ? 'theme-btn-active' : '')}
              onClick={() => onThemeChange('dark')}
            >
              <IconMoon />Dark
            </button>
            <button
              className={'theme-btn ' + (theme === 'light' ? 'theme-btn-active' : '')}
              onClick={() => onThemeChange('light')}
            >
              <IconSun />Light
            </button>
          </div>

          <p className="settings-section">ACCENT COLOR</p>
          <div className="swatches">
            {SWATCHES.map(s => (
              <button
                key={s.name}
                title={s.name}
                className={
                  'sw '
                  + (!s.hex ? 'sw-default ' : '')
                  + (accent === s.hex ? 'sw-on' : '')
                }
                style={s.hex ? { background: s.hex } : undefined}
                onClick={() => onAccentChange(s.hex)}
              />
            ))}
            {/* Custom color picker — invisible input overlaid on a swatch */}
            <div className="sw-picker-wrap" title="Custom color">
              <button
                className={'sw ' + (isCustomAccent ? 'sw-on' : 'sw-custom')}
                style={isCustomAccent ? { background: accent! } : undefined}
                onClick={() => colorInputRef.current?.click()}
              >
                {!isCustomAccent && <span className="sw-plus">+</span>}
              </button>
              <input
                ref={colorInputRef}
                type="color"
                value={accent && !SWATCHES.some(s => s.hex === accent) ? accent : '#1fff7a'}
                onChange={e => onAccentChange(e.target.value)}
                className="sw-color-input"
              />
            </div>
          </div>

          <button
            className={'default-btn ' + (isDefault ? 'default-btn-active' : '')}
            onClick={onSetDefault}
            disabled={isDefault}
          >
            {isDefault ? '★ Current default' : '☆ Set as default'}
          </button>

        </div>
      </div>

      {/* Settings bar (always visible, toggles panel) */}
      <button className="sb-settings-bar" onClick={() => setSettingsOpen(v => !v)}>
        <IconSettings />
        <span className="sb-settings-name">
          {DIRS.find(d => d.id === dir)?.label} · {theme === 'dark' ? 'Dark' : 'Light'}
          {accent && <span className="sb-settings-dot" style={{ background: accent }} />}
        </span>
        {isDefault && <span className="sb-settings-badge">DEFAULT</span>}
        <span className="sb-settings-arrow" style={{ transform: settingsOpen ? 'rotate(180deg)' : 'none' }}>▾</span>
      </button>

    </div>
  )
}
