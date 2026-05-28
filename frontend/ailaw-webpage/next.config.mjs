/** @type {import('next').NextConfig} */
const nextConfig = {
  // use standalone when build in Docker ONLY
  ...(process.env.DOCKER_BUILD === 'true' && { output: 'standalone' }),
  typescript: {
    ignoreBuildErrors: true,
  },
  images: {
    unoptimized: true,
  },
  reactStrictMode: false,
}

export default nextConfig