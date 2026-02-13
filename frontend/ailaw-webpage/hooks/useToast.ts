"use client"

import { useState, useCallback } from "react"
import { ToastType } from "@/components/toast"

interface ToastData {
  id: string
  type: ToastType
  title: string
  message: string
}

export function useToast() {
  const [toasts, setToasts] = useState<ToastData[]>([])

  const addToast = useCallback((type: ToastType, title: string, message: string) => {
    const id = `toast-${Date.now()}-${Math.random().toString(36).substr(2, 9)}`
    
    setToasts((prev) => [...prev, { id, type, title, message }])
    
    return id
  }, [])

  const removeToast = useCallback((id: string) => {
    setToasts((prev) => prev.filter((toast) => toast.id !== id))
  }, [])

  const showError = useCallback((title: string, message: string) => {
    return addToast("error", title, message)
  }, [addToast])

  const showSuccess = useCallback((title: string, message: string) => {
    return addToast("success", title, message)
  }, [addToast])

  const showWarning = useCallback((title: string, message: string) => {
    return addToast("warning", title, message)
  }, [addToast])

  const showInfo = useCallback((title: string, message: string) => {
    return addToast("info", title, message)
  }, [addToast])

  const clearAll = useCallback(() => {
    setToasts([])
  }, [])

  return {
    toasts,
    addToast,
    removeToast,
    showError,
    showSuccess,
    showWarning,
    showInfo,
    clearAll,
  }
}