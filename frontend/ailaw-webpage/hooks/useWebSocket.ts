"use client"

import { useCallback, useEffect, useRef, useState } from "react"

const WS_URL = process.env.NEXT_PUBLIC_WS_URL || "ws://localhost:8080"

export interface WSMessage {
  type: "chat" | "ping"
  sessionId?: string
  content?: string
  modelType?: string
}

export interface WSResponse {
  type: 
    | "ack" 
    | "chunk" 
    | "done" 
    | "error" 
    | "pong"
    | "guard_passed"
    | "status"
    | "plan"
    | "cot_step"
  content?: string
  sessionId?: string
  modelMessageId?: string
  error?: string
  usage?: {
    inputTokens: number
    outputTokens: number
    totalTokens: number
    remaining: number
  }
  steps?: string[]
  rationale?: string
  currentStep?: number
  totalSteps?: number
  stepDescription?: string
  status?: string
}

interface UseWebSocketOptions {
  onChunk?: (content: string, sessionId: string) => void
  onDone?: (response: WSResponse) => void
  onError?: (error: string) => void
  onConnect?: () => void
  onDisconnect?: () => void
  onPlan?: (steps: string[], rationale: string) => void
  onCotStep?: (step: number, total: number, description: string) => void
  onStatus?: (status: string) => void
}

export function useWebSocket(
  accessToken: string | null,
  options: UseWebSocketOptions = {}
) {
  const [isConnected, setIsConnected] = useState(false)
  const [isConnecting, setIsConnecting] = useState(false)
  
  const wsRef = useRef<WebSocket | null>(null)
  const reconnectTimeoutRef = useRef<NodeJS.Timeout | null>(null)
  const reconnectAttemptsRef = useRef(0)
  const maxReconnectAttempts = 5

  // ✅ ใช้ ref สำหรับ callbacks เพื่อไม่ให้ trigger re-render
  const optionsRef = useRef(options)
  optionsRef.current = options

  // ✅ ใช้ ref สำหรับ token
  const tokenRef = useRef(accessToken)
  tokenRef.current = accessToken

  const connect = useCallback(() => {
    const token = tokenRef.current
    
    // ป้องกัน connect ซ้ำ
    if (!token) {
      console.log("No token, skipping connect")
      return
    }
    
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      console.log("Already connected, skipping")
      return
    }
    
    if (wsRef.current?.readyState === WebSocket.CONNECTING) {
      console.log("Already connecting, skipping")
      return
    }

    setIsConnecting(true)

    const wsUrl = `${WS_URL}/api/ws?token=${encodeURIComponent(token)}`
    console.log("🔌 Connecting to WebSocket...")
    
    const ws = new WebSocket(wsUrl)

    ws.onopen = () => {
      console.log("✅ WebSocket connected")
      setIsConnected(true)
      setIsConnecting(false)
      reconnectAttemptsRef.current = 0
      optionsRef.current.onConnect?.()
    }

    ws.onmessage = (event) => {
      try {
        const data: WSResponse = JSON.parse(event.data)
        const opts = optionsRef.current
        
        switch (data.type) {
          case "chunk":
            if (data.content && data.sessionId) {
              opts.onChunk?.(data.content, data.sessionId)
            }
            break
          case "done":
            opts.onDone?.(data)
            break
          case "error":
            console.error("WebSocket error:", data.error)
            opts.onError?.(data.error || "Unknown error")
            break
          case "ack":
            console.log("Message acknowledged:", data.sessionId)
            break
          case "pong":
            break
          case "guard_passed":
            console.log("Guard validation passed")
            break
          case "status":
            if (data.status) opts.onStatus?.(data.status)
            break
          case "plan":
            if (data.steps) opts.onPlan?.(data.steps, data.rationale || "")
            break
          case "cot_step":
            if (data.currentStep !== undefined && data.totalSteps !== undefined) {
              opts.onCotStep?.(data.currentStep, data.totalSteps, data.stepDescription || "")
            }
            break
        }
      } catch (e) {
        console.error("Failed to parse WebSocket message:", e)
      }
    }

    ws.onerror = (error) => {
      console.error("WebSocket error:", error)
      setIsConnecting(false)
    }

    ws.onclose = (event) => {
      console.log("WebSocket closed:", event.code, event.reason)
      wsRef.current = null
      setIsConnected(false)
      setIsConnecting(false)
      optionsRef.current.onDisconnect?.()

      // Auto reconnect (ยกเว้น normal close)
      if (event.code !== 1000 && reconnectAttemptsRef.current < maxReconnectAttempts && tokenRef.current) {
        const delay = Math.min(1000 * Math.pow(2, reconnectAttemptsRef.current), 30000)
        console.log(`Reconnecting in ${delay}ms... (attempt ${reconnectAttemptsRef.current + 1})`)
        
        reconnectTimeoutRef.current = setTimeout(() => {
          reconnectAttemptsRef.current++
          connect()
        }, delay)
      }
    }

    wsRef.current = ws
  }, []) // ✅ ไม่มี dependency! ใช้ refs แทน

  const disconnect = useCallback(() => {
    console.log("🔌 Disconnecting WebSocket...")
    
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current)
      reconnectTimeoutRef.current = null
    }
    
    reconnectAttemptsRef.current = maxReconnectAttempts // ป้องกัน auto-reconnect
    
    if (wsRef.current) {
      wsRef.current.close(1000, "Client disconnect") // 1000 = normal close
      wsRef.current = null
    }
    
    setIsConnected(false)
    setIsConnecting(false)
  }, []) // ✅ ไม่มี dependency!

  const sendMessage = useCallback((message: WSMessage): boolean => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(message))
      return true
    }
    console.warn("WebSocket not connected")
    return false
  }, [])

  const sendChat = useCallback((
    sessionId: string,
    content: string,
    modelType: string = "default"
  ): boolean => {
    return sendMessage({
      type: "chat",
      sessionId,
      content,
      modelType,
    })
  }, [sendMessage])

  // ✅ Connect เมื่อ token เปลี่ยน (จาก null → มีค่า)
  useEffect(() => {
    if (accessToken && !wsRef.current) {
      connect()
    }
    
    // Cleanup เมื่อ unmount เท่านั้น
    return () => {
      if (wsRef.current) {
        disconnect()
      }
    }
  }, [accessToken]) // ✅ ขึ้นกับ accessToken เท่านั้น ไม่ใส่ connect/disconnect

  // Heartbeat ping
  useEffect(() => {
    if (!isConnected) return

    const interval = setInterval(() => {
      sendMessage({ type: "ping" })
    }, 30000)

    return () => clearInterval(interval)
  }, [isConnected, sendMessage])

  return {
    isConnected,
    isConnecting,
    connect,
    disconnect,
    sendMessage,
    sendChat,
  }
}