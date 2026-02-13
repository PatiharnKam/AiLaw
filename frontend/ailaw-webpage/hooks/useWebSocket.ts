"use client"

import { useCallback, useEffect, useRef, useState } from "react"

const WS_URL = process.env.NEXT_PUBLIC_WS_URL

export interface WSMessage {
  type: "chat" | "ping"
  sessionId?: string
  content?: string
  modelType?: string
}

export interface WSError {
  code?: string
  message?: string
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
  error?: WSError
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
  onError?: (error: WSError | string) => void
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

  const optionsRef = useRef(options)
  optionsRef.current = options

  const tokenRef = useRef(accessToken)
  tokenRef.current = accessToken

  const connect = useCallback(() => {
    const token = tokenRef.current
    
    if (!token) {
      return
    }
    
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      return
    }
    
    if (wsRef.current?.readyState === WebSocket.CONNECTING) {
      return
    }

    setIsConnecting(true)

    const wsUrl = `${WS_URL}/api/ws?token=${encodeURIComponent(token)}`
    
    const ws = new WebSocket(wsUrl)

    ws.onopen = () => {
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
            opts.onError?.(data.error || "Unknown error")
            break
          case "ack":
            break
          case "pong":
            break
          case "guard_passed":
            break
          case "status":
            console.log("status is :",data.status)
            if (data.status) opts.onStatus?.(data.status)
            break
          case "plan":
            // console.log("plan step :",data.steps)
            // console.log("rationale step :",data.rationale)
            if (data.steps) opts.onPlan?.(data.steps, data.rationale || "")
            break
          case "cot_step":
            console.log("Current steps :",data.currentStep)
            console.log("Total steps :",data.totalSteps)
            // console.log("Step Description :",data.stepDescription)
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
      setIsConnecting(false)
    }

    ws.onclose = (event) => {
      wsRef.current = null
      setIsConnected(false)
      setIsConnecting(false)
      optionsRef.current.onDisconnect?.()

      if (event.code !== 1000 && reconnectAttemptsRef.current < maxReconnectAttempts && tokenRef.current) {
        const delay = Math.min(1000 * Math.pow(2, reconnectAttemptsRef.current), 30000)
        
        reconnectTimeoutRef.current = setTimeout(() => {
          reconnectAttemptsRef.current++
          connect()
        }, delay)
      }
    }

    wsRef.current = ws
  }, [])

  const disconnect = useCallback(() => {
    
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current)
      reconnectTimeoutRef.current = null
    }
    
    reconnectAttemptsRef.current = maxReconnectAttempts
    
    if (wsRef.current) {
      wsRef.current.close(1000, "Client disconnect")
      wsRef.current = null
    }
    
    setIsConnected(false)
    setIsConnecting(false)
  }, [])

  const sendMessage = useCallback((message: WSMessage): boolean => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      wsRef.current.send(JSON.stringify(message))
      return true
    }
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

  useEffect(() => {
    if (accessToken && !wsRef.current) {
      connect()
    }
    
    return () => {
      if (wsRef.current) {
        disconnect()
      }
    }
  }, [accessToken])

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