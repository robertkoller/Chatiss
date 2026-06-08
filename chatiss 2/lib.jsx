// lib.jsx — icons, identicon generator, sample data. Exports to window.
const { useState, useEffect, useRef } = React

/* ---------- hashing ---------- */
function hashStr(s) {
  let h = 2166136261
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i)
    h = Math.imul(h, 16777619)
  }
  return h >>> 0
}

/* ---------- Identicon ----------
   5x5 symmetric grid derived from username hash. `shape` controls
   square (cipher) vs rounded (haven). Color from hue of hash. */
function Identicon({ name, size = 30, shape = 'square', radius = 0, mono = null }) {
  const h = hashStr(name || '?')
  const hue = h % 360
  // second hue for a subtle two-tone
  const hue2 = (hue + 38) % 360
  const cells = 5
  const cell = size / cells
  const bits = []
  let x = h
  for (let i = 0; i < 15; i++) { bits.push(x & 1); x = Math.imul(x ^ (x >>> 3), 2654435761) >>> 0 }
  const fillFor = two => mono != null
    ? `oklch(${two ? 0.82 : 0.6} 0.15 ${mono})`
    : `oklch(0.7 0.13 ${two ? hue2 : hue})`
  const rects = []
  let bi = 0
  for (let col = 0; col < 3; col++) {
    for (let row = 0; row < 5; row++) {
      const on = bits[bi++ % bits.length]
      if (!on) continue
      const cols = col === 2 ? [2] : [col, 4 - col]
      const two = (row + col) % 3 === 0
      cols.forEach(c => {
        rects.push(
          <rect key={c + '-' + row} x={c * cell} y={row * cell} width={cell + 0.4} height={cell + 0.4}
            rx={radius} ry={radius}
            fill={fillFor(two)} />
        )
      })
    }
  }
  const bg = mono != null ? `oklch(0.2 0.05 ${mono})` : `oklch(0.32 0.04 ${hue})`
  return (
    <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`} style={{ display: 'block', borderRadius: shape === 'round' ? size / 3.2 : radius, overflow: 'hidden', flexShrink: 0 }}>
      <rect x="0" y="0" width={size} height={size} fill={bg} />
      {rects}
    </svg>
  )
}

/* ---------- Inline SVG icons ---------- */
const ic = (p) => ({ width: p.size || 16, height: p.size || 16, viewBox: '0 0 24 24', fill: 'none', stroke: 'currentColor', strokeWidth: p.sw || 2, strokeLinecap: 'round', strokeLinejoin: 'round', style: p.style })
const IconPlus = (p = {}) => <svg {...ic(p)}><path d="M12 5v14M5 12h14" /></svg>
const IconClose = (p = {}) => <svg {...ic(p)}><path d="M6 6l12 12M18 6L6 18" /></svg>
const IconEye = (p = {}) => <svg {...ic(p)}><path d="M2 12s3.5-7 10-7 10 7 10 7-3.5 7-10 7-10-7-10-7Z" /><circle cx="12" cy="12" r="3" /></svg>
const IconEyeOff = (p = {}) => <svg {...ic(p)}><path d="M3 3l18 18" /><path d="M10.6 5.1A10.9 10.9 0 0 1 12 5c6.5 0 10 7 10 7a17.7 17.7 0 0 1-3.4 4.3M6.6 6.6A17.6 17.6 0 0 0 2 12s3.5 7 10 7a10.8 10.8 0 0 0 4.2-.8" /><path d="M9.9 9.9a3 3 0 0 0 4.2 4.2" /></svg>
const IconSend = (p = {}) => <svg {...ic(p)}><path d="M12 19V5M5 12l7-7 7 7" /></svg>
const IconLock = (p = {}) => <svg {...ic(p)}><rect x="4.5" y="11" width="15" height="9" rx="2" /><path d="M8 11V8a4 4 0 0 1 8 0v3" /></svg>
const IconSun = (p = {}) => <svg {...ic(p)}><circle cx="12" cy="12" r="4" /><path d="M12 2v2M12 20v2M4.9 4.9l1.4 1.4M17.7 17.7l1.4 1.4M2 12h2M20 12h2M4.9 19.1l1.4-1.4M17.7 6.3l1.4-1.4" /></svg>
const IconMoon = (p = {}) => <svg {...ic(p)}><path d="M21 12.8A9 9 0 1 1 11.2 3a7 7 0 0 0 9.8 9.8Z" /></svg>
const IconCheck = (p = {}) => <svg {...ic(p)}><path d="M5 13l4 4L19 7" /></svg>
const IconCheckCheck = (p = {}) => <svg {...ic(p)}><path d="M2 13l4 4 9-9M13 17l1 1 8-8" /></svg>
const IconSearch = (p = {}) => <svg {...ic(p)}><circle cx="11" cy="11" r="7" /><path d="M21 21l-4-4" /></svg>
const IconShield = (p = {}) => <svg {...ic(p)}><path d="M12 3l7 3v5c0 4.5-3 7.7-7 9-4-1.3-7-4.5-7-9V6l7-3Z" /><path d="M9 12l2 2 4-4" /></svg>
const IconDots = (p = {}) => <svg {...ic(p)}><circle cx="5" cy="12" r="1.4" /><circle cx="12" cy="12" r="1.4" /><circle cx="19" cy="12" r="1.4" /></svg>
const IconArrowLeft = (p = {}) => <svg {...ic(p)}><path d="M19 12H5M12 19l-7-7 7-7" /></svg>

/* ---------- Sample data ---------- */
const SAMPLE_CONTACTS = [
  { username: 'mara.k', online: true, preview: 'sent you the keys 🔑 — wiped after read', t: '14:22', unread: 2 },
  { username: 'ghostwire', online: true, preview: 'rotate the passphrase tonight', t: '13:58', unread: 0 },
  { username: 'len', online: false, preview: 'ok talk tomorrow', t: 'Tue', unread: 0 },
  { username: 'oksana', online: true, preview: 'meeting moved to the safehouse', t: 'Mon', unread: 0 },
  { username: 'r0ot', online: false, preview: 'verify my fingerprint when you can', t: 'Mon', unread: 0 },
  { username: 'priya.dev', online: false, preview: 'pushed the signed build', t: 'Sun', unread: 0 },
]

const now = Math.floor(Date.now() / 1000)
const SAMPLE_THREADS = {
  'mara.k': [
    { text: 'hey — you around?', outgoing: false, timestamp: now - 3600 },
    { text: 'yeah, channel is clean. go ahead', outgoing: true, timestamp: now - 3500 },
    { text: 'sent you the keys. they self-wipe after you read them once', outgoing: false, timestamp: now - 3400 },
    { text: 'got it. fingerprint matches what you posted', outgoing: true, timestamp: now - 3300 },
    { text: 'perfect. burn this thread when you’re done', outgoing: false, timestamp: now - 120 },
  ],
  'ghostwire': [
    { text: 'rotate the passphrase tonight, not tomorrow', outgoing: false, timestamp: now - 7200 },
    { text: 'why the rush?', outgoing: true, timestamp: now - 7100 },
    { text: 'saw a probe on the relay. nothing got through but still', outgoing: false, timestamp: now - 7000 },
    { text: 'on it. new one in 10', outgoing: true, timestamp: now - 200 },
  ],
  'len': [
    { text: 'ok talk tomorrow', outgoing: false, timestamp: now - 90000 },
  ],
  'oksana': [
    { text: 'meeting moved to the safehouse', outgoing: false, timestamp: now - 180000 },
    { text: 'noted', outgoing: true, timestamp: now - 179000 },
  ],
  'r0ot': [
    { text: 'verify my fingerprint when you can', outgoing: false, timestamp: now - 200000 },
  ],
  'priya.dev': [
    { text: 'pushed the signed build', outgoing: false, timestamp: now - 300000 },
    { text: 'pulling now, thanks', outgoing: true, timestamp: now - 299000 },
  ],
}

const AUTO_REPLIES = [
  'copy that.', 'on it.', 'understood — staying dark for a bit.', 'fingerprint checks out.',
  'give me two minutes.', 'channel still clean on my end.', 'noted. burning this after.',
]

function fmtTime(ts) {
  const d = new Date(ts * 1000)
  const n = new Date()
  if (d.toDateString() === n.toDateString()) return d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  return d.toLocaleDateString([], { month: 'short', day: 'numeric' }) + ' ' + d.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

Object.assign(window, {
  hashStr, Identicon,
  IconPlus, IconClose, IconEye, IconEyeOff, IconSend, IconLock, IconSun, IconMoon,
  IconCheck, IconCheckCheck, IconSearch, IconShield, IconDots, IconArrowLeft,
  SAMPLE_CONTACTS, SAMPLE_THREADS, AUTO_REPLIES, fmtTime,
})
