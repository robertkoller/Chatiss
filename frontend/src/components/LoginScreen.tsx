import { useState } from 'react'
import { Login } from '../wailsjs/go/app/App'

interface Props {
  onLogin: (username: string) => void
}

const IconEye = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7-10-7-10-7Z" /><circle cx="12" cy="12" r="3" />
  </svg>
)
const IconEyeOff = () => (
  <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <path d="M3 3l18 18" /><path d="M10.6 5.1A10.9 10.9 0 0 1 12 5c6.5 0 10 7 10 7a17.7 17.7 0 0 1-3.4 4.3M6.6 6.6A17.6 17.6 0 0 0 2 12s3.5 7 10 7a10.8 10.8 0 0 0 4.2-.8" /><path d="M9.9 9.9a3 3 0 0 0 4.2 4.2" />
  </svg>
)
const IconLock = () => (
  <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round">
    <rect x="4.5" y="11" width="15" height="9" rx="2" /><path d="M8 11V8a4 4 0 0 1 8 0v3" />
  </svg>
)
const IconShield = () => (
  <svg width="13" height="13" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
    <path d="M12 3l7 3v5c0 4.5-3 7.7-7 9-4-1.3-7-4.5-7-9V6l7-3Z" /><path d="M9 12l2 2 4-4" />
  </svg>
)

export default function LoginScreen({ onLogin }: Props) {
  const [passphrase, setPassphrase] = useState('')
  const [username, setUsername] = useState('')
  const [hidden, setHidden] = useState(true)
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const [focusField, setFocusField] = useState('')

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault()
    if (!passphrase || !username) { return }
    setLoading(true)
    setError('')
    try {
      const err = await Login(passphrase, username)
      if (err) {
        setError(err)
      } else {
        onLogin(username)
      }
    } catch (e) {
      setError('Unexpected error — please try again')
      console.error(e)
    } finally {
      setLoading(false)
    }
  }

  const disabled = loading || !passphrase || !username

  return (
    <div className="lg-wrap">
      <div className="lg-card lg-enter">
        <div className="lg-brandrow">
          <div className="lg-mark">
            <IconLock />
          </div>
          <div>
            <h1 className="lg-title">Chatiss</h1>
            <p className="lg-sub">SECURE CHANNEL // P2P</p>
          </div>
        </div>

        <div className="term-line">$ chatiss --unlock<span className="cursor" /></div>

        <form onSubmit={handleSubmit} className="lg-form">
          <label className="lbl">Passphrase</label>
          <div className={'field ' + (focusField === 'p' ? 'field-focus' : '')}>
            <input
              type={hidden ? 'password' : 'text'}
              value={passphrase}
              onChange={e => setPassphrase(e.target.value)}
              onFocus={() => setFocusField('p')}
              onBlur={() => setFocusField('')}
              placeholder="Your secret passphrase"
              className="inp"
              autoFocus
              autoCapitalize="off"
              autoCorrect="off"
              spellCheck={false}
            />
            <button type="button" className="ghost-btn" onClick={() => setHidden(h => !h)}
              title={hidden ? 'Show' : 'Hide'}>
              {hidden ? <IconEyeOff /> : <IconEye />}
            </button>
          </div>

          <label className="lbl">Username</label>
          <div className={'field ' + (focusField === 'u' ? 'field-focus' : '')}>
            <input
              type="text"
              value={username}
              onChange={e => setUsername(e.target.value)}
              onFocus={() => setFocusField('u')}
              onBlur={() => setFocusField('')}
              placeholder="How others will find you"
              className="inp"
              autoCapitalize="off"
              autoCorrect="off"
              spellCheck={false}
            />
          </div>

          {error && (
            <div className="errbox">
              <p className="errtext">{error}</p>
            </div>
          )}

          <button
            type="submit"
            disabled={disabled}
            className="primary-btn"
            style={{ opacity: disabled ? 0.5 : 1 }}
          >
            {loading
              ? <span className="dots3"><i /><i /><i /></span>
              : error ? 'Try Again' : 'Open Sesame'
            }
          </button>
        </form>

        <div className="lg-foot">
          <IconShield />
          <span>NO SERVERS · NO METADATA · KEYS LOCAL</span>
        </div>
      </div>
    </div>
  )
}
