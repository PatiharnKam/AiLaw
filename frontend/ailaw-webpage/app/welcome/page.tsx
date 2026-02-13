"use client"

import type React from "react"
import Image from "next/image"
import { useEffect, useState, useRef, useCallback } from "react"
import { useRouter } from "next/navigation"
import { useModelType } from "@/hooks/useModelType"
import { useToast } from "@/hooks/useToast"
import { ToastContainer } from "@/components/toast"
import { parseApiError } from "@/utils/errorMapping"
import { SharedSidebar } from "@/components/shared-sidebar"
import { ChatInput } from "@/components/chat-input"
import { useAuth } from "../providers"
import { usePrompt } from "../../components/prompt-context"

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL

export default function WelcomePage() {
  const router = useRouter()
  const [prompt, setUserPrompt] = useState("")
  const [isDark, setIsDark] = useState(true)
  const [sidebarOpen, setSidebarOpen] = useState(true)
  const { modelType, setModelType } = useModelType()
  const { accessToken, logout, refreshToken, getToken } = useAuth()
  const { setPrompt } = usePrompt()
  const initializedRef = useRef(false)
  const isRefreshingRef = useRef(false)
  const { toasts, removeToast, showError } = useToast()

  useEffect(() => {
    const savedTheme = localStorage.getItem("chatbot-theme")
    if (savedTheme) {
      setIsDark(savedTheme === "dark")
    }

    const handleResize = () => {
      if (window.innerWidth < 768) {
        setSidebarOpen(false)
      }
    }
    handleResize()
    window.addEventListener("resize", handleResize)
    return () => window.removeEventListener("resize", handleResize)
  }, [])

  useEffect(() => {
    if (initializedRef.current) return
    initializedRef.current = true

    const initAuth = async () => {
      if (!accessToken) {
        const refreshed = await refreshToken()
        if (!refreshed) {
          logout()
        }
      }
    }

    initAuth()
  }, [])

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

    if (!res.ok) {
      throw data
    }
    
    return data
  }, [getToken, refreshToken, logout])

  const toggleTheme = () => {
    const newTheme = !isDark
    setIsDark(newTheme)
    localStorage.setItem("chatbot-theme", newTheme ? "dark" : "light")
  }

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!prompt.trim()) return

    try {
      const chatTitle = prompt.slice(0, 50)
      const data = await apiFetch("/api/session", {
        method: "POST",
        body: JSON.stringify({ title: chatTitle }),
      })

      const chatId = data.data
      setPrompt(prompt)
      router.push(`/chat/${chatId}`)
    } catch (error: any) {
      const errorInfo = parseApiError(error)
      showError(errorInfo.title, errorInfo.message)
      router.push("/welcome")
    }
  }

  return (
    <div className={isDark ? "dark" : ""}>
      {/* 🎯 Toast Container */}
      <ToastContainer toasts={toasts} onClose={removeToast} />
      
      <div
        className={`flex min-h-screen transition-all duration-300 ${
          !sidebarOpen ? 'bg-center' : 'bg-[center_right_-130px]'
        } ${
          isDark 
            ? "bg-cover bg-[url('/dark-bg.png')] text-white" 
            : "bg-cover bg-[url('/light-bg.png')] text-slate-900"
        }`}
      >
        <SharedSidebar
          isDark={isDark}
          onToggleTheme={toggleTheme}
          isOpen={sidebarOpen}
          onToggle={() => setSidebarOpen(!sidebarOpen)}
        />

        {!sidebarOpen && (
          <button
            onClick={() => setSidebarOpen(true)}
            className={`fixed top-4 left-4 z-50 flex h-10 w-10 items-center justify-center rounded-lg transition-colors cursor-pointer ${
              isDark 
                ? "bg-[#3C3C3C] text-white hover:bg-[#3C3C3C]" 
                : "bg-white text-slate-900 hover:bg-[#C7CDE4] shadow-lg"
            }`}
          >
            <svg className="h-6 w-6" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M4 6h16M4 12h16M4 18h16" />
            </svg>
          </button>
        )}

        <main className={`flex flex-1 flex-col items-center justify-center p-4 md:p-8 transition-all duration-300 ${
          !sidebarOpen ? 'mx-auto' : ''
        }`}>
          <div className="w-full max-w-3xl space-y-12">
            <div className="text-center space-y-4">
              <h1 className={`text-4xl md:text-5xl font-bold text-balance ${
                isDark ? "text-[#E5E5E5]" : "text-slate-900"
              }`}>
                สวัสดี! วันนี้ให้ฉันช่วยอะไรคุณ?
              </h1>
              <p className={`text-lg ${isDark ? "text-slate-400" : "text-slate-600"}`}>
                ลองถามคำถามฉันสิ ฉันจะช่วยตรวจสอบกฎหมายให้
              </p>
            </div>

            <div className="grid grid-cols-1 gap-4 md:grid-cols-3 text-center">
              <div className="rounded-xl p-6 space-y-2 transition-colors flex flex-col items-center justify-center text-center">
                <Image
                  src="/search-logo.svg"
                  width={24}
                  height={24}
                  alt="icon"
                  className={isDark ? "invert" : ""}
                />
                <h3 className={`font-semibold ${isDark ? "text-white" : "text-slate-900"}`}>
                  ค้นหามาตรา
                </h3>
                <p className={`text-sm ${isDark ? "text-slate-400" : "text-slate-600"}`}>
                  มาตราของประมวลกฎหมายแพ่งและพาณิชย์ทั้งสิ้น 1755 มาตรา
                </p>
              </div>

              <div className="rounded-xl p-6 space-y-2 transition-colors flex flex-col items-center justify-center text-center">
                <Image
                  src="/star-1.svg"
                  width={24}
                  height={24}
                  alt="icon"
                  className={isDark ? "invert" : ""}
                />
                <h3 className={`font-semibold ${isDark ? "text-white" : "text-slate-900"}`}>
                  คิดวิเคราะห์
                </h3>
                <p className={`text-sm ${isDark ? "text-slate-400" : "text-slate-600"}`}>
                  มีกระบวนการคิดวิเคราะห์อย่างเป็นลำดับขั้นตอน
                </p>
              </div>

              <div className="rounded-xl p-6 space-y-2 transition-colors flex flex-col items-center justify-center text-center">
                <Image
                  src="/target.svg"
                  width={24}
                  height={24}
                  alt="icon"
                  className={isDark ? "invert" : ""}
                />
                <h3 className={`font-semibold ${isDark ? "text-white" : "text-slate-900"}`}>
                  ตอบตรงประเด็น
                </h3>
                <p className={`text-sm ${isDark ? "text-slate-400" : "text-slate-600"}`}>
                  ตอบคำถามอย่างกระชับพร้อมอ้างอิง มาตราที่เกี่ยวข้อง
                </p>
              </div>
            </div>

            <div className="pt-[30px]">
              <ChatInput
                input={prompt}
                setInput={setUserPrompt}
                handleSubmit={handleSubmit}
                isDark={isDark}
                modelType={modelType}
                setModelType={setModelType}
                placeholder="Type your message here..."
              />
            </div>
          </div>
        </main>
      </div>
    </div>
  )
}