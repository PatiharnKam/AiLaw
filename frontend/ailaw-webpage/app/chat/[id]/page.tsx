"use client"

import React from "react"
import Image from "next/image"
import { useEffect, useState, useRef, useCallback } from "react"
import { useRouter, useParams } from "next/navigation"
import { useModelType } from "@/hooks/useModelType"
import { useWebSocket, WSResponse } from "@/hooks/useWebSocket"
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
  isStreaming?: boolean  // New: flag for streaming message
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
  const { modelType, setModelType } = useModelType()
  const { accessToken, logout, refreshToken, getToken } = useAuth()
  const messagesEndRef = useRef<HTMLDivElement>(null)
  const { prompt, setPrompt } = usePrompt()
  const initializedRef = useRef(false)
  const isRefreshingRef = useRef(false)
  const previousMessageCountRef = useRef(0)
  const streamingMessageIdRef = useRef<string | null>(null)

  // ============ WebSocket Setup ============
  const handleChunk = useCallback((content: string, sessionId: string) => {
    console.log("📝 Chunk received, length:", content.length)

    setChat(prev => {
      if (!prev) return prev

      const messages = prev.messages.map(m => {
        if (m.isStreaming) {
          return {
            ...m,
            content: m.content + content,
          }
        }
        return m
      })

      return { ...prev, messages }
    })
  }, [])

  const handleDone = useCallback((response: WSResponse) => {
    setChat(prev => {
      if (!prev) return prev

      const streamingMsg = prev.messages.find(m => m.isStreaming)
      if (!streamingMsg) {
        console.log("⚠️ No streaming message in state!")
        return prev
      }

      console.log("📝 Found streaming message:", streamingMsg.messageId)

      const messages = prev.messages.map(m => {
        if (m.isStreaming) {
          return {
            ...m,
            messageId: response.modelMessageId || m.messageId,
            isStreaming: false,  
          }
        }
        return m
      })

      return { ...prev, messages }
    })

    streamingMessageIdRef.current = null
    setIsSending(false)

    if (response.usage) {
      console.log("Token usage:", response.usage)
    }
  }, [])

  const handleError = useCallback((error: string) => {
    setChat(prev => {
      if (!prev) return prev
      return {
        ...prev,
        messages: prev.messages.map(m => {
          if (m.isStreaming) {
            return {
              ...m,
              content: m.content || `เกิดข้อผิดพลาด: ${error}`,
              isStreaming: false,
            }
          }
          return m
        }),
      }
    })
    
    streamingMessageIdRef.current = null
    setIsSending(false)
  }, [])

  const { isConnected, sendChat } = useWebSocket(accessToken, {
    onChunk: handleChunk,
    onDone: handleDone,
    onError: handleError,
    onConnect: () => console.log("WebSocket connected"),
    onDisconnect: () => console.log("WebSocket disconnected"),
  })

  // ============ Initialization ============
  useEffect(() => {
    if (initializedRef.current) return
    initializedRef.current = true

    const initialize = async () => {
      if (!accessToken) {
        const refreshed = await refreshToken()
        if (!refreshed) {
          logout()
          return
        }
      }

      await loadMessages()

      if (prompt) {
        await sendMessage(String(prompt))
        setPrompt(null)
      }
    }

    initialize()
  }, [])

  useEffect(() => {
    const currentMessageCount = chat?.messages?.length || 0
    
    //Check for new message
    if (currentMessageCount > previousMessageCountRef.current) {
      messagesEndRef.current?.scrollIntoView({ behavior: "smooth" })
    }
    
    previousMessageCountRef.current = currentMessageCount
  }, [chat?.messages?.length])

  useEffect(() => {
  const streamingMessage = chat?.messages.find(m => m.isStreaming)
  
  if (streamingMessage) {
    // ใช้ requestAnimationFrame เพื่อรอ DOM update
    const scrollToBottom = () => {
      requestAnimationFrame(() => {
        messagesEndRef.current?.scrollIntoView({ behavior: "smooth" })
      })
    }
    
    scrollToBottom()
  }
}, [chat?.messages])

  useEffect(() => {
    const savedTheme = localStorage.getItem("chatbot-theme")
    if (savedTheme) setIsDark(savedTheme === "dark")

    const handleResize = () => setSidebarOpen(window.innerWidth >= 768)
    handleResize()
    window.addEventListener("resize", handleResize)
    return () => window.removeEventListener("resize", handleResize)
  }, [])

  // ============ API Functions ============
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

  // ============ Send Message (WebSocket) ============
  const sendMessage = useCallback(async (messageContent: string) => {
    if (!messageContent.trim() || isSending) return

    const tempUserId = crypto.randomUUID()
    const tempModelId = crypto.randomUUID()

    // Add user message
    const userMessage: Message = {
      messageId: tempUserId,
      role: "user",
      content: messageContent,
      createdAt: new Date().toISOString(),
      feedback: null,
    }

    // Add empty model message (will be filled by streaming)
    const modelMessage: Message = {
      messageId: tempModelId,
      role: "model",
      content: "",
      createdAt: new Date().toISOString(),
      feedback: null,
      isStreaming: true,
    }

    setChat(prev => ({
      sessionId: prev?.sessionId ?? chatId,
      title: prev?.title ?? "New Chat",
      messages: [...(prev?.messages ?? []), userMessage, modelMessage],
    }))

    setIsSending(true)
    streamingMessageIdRef.current = tempModelId

    // Send via WebSocket
    const sent = sendChat(chatId, messageContent, modelType)

    if (!sent) {
      // Fallback to HTTP if WebSocket not connected
      console.warn("WebSocket not connected, falling back to HTTP")
      await sendMessageHTTP(messageContent, tempUserId, tempModelId)
    }
    // await sendMessageHTTP(messageContent, tempUserId, tempModelId)
  }, [chatId, modelType, isSending, sendChat])

  // Fallback HTTP send (if WebSocket fails)
  const sendMessageHTTP = useCallback(async (
    messageContent: string,
    tempUserId: string,
    tempModelId: string
  ) => {
    try {
      const data = await apiFetch(`/api/model`, {
        method: "POST",
        body: JSON.stringify({
          sessionId: chatId,
          modelType: modelType,
          input: { messages: { role: "user", content: messageContent } },
        }),
      })

      setChat(prev => {
        if (!prev) return prev
        return {
          ...prev,
          messages: prev.messages.map(m => {
            if (m.messageId === tempModelId) {
              return {
                ...m,
                messageId: data.data.modelMessageID,
                content: data.data?.message || "No response",
                isStreaming: false,
              }
            }
            return m
          }),
        }
      })
    } catch (error) {
      console.error("HTTP fallback failed:", error)
      // Remove messages on error
      setChat(prev => {
        if (!prev) return prev
        return {
          ...prev,
          messages: prev.messages.filter(
            m => m.messageId !== tempUserId && m.messageId !== tempModelId
          ),
        }
      })
      setInput(messageContent)
    } finally {
      setIsSending(false)
      streamingMessageIdRef.current = null
    }
  }, [apiFetch, chatId, modelType])

  // ============ Event Handlers ============
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
    navigator.clipboard.writeText(content)
  }

  const handleLike = useCallback(async (messageId: string) => {
    try {
      setChat(prev => {
        if (!prev) return prev
        return {
          ...prev,
          messages: prev.messages.map(msg =>
            msg.messageId === messageId
              ? { ...msg, feedback: msg.feedback === 1 ? null : 1 }
              : msg
          ),
        }
      })

      await apiFetch(`/api/feedback/${messageId}`, {
        method: "PATCH",
        body: JSON.stringify({ feedback: 1 }),
      })
    } catch (error) {
      console.error("Failed to send like feedback:", error)
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
              ? { ...msg, feedback: msg.feedback === -1 ? null : -1 }
              : msg
          ),
        }
      })

      await apiFetch(`/api/feedback/${messageId}`, {
        method: "PATCH",
        body: JSON.stringify({ feedback: -1 }),
      })
    } catch (error) {
      console.error("Failed to send dislike feedback:", error)
    }
  }, [apiFetch])

  if (!chat) return null

  return (
    <div className={isDark ? "dark" : ""}>
      <div
        className={`flex h-screen transition-all duration-300 ${
          !sidebarOpen ? "bg-center" : "bg-[center_right_-130px]"
        } ${
          isDark
            ? "bg-cover bg-[url('/dark-bg.png')] text-white"
            : "bg-cover bg-[url('/light-bg.png')] text-slate-900"
        }`}
      >
        {/* Connection Status Indicator */}
        <div className="fixed top-4 right-4 z-50">
          <div
            className={`flex items-center gap-2 px-3 py-1 rounded-full text-sm ${
              isConnected
                ? "bg-green-500/20 text-green-400"
                : "bg-red-500/20 text-red-400"
            }`}
          >
            <div
              className={`w-2 h-2 rounded-full ${
                isConnected ? "bg-green-400" : "bg-red-400"
              }`}
            />
            {isConnected ? "Connected" : "Disconnected"}
          </div>
        </div>

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
                const isUser = message.role === "user"

                return (
                  <React.Fragment key={message.messageId}>
                    <MessageBubble 
                      message={message} 
                      isDark={isDark}
                      isStreaming={message.isStreaming}
                    />

                    {!isUser && !message.isStreaming && (
                      <div className="flex items-center mt-2 mb-6 w-[86%] mx-auto">
                        <button
                          onClick={() => handleCopy(message.content)}
                          className={`flex items-center justify-center p-2 cursor-pointer rounded-lg ${
                            isDark ? "hover:bg-[#3C3C3C]" : "hover:bg-[#C7CDE4]"
                          }`}
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
                              ? isDark ? "bg-[#3C3C3C]" : "bg-[#C7CDE4]"
                              : isDark ? "hover:bg-[#3C3C3C]" : "hover:bg-[#C7CDE4]"
                          }`}
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
                              ? isDark ? "bg-[#3C3C3C]" : "bg-[#C7CDE4]"
                              : isDark ? "hover:bg-[#3C3C3C]" : "hover:bg-[#C7CDE4]"
                          }`}
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
                    
                    {!isUser && !message.isStreaming && (
                      <div
                        className={`mt-4 mb-6 h-[0.5px] w-[86%] mx-auto ${
                          isDark ? "bg-white/10" : "bg-[#D7DFFF]"
                        }`}
                      />
                    )}
                  </React.Fragment>
                )
              })
            )}
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

// ============ Components ============

const MessageBubble = ({ 
  message, 
  isDark,
  isStreaming 
}: { 
  message: Message
  isDark: boolean
  isStreaming?: boolean
}) => {
  const isUser = message.role === "user"

  return (
    <div className={`flex gap-4 ${isUser ? "justify-end" : "justify-start"}`}>
      {!isUser && (
        <div className="flex h-12 w-12 items-center justify-center rounded-full">
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
          isUser
            ? isDark
              ? "bg-white/10 text-white"
              : "bg-[#4557A1] text-white"
            : isDark
              ? "text-slate-100"
              : "text-slate-900"
        }`}
      >
        <p className="text-pretty leading-relaxed whitespace-pre-wrap">
          {message.content}
          {isStreaming && (
            <span className="inline-block w-2 h-4 ml-1 bg-current animate-pulse" />
          )}
        </p>
      </div>
      {isUser && (
        <div className="flex h-10 w-10 items-center justify-center rounded-full">
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