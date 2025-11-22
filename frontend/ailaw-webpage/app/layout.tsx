import type React from "react"
import type { Metadata } from "next"
import { GeistSans } from "geist/font/sans"
import { GeistMono } from "geist/font/mono"
import { Suspense } from "react"
import "./globals.css"
import { AuthProvider } from "./providers"
import { PromptProvider } from "../components/prompt-context"

export const metadata: Metadata = {
  title: "AI Chatbot Interface",
  description: "Modern chatbot interface similar to ChatGPT and Claude",
  generator: "v0.app",
}

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode
}>) {
  return (
    <html lang="en">
      <body className={`font-sans ${GeistSans.variable} ${GeistMono.variable}`}>
        {/* <div className="fixed inset-0 -z-10 bg-cover bg-center bg-no-repeat bg-[url('/bg.jpg')] opacity-30"></div> */}
        <AuthProvider>
          <PromptProvider>
            <Suspense fallback={null}>{children}</Suspense>
          </PromptProvider>
        </AuthProvider>
      </body>
    </html>
  )
}
