'use client'

import type { ReactNode } from 'react'
import { ProvenanceEyebrow } from './ProvenanceEyebrow'

interface AnswerCardShellProps {
  answerClass: string
  subject: string
  eyebrowLabel: string
  source: string
  sourceHref?: string
  children: ReactNode
}

export function AnswerCardShell({
  answerClass,
  subject,
  eyebrowLabel,
  source,
  sourceHref,
  children,
}: AnswerCardShellProps) {
  return (
    <section
      role="region"
      aria-label={`${answerClass}: ${subject}`}
      style={{
        background: 'var(--surface)',
        boxShadow: 'var(--raise)',
        borderRadius: 18,
        padding: '20px 24px',
        marginBottom: 28,
      }}
    >
      <ProvenanceEyebrow label={eyebrowLabel} source={source} sourceHref={sourceHref} />
      {children}
    </section>
  )
}
