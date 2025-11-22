"use client"

import React, { useState, useEffect } from "react"
import { useRouter, useSearchParams } from "next/navigation"
import { useAuth } from "../providers"

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080"

export default function LoginPage() {
  const router = useRouter()
  const searchParams = useSearchParams()
  const { setAccessToken } = useAuth()

  const [email, setEmail] = useState("")
  const [isLoading, setIsLoading] = useState(false)
  const [error, setError] = useState("")

  // เมื่อ Google redirect กลับมาพร้อม code
  useEffect(() => {
    const code = searchParams.get("code")
    if (code) {
      console.log("OAuth code received:", code)
      exchangeCodeForToken(code)
    }
  }, [searchParams])

  // ส่ง code กลับไป backend เพื่อแลก accessToken
  const exchangeCodeForToken = async (code: string) => {
    try {
      setIsLoading(true)
      const res = await fetch(`${API_BASE_URL}/auth/google/callback?code=${code}`, {
        method: "GET",
        credentials: "include",
      })

      if (!res.ok) {
        throw new Error("Failed to exchange code for token")
      }

      const data = await res.json()

      const accessToken = data?.data?.accessToken
      if (!accessToken) throw new Error("Access token not found")

      setAccessToken(accessToken)
      router.replace("/welcome")
    } catch (err: any) {
      setError(err.message || "Authentication failed")
      router.replace("/login")
    } finally {
      setIsLoading(false)
    }
  }

  // กด Continue → เริ่ม flow Google OAuth
  const handleContinue = (e: React.FormEvent) => {
    e.preventDefault()
    if (email) handleGoogleLogin(email)
  }

  // เปิดหน้า login ของ Google (เริ่ม OAuth)
  const handleGoogleLogin = (emailHint?: string) => {
    setIsLoading(true)
    const url = emailHint
      ? `${API_BASE_URL}/auth/google/login?email=${encodeURIComponent(emailHint)}`
      : `${API_BASE_URL}/auth/google/login`

    window.location.href = url
  }

    return (
      <div className="flex min-h-screen items-center justify-center bg-gradient-to-br from-gray-50 to-gray-100">
        <div className="w-full max-w-md space-y-6 px-8">
          <div className="text-center">
            <h1 className="text-4xl font-bold text-black">Log in or sign up</h1>
          </div>

          {/* Error Message */}
          {error && (
            <div className="rounded-xl bg-red-50 border border-red-200 px-4 py-3 text-sm text-red-600">
              {error}
            </div>
          )}

          {/* Loading State */}
          {isLoading ? (
            <div className="flex flex-col items-center justify-center space-y-4 py-8">
              <div className="h-8 w-8 animate-spin rounded-full border-4 border-gray-300 border-t-black"></div>
              <p className="text-gray-600">Authenticating...</p>
            </div>
          ) : (
            <>
              <div className="space-y-4">
                <div>
                  <input
                    id="email"
                    type="email"
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    onKeyPress={(e) => {
                      if (e.key === "Enter" && email) {
                        handleContinue(e)
                      }
                    }}
                    className="w-full rounded-xl border border-gray-300 bg-white px-4 py-3 text-gray-900 placeholder-gray-400 focus:border-gray-400 focus:outline-none focus:ring-2 focus:ring-gray-200"
                    placeholder="Email"
                    disabled={isLoading}
                  />
                </div>

                <button
                  type="button"
                  onClick={handleContinue}
                  disabled={isLoading || !email}
                  className="w-full rounded-xl bg-black px-4 py-3 font-semibold text-white transition-colors hover:bg-gray-800 disabled:opacity-50 disabled:cursor-not-allowed"
                >
                  Continue
                </button>
              </div>

              <div className="flex items-center gap-4">
                <div className="h-px flex-1 bg-gray-300"></div>
                <span className="text-sm text-gray-500">or</span>
                <div className="h-px flex-1 bg-gray-300"></div>
              </div>

              <button
                onClick={() => handleGoogleLogin()}
                disabled={isLoading}
                className="flex w-full items-center justify-center gap-3 rounded-xl border border-gray-300 bg-white px-4 py-3 font-medium text-gray-700 transition-colors hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed"
              >
                <svg className="h-5 w-5" viewBox="0 0 24 24">
                  <path
                    fill="#4285F4"
                    d="M22.56 12.25c0-.78-.07-1.53-.2-2.25H12v4.26h5.92c-.26 1.37-1.04 2.53-2.21 3.31v2.77h3.57c2.08-1.92 3.28-4.74 3.28-8.09z"
                  />
                  <path
                    fill="#34A853"
                    d="M12 23c2.97 0 5.46-.98 7.28-2.66l-3.57-2.77c-.98.66-2.23 1.06-3.71 1.06-2.86 0-5.29-1.93-6.16-4.53H2.18v2.84C3.99 20.53 7.7 23 12 23z"
                  />
                  <path
                    fill="#FBBC05"
                    d="M5.84 14.09c-.22-.66-.35-1.36-.35-2.09s.13-1.43.35-2.09V7.07H2.18C1.43 8.55 1 10.22 1 12s.43 3.45 1.18 4.93l2.85-2.22.81-.62z"
                  />
                  <path
                    fill="#EA4335"
                    d="M12 5.38c1.62 0 3.06.56 4.21 1.64l3.15-3.15C17.45 2.09 14.97 1 12 1 7.7 1 3.99 3.47 2.18 7.07l3.66 2.84c.87-2.6 3.3-4.53 6.16-4.53z"
                  />
                </svg>
                Continue with google
              </button>
            </>
          )}

          <div className="text-center">
            <p className="text-sm text-gray-500">
              <a href="#" className="hover:underline">
                Terms of Service
              </a>
              {" | "}
              <a href="#" className="hover:underline">
                Privacy Policy
              </a>
            </p>
          </div>
        </div>
      </div>
    )
  }