/** @type {import('next').NextConfig} */
const nextConfig = {
  async rewrites() {
    return [
      {
        source: '/api/:path*',
        destination: 'http://inventory-api:8080/api/:path*',
      },
      {
        source: '/ws',
        destination: 'http://inventory-api:8080/ws',
      },
    ];
  },
};

module.exports = nextConfig;