"use client"

import React, { createContext, useState, useContext, ReactNode, useRef, useEffect } from "react"
import { useRouter } from "next/navigation"

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL

interface AuthContextType {
  accessToken: string | null
  setAccessToken: (token: string | null) => void
  logout: () => void
  refreshToken: () => Promise<boolean>
  getToken: () => string | null
}

const AuthContext = createContext<AuthContextType | undefined>(undefined)

export const AuthProvider = ({ children }: { children: ReactNode }) => {
  const [accessToken, setAccessTokenState] = useState<string | null>(null)
  const accessTokenRef = useRef<string | null>(null)
  const router = useRouter()

  // wrapped setter
  const setAccessToken = (token: string | null) => {
    accessTokenRef.current = token
    setAccessTokenState(token)
  }

  const getToken = () => accessTokenRef.current

  const logout = async () => {
    try {
      await fetch(`${API_BASE_URL}/auth/logout`, {
        method: "POST",
        credentials: "include",
        headers: {
          "Authorization": `Bearer ${accessTokenRef.current}`
        }
      })
    } catch (error) {
      console.error("Logout error:", error)
    } finally {
      setAccessToken(null)
      router.push("/login")
    }
  }

  const refreshToken = async (): Promise<boolean> => {
    try {
      const res = await fetch(`${API_BASE_URL}/auth/refresh`, {
        method: "POST",
        credentials: "include",
      })

      const data = await res.json()

      if (!res.ok) return false

      const newAccessToken = data?.data?.accessToken
      if (!newAccessToken) throw new Error("Access token not found")

      accessTokenRef.current = newAccessToken
      setAccessTokenState(newAccessToken)

      return true
    } catch (error) {

      return false
    }
  }

  return (
    <AuthContext.Provider
      value={{
        accessToken,
        setAccessToken,
        logout,
        refreshToken,
        getToken,
      }}
    >
      {children}
    </AuthContext.Provider>
  )
}

export const useAuth = () => {
  const context = useContext(AuthContext)
  if (!context) throw new Error("useAuth must be used within an AuthProvider")
  return context
}

