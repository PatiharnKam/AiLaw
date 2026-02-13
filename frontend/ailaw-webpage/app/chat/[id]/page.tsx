"use client"

import React from "react"
import Image from "next/image"
import { useEffect, useState, useRef, useCallback } from "react"
import { useRouter, useParams } from "next/navigation"
import { useModelType } from "@/hooks/useModelType"
import { useWebSocket, WSResponse } from "@/hooks/useWebSocket"
import { useToast } from "@/hooks/useToast"
import { ToastContainer } from "@/components/toast"
import { parseApiError, parseWSError } from "@/utils/errorMapping"
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
  isStreaming?: boolean
  statusMessage?: string
}

interface Chat {
  sessionId: string
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
  const { toasts, removeToast, showError } = useToast()
  const pendingMessageRef = useRef<string>("")

  // ============ WebSocket Setup ============
  const handleChunk = useCallback((content: string, sessionId: string) => {
    setChat(prev => {
      if (!prev) return prev

      const messages = prev.messages.map(m => {
        if (m.isStreaming) {
          return {
            ...m,
            content: m.content + content,
            statusMessage: undefined,
          }
        }
        return m
      })

      return { ...prev, messages }
    })
  }, [])

  const handleStatus = useCallback((status: string) => {
    console.log("📊 Status:", status)
    
    setChat(prev => {
      if (!prev) return prev

      const messages = prev.messages.map(m => {
        if (m.isStreaming) {
          return {
            ...m,
            statusMessage: status,  //status message
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
        return prev
      }

      const messages = prev.messages.map(m => {
        if (m.isStreaming) {
          return {
            ...m,
            messageId: response.modelMessageId || m.messageId,
            isStreaming: false,
            statusMessage: undefined,
          }
        }
        return m
      })

      return { ...prev, messages }
    })

    streamingMessageIdRef.current = null
    pendingMessageRef.current = ""
    setIsSending(false)
  }, [])

  const handleError = useCallback((error: string | any) => {
    const errorInfo = parseWSError(error)
    showError(errorInfo.title, errorInfo.message)
    
    const messageToRestore = pendingMessageRef.current
    if (messageToRestore) {
      setInput(messageToRestore)
      pendingMessageRef.current = ""
    }
    
    setChat(prev => {
      if (!prev) return prev
      const messagesWithoutError = prev.messages.filter(m => !m.isStreaming)
      
      if (messagesWithoutError.length > 0) {
        const lastMessage = messagesWithoutError[messagesWithoutError.length - 1]
        if (lastMessage.role === "user") {
          return {
            ...prev,
            messages: messagesWithoutError.slice(0, -1)
          }
        }
      }
      
      return {
        ...prev,
        messages: messagesWithoutError
      }
    })
    
    streamingMessageIdRef.current = null
    setIsSending(false)
  }, [showError])

  const { isConnected, sendChat } = useWebSocket(accessToken, {
    onChunk: handleChunk,
    onDone: handleDone,
    onError: handleError,
    onStatus: handleStatus,
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
    
    if (currentMessageCount > previousMessageCountRef.current) {
      messagesEndRef.current?.scrollIntoView({ behavior: "smooth" })
    }
    
    previousMessageCountRef.current = currentMessageCount
  }, [chat?.messages?.length])

  useEffect(() => {
    const streamingMessage = chat?.messages.find(m => m.isStreaming)
    
    if (streamingMessage) {
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

  const loadMessages = useCallback(async () => {
    try {
      const data = await apiFetch(`/api/messages-history/${chatId}`, { method: "GET" })
      setChat({
        sessionId: chatId,
        messages: data.data || [],
      })
    } catch (error: any) {
      const errorInfo = parseApiError(error)
      showError(errorInfo.title, errorInfo.message)
      
      router.push("/welcome")
    }
  }, [apiFetch, chatId, router, showError])

  // ============ Send Message (WebSocket) ============
  const sendMessage = useCallback(async (messageContent: string) => {
    if (!messageContent.trim() || isSending) return

    const tempUserId = crypto.randomUUID()
    const tempModelId = crypto.randomUUID()

    pendingMessageRef.current = messageContent

    const userMessage: Message = {
      messageId: tempUserId,
      role: "user",
      content: messageContent,
      createdAt: new Date().toISOString(),
      feedback: null,
    }

    const modelMessage: Message = {
      messageId: tempModelId,
      role: "model",
      content: "",
      createdAt: new Date().toISOString(),
      feedback: null,
      isStreaming: true,
      statusMessage: "กำลังประมวลผล...",
    }

    setChat(prev => ({
      sessionId: prev?.sessionId ?? chatId,
      messages: [...(prev?.messages ?? []), userMessage, modelMessage],
    }))

    setIsSending(true)
    streamingMessageIdRef.current = tempModelId

    const sent = sendChat(chatId, messageContent, modelType)

    if (!sent) {
      await sendMessageHTTP(messageContent, tempUserId, tempModelId)
    }
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
                statusMessage: undefined,
              }
            }
            return m
          }),
        }
      })
      
      pendingMessageRef.current = ""
    } catch (error: any) {
      const errorInfo = parseApiError(error)
      showError(errorInfo.title, errorInfo.message)
      
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
      pendingMessageRef.current = ""
    } finally {
      setIsSending(false)
      streamingMessageIdRef.current = null
    }
  }, [apiFetch, chatId, modelType, showError])

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
    const currentMessage = chat?.messages.find(m => m.messageId === messageId)
    if (!currentMessage) return

    const newFeedback = currentMessage.feedback === 1 ? null : 1

    try {
      setChat(prev => {
        if (!prev) return prev
        return {
          ...prev,
          messages: prev.messages.map(msg =>
            msg.messageId === messageId
              ? { ...msg, feedback: newFeedback }
              : msg
          ),
        }
      })

      await apiFetch(`/api/feedback/${messageId}`, {
        method: "PATCH",
        body: JSON.stringify({ feedback: newFeedback }),
      })
      
    } catch (error: any) {
      const errorInfo = parseApiError(error)
      showError(errorInfo.title, errorInfo.message)
      
      setChat(prev => {
        if (!prev) return prev
        return {
          ...prev,
          messages: prev.messages.map(msg =>
            msg.messageId === messageId
              ? { ...msg, feedback: currentMessage.feedback }
              : msg
          ),
        }
      })
    }
  }, [apiFetch, chat?.messages, showError])

  const handleDislike = useCallback(async (messageId: string) => {
    const currentMessage = chat?.messages.find(m => m.messageId === messageId)
    if (!currentMessage) return

    const newFeedback = currentMessage.feedback === -1 ? null : -1

    try {
      setChat(prev => {
        if (!prev) return prev
        return {
          ...prev,
          messages: prev.messages.map(msg =>
            msg.messageId === messageId
              ? { ...msg, feedback: newFeedback }
              : msg
          ),
        }
      })

      await apiFetch(`/api/feedback/${messageId}`, {
        method: "PATCH",
        body: JSON.stringify({ feedback: newFeedback }),
      })
      
    } catch (error: any) {
      const errorInfo = parseApiError(error)
      showError(errorInfo.title, errorInfo.message)
      
      setChat(prev => {
        if (!prev) return prev
        return {
          ...prev,
          messages: prev.messages.map(msg =>
            msg.messageId === messageId
              ? { ...msg, feedback: currentMessage.feedback }
              : msg
          ),
        }
      })
    }
  }, [apiFetch, chat?.messages, showError])

  if (!chat) return null

  return (
    <div className={isDark ? "dark" : ""}>
      <ToastContainer toasts={toasts} onClose={removeToast} />
      
      <div
        className={`flex h-screen transition-all duration-300 ${
          !sidebarOpen ? "bg-center" : "bg-[center_right_-130px]"
        } ${
          isDark
            ? "bg-cover bg-[url('/dark-bg.png')] text-white"
            : "bg-cover bg-[url('/light-bg.png')] text-slate-900"
        }`}
      >
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
        {isStreaming && message.statusMessage && (
          <div className="flex items-center gap-2 mb-2">
            <div className="flex space-x-1">
              <div className={`w-2 h-2 rounded-full animate-bounce ${isDark ? "bg-slate-300" : "bg-slate-600"}`} style={{ animationDelay: "0ms" }} />
              <div className={`w-2 h-2 rounded-full animate-bounce ${isDark ? "bg-slate-300" : "bg-slate-600"}`} style={{ animationDelay: "150ms" }} />
              <div className={`w-2 h-2 rounded-full animate-bounce ${isDark ? "bg-slate-300" : "bg-slate-600"}`} style={{ animationDelay: "300ms" }} />
            </div>
            <span className={`text-sm italic ${isDark ? "text-slate-400" : "text-slate-600"}`}>
              {message.statusMessage}
            </span>
          </div>
        )}

        <p className="text-pretty leading-relaxed whitespace-pre-wrap">
          {message.content || (isStreaming && !message.statusMessage && (
            <span className={`text-sm italic ${isDark ? "text-slate-400" : "text-slate-600"}`}>
              กำลังประมวลผล...
            </span>
          ))}
          {isStreaming && message.content && (
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