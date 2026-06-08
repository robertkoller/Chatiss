import React from 'react'

function hashStr(s: string): number {
  let h = 2166136261
  for (let i = 0; i < s.length; i++) {
    h ^= s.charCodeAt(i)
    h = Math.imul(h, 16777619)
  }
  return h >>> 0
}

interface IdenticonProps {
  name: string
  size?: number
  mono?: number | null
}

export default function Identicon({ name, size = 30, mono = null }: IdenticonProps) {
  const h = hashStr(name || '?')
  const hue = h % 360
  const hue2 = (hue + 38) % 360
  const cell = size / 5
  const bits: number[] = []
  let x = h
  for (let i = 0; i < 15; i++) {
    bits.push(x & 1)
    x = Math.imul(x ^ (x >>> 3), 2654435761) >>> 0
  }
  const fillFor = (two: boolean): string => {
    if (mono != null) {
      return `oklch(${two ? 0.82 : 0.6} 0.15 ${mono})`
    }
    return `oklch(0.7 0.13 ${two ? hue2 : hue})`
  }
  const rects: React.ReactElement[] = []
  let bi = 0
  for (let col = 0; col < 3; col++) {
    for (let row = 0; row < 5; row++) {
      const on = bits[bi++ % bits.length]
      if (!on) { continue }
      const cols = col === 2 ? [2] : [col, 4 - col]
      const two = (row + col) % 3 === 0
      cols.forEach(c => {
        rects.push(
          <rect key={`${c}-${row}-${col}`} x={c * cell} y={row * cell}
            width={cell + 0.4} height={cell + 0.4}
            rx={0} ry={0} fill={fillFor(two)} />
        )
      })
    }
  }
  const bg = mono != null ? `oklch(0.2 0.05 ${mono})` : `oklch(0.32 0.04 ${hue})`
  return (
    <svg width={size} height={size} viewBox={`0 0 ${size} ${size}`}
      style={{ display: 'block', overflow: 'hidden', flexShrink: 0 }}>
      <rect x={0} y={0} width={size} height={size} fill={bg} />
      {rects}
    </svg>
  )
}
