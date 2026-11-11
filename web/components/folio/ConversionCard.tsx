'use client'

import { AnswerCardShell } from './AnswerCardShell'
import { formatAmount, formatRate } from '@/lib/answersFormat'
import type { CurrencyAnswer } from '@/lib/types'

export function ConversionCard({ currency }: { currency: CurrencyAnswer }) {
  const converted = formatAmount(currency.converted, currency.minor_units)
  const amount = formatAmount(currency.amount, currency.minor_units)

  return (
    <AnswerCardShell
      answerClass="Currency conversion"
      subject={`${currency.base} to ${currency.quote}`}
      eyebrowLabel="Conversion"
      source={currency.source}
    >
      <div
        style={{
          fontFamily: "'Newsreader', serif",
          fontWeight: 500,
          fontSize: 30,
          lineHeight: 1.15,
          letterSpacing: '-0.01em',
          marginBottom: 10,
        }}
      >
        {converted} {currency.quote}
      </div>
      <div style={{ fontSize: 'var(--fs-body)', marginBottom: 12 }}>
        {amount} {currency.base} → {currency.quote} · rate {formatRate(currency.rate)}
      </div>
      <div
        data-asof={currency.rate_date}
        style={{
          font: "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace",
          letterSpacing: '.08em',
          textTransform: 'uppercase',
          color: 'var(--muted)',
        }}
      >
        Rate as of {currency.rate_date}
      </div>
    </AnswerCardShell>
  )
}
