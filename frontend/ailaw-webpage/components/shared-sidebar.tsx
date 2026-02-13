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

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL

export function SharedSidebar({ isDark, onToggleTheme, currentChatId, isOpen, onToggle }: SharedSidebarProps) {
  const router = useRouter()
  const [chats, setChats] = useState<Chat[]>([])
  const [openMenuId, setOpenMenuId] = useState<string | null>(null)
  const [deleteSession, setDeleteSession] = useState<{ isOpen: boolean; sessionId: string; title: string } | null>(null)
  const [renameSession, setRenameSession] = useState<{ isOpen: boolean; sessionId: string; currentTitle: string } | null>(null)
  const [renameValue, setRenameValue] = useState("")
  const { accessToken, logout, refreshToken, getToken } = useAuth()
  const isRefreshingRef = useRef(false)
  const initializedRef = useRef(false)
  const menuRef = useRef<HTMLDivElement>(null)
  const renameInputRef = useRef<HTMLInputElement>(null)

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
  }, [loadChats, accessToken])

  useEffect(() => {
    if (initializedRef.current && currentChatId) {
      loadChats()
    }
  }, [currentChatId, loadChats])

  // Close menu when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (menuRef.current && !menuRef.current.contains(event.target as Node)) {
        setOpenMenuId(null)
      }
    }

    if (openMenuId) {
      document.addEventListener("mousedown", handleClickOutside)
      return () => document.removeEventListener("mousedown", handleClickOutside)
    }
  }, [openMenuId])

  // Focus input when renaming
  useEffect(() => {
    if (renameSession?.isOpen && renameInputRef.current) {
      renameInputRef.current.focus()
      renameInputRef.current.select()
    }
  }, [renameSession?.isOpen])

  const handleDeleteSession = useCallback(async () => {
    if (!deleteSession) return
    
    try {
      await apiFetch(`/api/session/${deleteSession.sessionId}`, { method: "DELETE" })
      setChats(prev => prev.filter(chat => chat.sessionId !== deleteSession.sessionId))
      
      // If deleting current chat, redirect to welcome
      if (deleteSession.sessionId === currentChatId) {
        router.push("/welcome")
      }
    } catch (error) {
      console.error("Delete session error:", error)
    } finally {
      setDeleteSession(null)
      setOpenMenuId(null)
    }
  }, [apiFetch, deleteSession, currentChatId, router])

  const handleRenameSession = useCallback(async () => {
    if (!renameSession || !renameValue.trim()) return
    
    try {
      await apiFetch(`/api/name/session/${renameSession.sessionId}`, {
        method: "PATCH",
        body: JSON.stringify({ newName: renameValue.trim() })
      })
      
      setChats(prev => prev.map(chat => 
        chat.sessionId === renameSession.sessionId 
          ? { ...chat, title: renameValue.trim() }
          : chat
      ))
    } catch (error) {
      console.error("Rename session error:", error)
    } finally {
      setRenameSession(null)
      setRenameValue("")
      setOpenMenuId(null)
    }
  }, [apiFetch, renameSession, renameValue])

  const openDeleteSession = useCallback((sessionId: string, title: string) => {
    setDeleteSession({ isOpen: true, sessionId, title })
    setOpenMenuId(null)
  }, [])

  const openRenameSession = useCallback((sessionId: string, currentTitle: string) => {
    setRenameSession({ isOpen: true, sessionId, currentTitle })
    setRenameValue(currentTitle)
    setOpenMenuId(null)
  }, [])

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
            <Image
              src="/sidebar.svg"
              alt="Sidebar Icon"
              width={20}
              height={20}
              className={`${isDark ? "invert opacity-60" : "opacity-80"}`}
            />
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
                <div
                  key={chat.sessionId}
                  className={`group relative flex items-center rounded-lg transition-colors ${
                    chat.sessionId === currentChatId
                      ? isDark
                        ? "bg-[#3C3C3C]"
                        : "bg-[#C7CDE4]"
                      : isDark
                        ? "hover:bg-[#3C3C3C]"
                        : "hover:bg-[#C7CDE4]"
                  }`}
                >
                  <button
                    onClick={() => {
                      router.push(`/chat/${chat.sessionId}`)
                      if (window.innerWidth < 768) onToggle()
                    }}
                    className={`flex flex-1 items-center gap-2 px-3 py-2 text-left text-sm transition-colors cursor-pointer min-w-0 ${
                      chat.sessionId === currentChatId
                        ? isDark
                          ? "text-white"
                          : "text-indigo-900"
                        : isDark
                          ? "text-slate-300"
                          : "text-slate-700"
                    }`}
                  >
                    <Image
                      src="/session-logo.svg"
                      alt="Chat Icon"
                      width={20}
                      height={20}
                      className={`flex-shrink-0 ${isDark ? "invert" : ""}`}
                    />
                    <span className="truncate">{chat.title}</span>
                  </button>

                  {/* Three Dots Menu Button */}
                  <div className="relative flex-shrink-0" ref={openMenuId === chat.sessionId ? menuRef : null}>
                    <button
                      onClick={(e) => {
                        e.stopPropagation()
                        setOpenMenuId(openMenuId === chat.sessionId ? null : chat.sessionId)
                      }}
                      className={`opacity-0 group-hover:opacity-100 mr-2 p-1.5 rounded transition-all ${
                        openMenuId === chat.sessionId ? "opacity-100" : ""
                      } ${
                        isDark 
                          ? "hover:bg-[#303030] text-slate-400 hover:text-white" 
                          : "hover:bg-[#B5BCD9] text-slate-600 hover:text-slate-900"
                      }`}
                    >
                      <svg className="h-4 w-4" fill="currentColor" viewBox="0 0 24 24">
                        <circle cx="12" cy="5" r="2"/>
                        <circle cx="12" cy="12" r="2"/>
                        <circle cx="12" cy="19" r="2"/>
                      </svg>
                    </button>

                    {/* Dropdown Menu */}
                    {openMenuId === chat.sessionId && (
                      <div 
                        className={`absolute right-0 top-8 z-50 w-40 rounded-lg shadow-lg border ${
                          isDark 
                            ? "bg-[#2A2A2A] border-[#3C3C3C]" 
                            : "bg-white border-gray-200"
                        }`}
                      >
                        <button
                          onClick={(e) => {
                            e.stopPropagation()
                            openRenameSession(chat.sessionId, chat.title)
                          }}
                          className={`flex w-full items-center gap-2 px-3 py-2 text-sm transition-colors ${
                            isDark 
                              ? "text-slate-300 hover:bg-[#3C3C3C]" 
                              : "text-slate-700 hover:bg-gray-100"
                          } rounded-t-lg`}
                        >
                          <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
                          </svg>
                          Rename
                        </button>
                        <button
                          onClick={(e) => {
                            e.stopPropagation()
                            openDeleteSession(chat.sessionId, chat.title)
                          }}
                          className={`flex w-full items-center gap-2 px-3 py-2 text-sm transition-colors ${
                            isDark 
                              ? "text-red-400 hover:bg-[#3C3C3C]" 
                              : "text-red-600 hover:bg-red-50"
                          } rounded-b-lg`}
                        >
                          <svg className="h-4 w-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
                          </svg>
                          Delete
                        </button>
                      </div>
                    )}
                  </div>
                </div>
              ))
            )}
          </div>
        </div>

        {/* Delete Confirmation Session */}
        {deleteSession?.isOpen && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
            <div 
              className="absolute inset-0 bg-black/50" 
              onClick={() => setDeleteSession(null)}
            />
            <div 
              className={`relative w-full max-w-md rounded-lg shadow-xl ${
                isDark ? "bg-[#2A2A2A]" : "bg-white"
              }`}
              onKeyDown={(e) => {
                if (e.key === "Enter") handleDeleteSession()
                if (e.key === "Escape") setDeleteSession(null)
              }}
            >
              <div className="p-6">
                <h3 className={`text-lg font-semibold mb-2 ${
                  isDark ? "text-white" : "text-slate-900"
                }`}>
                  Delete Chat
                </h3>
                <p className={`text-sm mb-4 ${
                  isDark ? "text-slate-300" : "text-slate-600"
                }`}>
                  Are you sure you want to delete "{deleteSession.title}"? This action cannot be undone.
                </p>
                <div className="flex gap-3 justify-end">
                  <button
                    onClick={() => setDeleteSession(null)}
                    className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                      isDark 
                        ? "bg-[#3C3C3C] text-white hover:bg-[#303030]" 
                        : "bg-gray-200 text-slate-900 hover:bg-gray-300"
                    }`}
                  >
                    Cancel
                  </button>
                  <button
                    onClick={handleDeleteSession}
                    className="px-4 py-2 bg-red-600 text-white rounded-lg text-sm font-medium hover:bg-red-700 transition-colors"
                  >
                    Delete
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}

        {/* Rename Session */}
        {renameSession?.isOpen && (
          <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
            <div 
              className="absolute inset-0 bg-black/50" 
              onClick={() => {
                setRenameSession(null)
                setRenameValue("")
              }}
            />
            <div 
              className={`relative w-full max-w-md rounded-lg shadow-xl ${
                isDark ? "bg-[#2A2A2A]" : "bg-white"
              }`}
            >
              <div className="p-6">
                <h3 className={`text-lg font-semibold mb-4 ${
                  isDark ? "text-white" : "text-slate-900"
                }`}>
                  Rename Chat
                </h3>
                <input
                  ref={renameInputRef}
                  type="text"
                  value={renameValue}
                  onChange={(e) => setRenameValue(e.target.value)}
                  onKeyDown={(e) => {
                    if (e.key === "Enter") handleRenameSession()
                    if (e.key === "Escape") {
                      setRenameSession(null)
                      setRenameValue("")
                    }
                  }}
                  placeholder="Enter new chat name"
                  className={`w-full px-4 py-2 rounded-lg border text-sm mb-4 ${
                    isDark 
                      ? "bg-[#3C3C3C] border-[#4C4C4C] text-white placeholder-slate-500" 
                      : "bg-white border-gray-300 text-slate-900 placeholder-slate-400"
                  } focus:outline-none focus:ring-2 focus:ring-blue-500`}
                />
                <div className="flex gap-3 justify-end">
                  <button
                    onClick={() => {
                      setRenameSession(null)
                      setRenameValue("")
                    }}
                    className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                      isDark 
                        ? "bg-[#3C3C3C] text-white hover:bg-[#303030]" 
                        : "bg-gray-200 text-slate-900 hover:bg-gray-300"
                    }`}
                  >
                    Cancel
                  </button>
                  <button
                    onClick={handleRenameSession}
                    disabled={!renameValue.trim()}
                    className={`px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
                      renameValue.trim()
                        ? "bg-blue-600 text-white hover:bg-blue-700"
                        : "bg-gray-400 text-gray-200 cursor-not-allowed"
                    }`}
                  >
                    Rename
                  </button>
                </div>
              </div>
            </div>
          </div>
        )}

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