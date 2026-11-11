'use client'

import { useEffect, useRef } from 'react'
import Markdown from 'react-markdown'
import remarkGfm from 'remark-gfm'
import { ArrowRight } from './icons'
import type { ReaderState } from '@/lib/useFolioSearch'

const ERROR_TEXT: Record<string, string> = {
  extraction_failed: 'This page has no readable article content.',
  unsupported_content_type: 'This kind of file can’t be read here.',
  fetch_timeout: 'The page took too long to respond. Try again in a moment.',
  upstream_unavailable: 'The page couldn’t be reached. Try again in a moment.',
  blocked_domain: 'This site is on the blocklist.',
  invalid_url: 'That doesn’t look like a valid link.',
  response_too_large: 'This page is too large to read here.',
}

const META_FONT = "var(--fw-meta) var(--fs-meta) 'IBM Plex Mono', monospace"

interface PageEntryProps {
  id: number
  reader: ReaderState
  focusHeading: boolean
}

export function PageEntry({ id, reader, focusHeading }: PageEntryProps) {
  const headingRef = useRef<HTMLHeadingElement>(null)
  const focusedOnce = useRef(false)
  useEffect(() => {
    if (focusHeading && !focusedOnce.current) {
      focusedOnce.current = true
      headingRef.current?.focus()
    }
  }, [focusHeading])

  const { article } = reader
  const meta = [article?.site_name, article?.byline, article?.word_count ? `${article.word_count} words` : '']
    .filter(Boolean)
    .join(' · ')

  return (
    <article data-entry-id={id} data-page-entry style={{ marginBottom: 56, animation: 'folio-rise .5s ease both' }}>
      <div
        style={{
          font: META_FONT,
          letterSpacing: '.14em',
          textTransform: 'uppercase',
          color: 'var(--accent)',
          marginBottom: 10,
        }}
      >
        Page
      </div>
      <h1
        ref={headingRef}
        tabIndex={-1}
        style={{
          fontFamily: "'Newsreader', serif",
          fontWeight: 500,
          fontSize: 34,
          lineHeight: 1.18,
          letterSpacing: '-0.015em',
          margin: '0 0 12px',
          outline: 'none',
          overflowWrap: 'anywhere',
        }}
      >
        {article?.title || reader.url}
      </h1>
      <a
        href={reader.url}
        target="_blank"
        rel="noopener noreferrer"
        data-page-url
        style={{
          display: 'inline-flex',
          alignItems: 'center',
          gap: 6,
          font: META_FONT,
          color: 'var(--accent)',
          textDecoration: 'none',
          overflowWrap: 'anywhere',
        }}
      >
        {reader.url}
        <ArrowRight size={12} color="currentColor" />
      </a>
      {meta && <div style={{ font: META_FONT, color: 'var(--muted)', marginTop: 8 }}>{meta}</div>}

      {reader.status === 'loading' && (
        <div style={{ marginTop: 28, maxWidth: 760 }}>
          <div className="folio-skel" style={{ height: 18, width: '92%', marginBottom: 12 }} />
          <div className="folio-skel" style={{ height: 18, width: '84%', marginBottom: 12 }} />
          <div className="folio-skel" style={{ height: 18, width: '70%' }} />
        </div>
      )}

      {reader.status === 'error' && (
        <div
          role="alert"
          style={{
            marginTop: 28,
            padding: '18px 22px',
            borderRadius: 16,
            background: 'var(--surface)',
            boxShadow: 'var(--raise-sm)',
            fontSize: 'var(--fs-body)',
          }}
        >
          {ERROR_TEXT[reader.errorCode ?? ''] ?? 'Couldn’t read this page.'}
        </div>
      )}

      {reader.status === 'ready' && article?.content_markdown && (
        <div className="folio-prose">
          <Markdown
            remarkPlugins={[remarkGfm]}
            components={{
              h1: 'h2',
              a: ({ href, children }) => (
                <a href={href} target="_blank" rel="noopener noreferrer">
                  {children}
                </a>
              ),
              img: () => null,
            }}
          >
            {article.content_markdown}
          </Markdown>
        </div>
      )}
    </article>
  )
}
