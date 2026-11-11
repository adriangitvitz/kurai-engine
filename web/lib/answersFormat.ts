const EN = 'en-US'

export function formatAmount(value: number, minorUnits: number): string {
  return new Intl.NumberFormat(EN, {
    minimumFractionDigits: minorUnits,
    maximumFractionDigits: minorUnits,
  }).format(value)
}

export function formatRate(rate: number): string {
  return new Intl.NumberFormat(EN, { maximumSignificantDigits: 6 }).format(rate)
}
