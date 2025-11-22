"use client"

import Image from "next/image"
import { useState, useEffect, useRef, useCallback } from "react"
import { useRouter } from "next/navigation"
import { Button } from "@/components/ui/button"
import { useAuth } from "../app/providers"

interface Chat {
  sessionId: string
  title: string
  createdAt: string
  lastMessageAt: string
}

interface SharedSidebarProps {
  isDark: boolean
  onToggleTheme: () => void
  currentChatId?: string
  isOpen: boolean
  onToggle: () => void
}

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080"

export function SharedSidebar({ isDark, onToggleTheme, currentChatId, isOpen, onToggle }: SharedSidebarProps) {
  const router = useRouter()
  const [chats, setChats] = useState<Chat[]>([])
  const { accessToken, logout, refreshToken, getToken } = useAuth()
  const isRefreshingRef = useRef(false)
  const initializedRef = useRef(false)

  const apiFetch = useCallback(async (path: string, options?: RequestInit) => {
    if (isRefreshingRef.current) {
      await new Promise(resolve => setTimeout(resolve, 100))
      return apiFetch(path, options)
    }

    const currentToken = getToken()

    const res = await fetch(`${API_BASE_URL}${path}`, {
      credentials: "include",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${currentToken}`,
        ...(options?.headers || {}),
      },
      ...options,
    })

    const data = await res.json()

    if (data?.data?.action === "logout") {
      logout()
      throw new Error("Logged out")
    }
    
    if (data?.data?.action === "refresh") {
      if (isRefreshingRef.current) {
        await new Promise(resolve => setTimeout(resolve, 100))
        return apiFetch(path, options)
      }

      isRefreshingRef.current = true
      const refreshed = await refreshToken()
      isRefreshingRef.current = false

      if (refreshed) {
        return apiFetch(path, options)
      } else {
        logout()
        throw new Error("Refresh failed")
      }
    }

    if (!res.ok) throw new Error(data.message || "API request failed")
    return data
  }, [getToken, refreshToken, logout])

  const loadChats = useCallback(async () => {
    try {
      const data = await apiFetch("/api/sessions-history", { method: "GET" })
      setChats(data.data || [])
    } catch (error) {
      console.error("Failed to load chat history:", error)
      setChats([])
    }
  }, [apiFetch])

  useEffect(() => {
    if (initializedRef.current) return
    if (!accessToken) return
    
    initializedRef.current = true
    loadChats()
  }, [loadChats])

  useEffect(() => {
    if (initializedRef.current && currentChatId) {
      loadChats()
    }
  }, [currentChatId, loadChats])

  const handleLogout = useCallback(async () => {
    try {
      logout()
    } catch (error) {
      console.error("Logout error:", error)
    }
  }, [logout])

  return (
    <>
      {isOpen && (
        <div 
          className="fixed inset-0 z-30 bg-black/50 md:hidden" 
          onClick={onToggle} 
        />
      )}

      <aside
        className={`relative z-40 flex h-screen flex-col transition-all duration-300 ${
          isOpen ? "w-64" : "w-0 md:w-0"
        } ${
          isDark 
            ? "border-r-2 border-[#FFFFFF]/10 bg-[#222222]" 
            : "bg-white shadow-lg"
        } overflow-hidden`}
      >
        {/* Header */}
        <div className="flex items-center justify-between gap-3 p-4">
          <div className="flex items-center gap-1 min-w-0">
            <div className="flex h-16 w-16 flex-shrink-0 items-center justify-center rounded-lg">
              <Image
                src="/AiLaw.png"
                alt="Model Icon"
                width={100}
                height={100}
                className="rounded-full"
              />
            </div>
            <span className={`text-3xl font-bold whitespace-nowrap ${
              isDark ? "text-white" : "text-slate-900"
            }`}>
              AILaw
            </span>
          </div>
          <button
            onClick={onToggle}
            className={`flex h-8 w-8 flex-shrink-0 items-center justify-center rounded-lg transition-colors cursor-pointer ${
              isDark 
                ? "text-slate-400 hover:bg-[#3C3C3C] hover:text-white" 
                : "text-slate-600 hover:bg-[#C7CDE4] hover:text-slate-900"
            }`}
          >
            <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* New Chat Button */}
        <div className="p-3">
          <Button
            onClick={() => {
              router.push("/welcome")
              if (window.innerWidth < 768) onToggle()
            }}
            className={`w-full justify-start gap-3 cursor-pointer ${
              isDark 
                ? "bg-[#3C3C3C] text-white hover:bg-[#303030]" 
                : "bg-[#485BA9] hover:bg-[#4054A6] text-white"
            }`}
          >
            <Image
              src="/newchat-logo.svg"
              alt="New Chat Icon"
              width={20}
              height={20}
              className="invert"
            />
            New Chat
          </Button>
        </div>

        {/* Chat List */}
        <div className="custom-scroll flex-1 overflow-y-auto p-3">
          <div className="space-y-1">
            {chats.length === 0 ? (
              <p className={`text-center text-sm py-4 ${
                isDark ? "text-slate-500" : "text-slate-400"
              }`}>
                No chat history yet
              </p>
            ) : (
              chats.map((chat) => (
                <button
                  key={chat.sessionId}
                  onClick={() => {
                    router.push(`/chat/${chat.sessionId}`)
                    if (window.innerWidth < 768) onToggle()
                  }}
                  className={`flex w-full items-center gap-2 rounded-lg px-3 py-2 text-left text-sm transition-colors cursor-pointer ${
                    chat.sessionId === currentChatId
                      ? isDark
                        ? "bg-[#3C3C3C] text-white"
                        : "bg-[#C7CDE4] text-indigo-900"
                      : isDark
                        ? "text-slate-300 hover:bg-[#3C3C3C]"
                        : "text-slate-700 hover:bg-[#C7CDE4]"
                  }`}
                >
                  <Image
                    src="/session-logo.svg"
                    alt="Chat Icon"
                    width={20}
                    height={20}
                    className={isDark ? "invert" : ""}
                  />
                  <span className="truncate">{chat.title}</span>
                </button>
              ))
            )}
          </div>
        </div>

        {/* Footer */}
        <div className="p-3 space-y-2">
          <button
            onClick={onToggleTheme}
            className={`flex w-full items-center gap-3 rounded-lg px-3 py-2 text-sm transition-colors cursor-pointer ${
              isDark 
                ? "text-slate-300 hover:bg-[#3C3C3C]" 
                : "text-slate-700 hover:bg-[#C7CDE4]"
            }`}
          >
            <Image
              src={isDark ? "/moon.svg" : "/sun.svg"}
              alt="Theme Icon"
              width={20}
              height={20}
              className={isDark ? "invert" : ""}
            />
            Switch {isDark ? "Light" : "Dark"} Mode
          </button>
          
          <button
            onClick={handleLogout}
            className={`flex w-full items-center gap-3 rounded-lg px-3 py-2 text-sm transition-colors cursor-pointer ${
              isDark 
                ? "text-red-400 hover:bg-[#3C3C3C]" 
                : "text-red-600 hover:bg-red-50"
            }`}
          >
            <Image
              src="/logout.png"
              alt="Logout Icon"
              width={20}
              height={20}
            />
            Log out
          </button>
        </div>
      </aside>
    </>
  )
}
