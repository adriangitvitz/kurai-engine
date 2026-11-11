import type { NextConfig } from "next";

const securityHeaders = [
  {
    key: 'Referrer-Policy',
    value: 'no-referrer',
  },
  {
    key: 'X-Content-Type-Options',
    value: 'nosniff',
  },
  {
    key: 'X-Frame-Options',
    value: 'DENY',
  },
  {
    key: 'Permissions-Policy',
    value: 'camera=(), microphone=(), geolocation=(), interest-cohort=()',
  },
  {
    key: 'Content-Security-Policy',
    value: [
      "default-src 'self'",
      "script-src 'self' 'unsafe-inline' 'unsafe-eval'",
      "style-src 'self' 'unsafe-inline'",
      "img-src 'self' data: https:",
      "font-src 'self'",
      "connect-src 'self'",
      "frame-ancestors 'none'",
    ].join('; '),
  },
];

const nextConfig: NextConfig = {
  output: 'standalone',

  async headers() {
    return [
      {
        source: '/:path*',
        headers: securityHeaders,
      },
    ];
  },

  async rewrites() {
    const backendUrl = process.env.KURAI_BACKEND_URL || 'http://localhost:8080'
    return [
      {
        source: '/api/:path*',
        destination: `${backendUrl}/:path*`,
      },
    ]
  },
};

export default nextConfig;
