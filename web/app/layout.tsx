import type { Metadata, Viewport } from 'next'
import './globals.css'

export const metadata: Metadata = {
  title: 'Kurai · a calmer way to search the web',
  description:
    'Kurai reads the web, ranks the strongest sources, and can synthesize one clear answer, a calmer way to search.',
}

export const viewport: Viewport = {
  themeColor: '#171716',
}

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode
}>) {
  return (
    <html lang="en">
      <body
        style={{
          background: 'var(--bg)',
          color: 'var(--text)',
          fontFamily: "'Hanken Grotesk', system-ui, sans-serif",
        }}
      >
        {children}
      </body>
    </html>
  )
}
