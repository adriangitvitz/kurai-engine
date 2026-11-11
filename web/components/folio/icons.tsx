import type { CSSProperties } from 'react'

interface IconProps {
  size?: number
  style?: CSSProperties
}

export function SearchIcon({ size = 18, style }: IconProps) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor"
      strokeWidth="2" strokeLinecap="round" style={style}>
      <circle cx="11" cy="11" r="7" />
      <path d="m20 20-3.2-3.2" />
    </svg>
  )
}

export function ArrowRight({ size = 20, color = '#161616' }: IconProps & { color?: string }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={color}
      strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M5 12h13M13 6l6 6-6 6" />
    </svg>
  )
}

export function ArrowUp({ size = 19, color = '#161616' }: IconProps & { color?: string }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke={color}
      strokeWidth="2.3" strokeLinecap="round" strokeLinejoin="round">
      <path d="M12 19V5M6 11l6-6 6 6" />
    </svg>
  )
}

export function Spark({ size = 12, fill = 'currentColor' }: IconProps & { fill?: string }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill={fill}>
      <path d="M12 2l1.8 6.2L20 10l-6.2 1.8L12 18l-1.8-6.2L4 10l6.2-1.8z" />
    </svg>
  )
}

export function Bookmark({ size = 14, fill = 'none', stroke = 'currentColor' }: IconProps & { fill?: string; stroke?: string }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill={fill} stroke={stroke}
      strokeWidth="2" strokeLinejoin="round" strokeLinecap="round">
      <path d="M6 4h12v16l-6-3.6L6 20z" />
    </svg>
  )
}

export function Close({ size = 16 }: IconProps) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor"
      strokeWidth="2.2" strokeLinecap="round">
      <path d="M6 6l12 12M18 6L6 18" />
    </svg>
  )
}

export function BookOpen({ size = 12 }: IconProps) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor"
      strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <path d="M4 5.5A2.5 2.5 0 0 1 6.5 3H19v15H6.5A2.5 2.5 0 0 0 4 20.5z" />
      <path d="M4 20.5V5.5" />
    </svg>
  )
}

export function ChevronDown({ size = 14 }: IconProps) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor"
      strokeWidth="2.4" strokeLinecap="round" strokeLinejoin="round">
      <path d="M6 9l6 6 6-6" />
    </svg>
  )
}

export function Cards({ size = 16, slashed = false }: IconProps & { slashed?: boolean }) {
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor"
      strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
      <rect x="4" y="7" width="16" height="13" rx="2.5" />
      <path d="M7 4h10" />
      {slashed && <path d="M3.5 21.5 20.5 3.5" strokeWidth="2.2" />}
    </svg>
  )
}
