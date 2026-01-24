"use client"

import React from "react"
import Image from "next/image"
import { useEffect, useState, useRef, useCallback } from "react"
import { useRouter, useParams } from "next/navigation"
import { SharedSidebar } from "@/components/shared-sidebar"
import { ChatInput } from "@/components/chat-input"
import { usePrompt } from "@/components/prompt-context"
import { useAuth } from "../../providers"

interface Message {
  messageId: string
  role: "user" | "model"
  content: string
  createdAt: string
  feedback: number | null
}

interface Chat {
  sessionId: string
  title: string
  messages: Message[]
}

const API_BASE_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080"

export default function ChatPage() {
  const router = useRouter()
  const params = useParams()
  const chatId = params.id as string

  const [chat, setChat] = useState<Chat | null>(null)
  const [input, setInput] = useState("")
  const [isDark, setIsDark] = useState(true)
  const [sidebarOpen, setSidebarOpen] = useState(true)
  const [isSending, setIsSending] = useState(false)
  const [modelType, setModelType] = useState<"NORMAL" | "COT">("NORMAL")
  const { accessToken, logout, refreshToken, getToken } = useAuth()
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const { prompt, setPrompt } = usePrompt()
  const initializedRef = useRef(false)
  const isRefreshingRef = useRef(false)

  useEffect(() => {
    if (initializedRef.current) return
    initializedRef.current = true

    const initialize = async () => {
      // รอ token ถ้ายังไม่มี
      if (!accessToken) {
        const refreshed = await refreshToken()
        if (!refreshed) {
          logout()
          return
        }
      }

      await loadMessages()

      // ถ้ามี initial prompt ให้ส่งทันที
      if (prompt) {
        await sendMessage(String(prompt))
        setPrompt(null)
      }
    }

    initialize()
  }, [])

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: "smooth" })
  }, [chat?.messages])

  useEffect(() => {

    const savedTheme = localStorage.getItem("chatbot-theme")
    if (savedTheme) setIsDark(savedTheme === "dark")

    const handleResize = () => setSidebarOpen(window.innerWidth >= 768)
    handleResize()

    window.addEventListener("resize", handleResize)
    return () => window.removeEventListener("resize", handleResize)
  }, [])

  const appendMessage = useCallback((newMessages: Message[]) => {
    setChat(prev => ({
      sessionId: prev?.sessionId ?? chatId,
      title: prev?.title ?? "New Chat",
      messages: [...(prev?.messages ?? []), ...newMessages],
    }))
  }, [chatId])

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
      console.log("🚪 Logout action received")
      logout()
      throw new Error("Logged out")
    }
    
    if (data?.data?.action === "refresh") {
      if (isRefreshingRef.current) {
        await new Promise(resolve => setTimeout(resolve, 100))
        return apiFetch(path, options)
      }

      console.log("🔄 Refresh token needed")
      isRefreshingRef.current = true
      const refreshed = await refreshToken()
      isRefreshingRef.current = false

      if (refreshed) {
        return apiFetch(path, options)
      } else {
        console.log("❌ Refresh failed, logging out")
        logout()
        throw new Error("Refresh failed")
      }
    }

    if (!res.ok) throw new Error(data.message || "API request failed")
    return data
  }, [getToken, refreshToken, logout]) // ✅ เพิ่ม dependencies ครบ

  const loadMessages = useCallback(async () => {
    try {
      const data = await apiFetch(`/api/messages-history/${chatId}`, { method: "GET" })
      setChat({
        sessionId: chatId,
        title: data.data[0]?.content?.slice(0, 50) || "New Chat",
        messages: data.data || [],
      })
    } catch (error) {
      console.error("Failed to load messages:", error)
      router.push("/welcome")
    }
  }, [apiFetch, chatId, router])

  const sendMessage = useCallback(async (messageContent: string, role: "user" | "model" = "user") => {
    if (!messageContent.trim()) return

    const userMessage: Message = {
      messageId: crypto.randomUUID(),
      role,
      content: messageContent,
      createdAt: new Date().toISOString(),
      feedback: null,
    }

    appendMessage([userMessage])
    setIsSending(true)

    try {
      const data = await apiFetch(`/api/model`, {
        method: "POST",
        body: JSON.stringify({
          sessionId: chatId,
          modelType: modelType, // ส่ง modelType ที่เลือก
          input: {
            messages: [
              { role: "user", content: messageContent }
            ]
          }
        }),
      })

      const modelMessage: Message = {
        messageId: crypto.randomUUID(),
        role: "model",
        content: data.data?.message || "No response",
        createdAt: new Date().toISOString(),
        feedback: null,
      }

      appendMessage([modelMessage])
    } catch (error) {
      console.error("Failed to send message:", error)
      setInput(messageContent)
    } finally {
      setIsSending(false)
    }
  }, [appendMessage, apiFetch, chatId, modelType]) // เพิ่ม modelType ใน dependencies

  // Theme toggle
  const toggleTheme = () => {
    const newTheme = !isDark
    setIsDark(newTheme)
    localStorage.setItem("chatbot-theme", newTheme ? "dark" : "light")
  }

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault()
    if (!input.trim() || isSending) return
    const messageToSend = input
    setInput("")
    sendMessage(messageToSend)
  }

  const handleCopy = (content: string) => {
    navigator.clipboard.writeText(content).then(() => {
    }).catch((error) => {
      console.error("Failed to copy message:", error);
    });
  };

  const handleLike = useCallback(async (messageId: string) => {
    try {
      setChat(prev => {
        if (!prev) return prev
        return {
          ...prev,
          messages: prev.messages.map(msg =>
            msg.messageId === messageId
              ? { ...msg, feedback: msg.feedback === 1 ? null : 1 } // Toggle like
              : msg
          )
        }
      })

      await apiFetch(`/api/feedback/${messageId}`, {
        method: "PATCH",
        body: JSON.stringify({
          feedback: 1,
        }),
      })
      console.log("✅ Liked message:", messageId)
    } catch (error) {
      console.error("Failed to send like feedback:", error)
      // Revert ถ้า error
      setChat(prev => {
        if (!prev) return prev
        return {
          ...prev,
          messages: prev.messages.map(msg =>
            msg.messageId === messageId
              ? { ...msg, feedback: null }
              : msg
          )
        }
      })
    }
  }, [apiFetch])

  const handleDislike = useCallback(async (messageId: string) => {
    try {
      setChat(prev => {
        if (!prev) return prev
        return {
          ...prev,
          messages: prev.messages.map(msg =>
            msg.messageId === messageId
              ? { ...msg, feedback: msg.feedback === -1 ? null : -1 } // Toggle dislike
              : msg
          )
        }
      })

      await apiFetch(`/api/feedback/${messageId}`, {
        method: "PATCH",
        body: JSON.stringify({
          feedback: -1,
        }),
      })
      console.log("✅ Disliked message:", messageId)
    } catch (error) {
      console.error("Failed to send dislike feedback:", error)
      // Revert ถ้า error
      setChat(prev => {
        if (!prev) return prev
        return {
          ...prev,
          messages: prev.messages.map(msg =>
            msg.messageId === messageId
              ? { ...msg, feedback: null }
              : msg
          )
        }
      })
    }
  }, [apiFetch])

  if (!chat) return null

  return (
    <div className={isDark ? "dark" : ""}>
      <div
        className={`flex h-screen transition-all duration-300 ${!sidebarOpen ? 'bg-center' : 'bg-[center_right_-130px]'} ${isDark ? "bg-cover bg-[url('/dark-bg.png')] text-white" : "bg-cover bg-[url('/light-bg.png')] text-slate-900"}`}
        // className={`flex h-screen ${
        //   isDark ? "bg-[#1E1E1E] text-white" : "bg-gradient-to-br from-purple-50 via-blue-50 to-indigo-100 text-slate-900"
        // }`}
      >
        {/* Sidebar */}
        <SharedSidebar
          isDark={isDark}
          onToggleTheme={toggleTheme}
          currentChatId={chatId}
          isOpen={sidebarOpen}
          onToggle={() => setSidebarOpen(!sidebarOpen)}
        />

        {!sidebarOpen && (
          <button
            onClick={() => setSidebarOpen(true)}
            className={`fixed top-4 left-4 z-50 flex h-10 w-10 items-center justify-center rounded-lg transition-colors ${
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

        {/* Main Chat Area */}
        <main className="flex flex-1 flex-col overflow-hidden">
          <div className="custom-scroll flex-1 overflow-y-auto p-4 md:p-6 space-y-6">
            {chat.messages.length === 0 ? (
              <div className={`text-center py-12 ${isDark ? "text-slate-500" : "text-slate-400"}`}>
                No messages yet. Start the conversation!
              </div>
            ) : (
                  chat.messages.map((message) => {
                    const isUser = message.role === "user";
                    
                    return (
                      <React.Fragment key={message.messageId}>
                        <MessageBubble message={message} isDark={isDark} />
                        
                        {!isUser && (
                          <div className="flex items-center mt-2 mb-6 w-[86%] mx-auto">
                            <button 
                              onClick={() => handleCopy(message.content)} 
                              className={`flex items-center justify-center p-2 cursor-pointer rounded-lg ${
                                isDark
                                  ? "hover:bg-[#3C3C3C]"
                                  : "hover:bg-[#C7CDE4]"
                              }`}
                              aria-label="Copy message"
                            >
                              <Image
                                src={isDark ? "/copy-dark.png" : "/copy-light.png"}
                                alt="Copy"
                                width={18}
                                height={18}
                              />
                            </button>
                            <button 
                              onClick={() => handleLike(message.messageId)} 
                              className={`flex items-center justify-center p-2 cursor-pointer rounded-lg ${
                                message.feedback === 1
                                  ? isDark
                                    ? "bg-[#3C3C3C]"
                                    : "bg-[#C7CDE4]" 
                                  : isDark
                                    ? "hover:bg-[#3C3C3C]"
                                    : "hover:bg-[#C7CDE4]"
                              }`}
                              aria-label="Like message"
                            >
                              <Image
                                src={isDark ? "/like-dark.png" : "/like-light.png"}
                                alt="Like"
                                width={18}
                                height={18}
                              />
                            </button>
                            <button 
                              onClick={() => handleDislike(message.messageId)} 
                              className={`flex items-center justify-center p-2 cursor-pointer rounded-lg ${
                                message.feedback === -1
                                  ? isDark
                                    ? "bg-[#3C3C3C]"
                                    : "bg-[#C7CDE4]" 
                                  : isDark
                                    ? "hover:bg-[#3C3C3C]"
                                    : "hover:bg-[#C7CDE4]"
                              }`}
                              aria-label="Dislike message"
                            >
                              <Image
                                src={isDark ? "/dislike-dark.png" : "/dislike-light.png"}
                                alt="Dislike"
                                width={18}
                                height={18}
                              />
                            </button>
                          </div>
                        )}
                        {!isUser && (
                          <div
                            className={`mt-4 mb-6 h-[0.5px] w-[86%] mx-auto ${isDark ? "bg-white/10" : "bg-[#D7DFFF]"}`}
                          ></div>
                        )}
                      </React.Fragment>
                    );
                  })
                )}

            {/* Loading Indicator */}
            {isSending && <TypingIndicator isDark={isDark} />}
            <div ref={messagesEndRef} />
          </div>

          {/* Input Form */}
          <ChatInput
            input={input}
            setInput={setInput}
            handleSubmit={handleSubmit}
            isSending={isSending}
            isDark={isDark}
            modelType={modelType}
            setModelType={setModelType}
          />
        </main>
      </div>
    </div>
  )
}

const MessageBubble = ({ message, isDark }: { message: Message; isDark: boolean }) => {
  const isUser = message.role === "user"
  return (
    <div className={`flex gap-4 ${isUser ? "justify-end" : "justify-start"}`}>
      {!isUser && (
        // <div className="flex h-8 w-8 items-center justify-center rounded-full bg-gradient-to-br from-blue-500 to-purple-600">
        <div className="flex h-12 w-12 items-center justify-center rounded-full ">
          <Image
            src="/AiLaw.png"
            alt="Model Icon"
            width={48}
            height={48}
            className="rounded-full"
          />
        </div>
      )}
      <div 
        className={`max-w-[90%] md:max-w-[90%] rounded-2xl px-5 py-3 ${
        // className={`max-w-[85%] md:max-w-2xl rounded-2xl px-5 py-3 ${
          isUser
            ? isDark
              ? "bg-white/10 text-white"
              : "bg-[#4557A1] text-white"
              // : "bg-[#E8EAF3] text-slate-900"
            : isDark
              ? "text-slate-100"
              : "text-slate-900"
            // ? isDark
            //   ? "bg-[#3C3C3C] text-white"
            //   : "bg-white/80 text-slate-900"
            // : isDark
            //   ? "bg-[#3C3C3C] text-slate-100"
            //   : "bg-white/80 backdrop-blur-sm text-slate-900 shadow-sm"
        }`}
      >
        <p className="text-pretty leading-relaxed whitespace-pre-wrap ">{message.content}</p>
      </div>
      {isUser && (
        <div
          className={`flex h-10 w-10 items-center justify-center rounded-full`}
        >
          <Image
            src="/user-profile.svg"
            alt="User Icon"
            width={48}
            height={48}
            className={isDark ? "invert" : ""}
          />
        </div>
      )}
    </div>
  )
}

const TypingIndicator = ({ isDark }: { isDark: boolean }) => (
  <div className="flex gap-4 justify-start">
    <div className="flex h-12 w-12 items-center justify-center rounded-full">
      <Image
            src="/AiLaw.png"
            alt="Model Icon"
            width={48}
            height={48}
            className="rounded-full"
          />
    </div>
    <div
      className={`max-w-[85%] md:max-w-2xl rounded-2xl px-5 py-3 ${
        // isDark ? "bg-slate-800 text-slate-100" : "bg-white/80 backdrop-blur-sm text-slate-900 shadow-sm"
        isDark ? "text-slate-100" : "text-slate-900"
      }`}
    >
      <div className="flex gap-1">
        <span className="animate-bounce">●</span>
        <span className="animate-bounce" style={{ animationDelay: "0.1s" }}>
          ●
        </span>
        <span className="animate-bounce" style={{ animationDelay: "0.2s" }}>
          ●
        </span>
      </div>
    </div>
  </div>
)