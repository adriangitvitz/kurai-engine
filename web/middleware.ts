import { NextResponse } from 'next/server'
import type { NextRequest } from 'next/server'
import { safeNext } from '@/lib/nav'

const SESSION_COOKIE = 'kurai_session'

export function middleware(request: NextRequest) {
  if (process.env.NEXT_PUBLIC_AUTH_ENABLED !== 'true') {
    return NextResponse.next()
  }

  const { pathname, search } = request.nextUrl

  if (pathname === '/login') {
    if (request.cookies.has(SESSION_COOKIE)) {
      return NextResponse.redirect(new URL(safeNext(request.nextUrl.searchParams.get('next')), request.url))
    }
    return NextResponse.next()
  }

  if (!request.cookies.has(SESSION_COOKIE)) {
    const login = new URL('/login', request.url)
    if (pathname + search !== '/') login.searchParams.set('next', pathname + search)
    return NextResponse.redirect(login)
  }

  return NextResponse.next()
}

export const config = {
  matcher: ['/((?!api|_next/static|_next/image|fonts|favicon.ico).*)'],
}
