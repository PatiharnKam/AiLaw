"use client"

import { useState, useEffect } from "react"

interface FeedbackDetailProps {
  isOpen: boolean
  onClose: () => void
  onSubmit: (detail: string) => void
  feedbackType: "like" | "dislike" | null
  isDark: boolean
}

export function FeedbackDetail({ 
  isOpen, 
  onClose, 
  onSubmit, 
  feedbackType,
  isDark 
}: FeedbackDetailProps) {
  const [detail, setDetail] = useState("")

  useEffect(() => {
    if (isOpen) {
      setDetail("")
    }
  }, [isOpen])

  useEffect(() => {
    const handleEscape = (e: KeyboardEvent) => {
      if (e.key === "Escape" && isOpen) {
        onClose()
      }
    }

    document.addEventListener("keydown", handleEscape)
    return () => document.removeEventListener("keydown", handleEscape)
  }, [isOpen, onClose])

  const handleSubmit = () => {
    onSubmit(detail.trim())
    onClose()
  }

  const handleSkip = () => {
    onSubmit("")
    onClose()
  }

  if (!isOpen) return null

  return (
    <div className="fixed inset-0 z-[10000] flex items-center justify-center p-4">
      {/* Backdrop */}
      <div 
        className="absolute inset-0 bg-black/40"
        onClick={onClose}
      />

      {/* Modal */}
      <div 
        className={`relative w-full max-w-md rounded-xl shadow-lg ${
          isDark ? "bg-[#2A2A2A]" : "bg-white"
        }`}
      >
        {/* Header */}
        <div className="flex items-center justify-between p-5 pb-4">
          <h2 className={`text-base font-medium ${
            isDark ? "text-white" : "text-slate-900"
          }`}>
            {feedbackType === "like" ? "ชอบคำตอบนี้" : "ไม่ชอบคำตอบนี้"}
          </h2>
          <button
            onClick={onClose}
            className={`p-1 rounded transition-colors ${
              isDark 
                ? "hover:bg-white/10 text-slate-400" 
                : "hover:bg-slate-100 text-slate-600"
            }`}
          >
            <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M6 18L18 6M6 6l12 12" />
            </svg>
          </button>
        </div>

        {/* Body */}
        <div className="px-5 pb-5">
          <textarea
            value={detail}
            onChange={(e) => setDetail(e.target.value)}
            placeholder={
              feedbackType === "like"
                ? "บอกเราได้ไหมว่าชอบตรงไหน? (optional)"
                : "บอกเราได้ไหมว่าต้องปรับปรุงตรงไหน? (optional)"
            }
            rows={4}
            autoFocus
            className={`w-full px-3 py-2.5 rounded-lg text-sm resize-none focus:outline-none focus:ring-1 transition-all ${
              isDark
                ? "bg-white/5 text-white placeholder-slate-500 focus:ring-white/20"
                : "bg-slate-50 text-slate-900 placeholder-slate-400 focus:ring-slate-300"
            }`}
            maxLength={500}
          />
          <div className={`mt-1.5 text-xs text-right ${
            isDark ? "text-slate-500" : "text-slate-400"
          }`}>
            {detail.length}/500
          </div>
        </div>

        {/* Footer */}
        <div className="flex gap-2 px-5 pb-5">
          <button
            onClick={handleSkip}
            className={`flex-1 px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
              isDark
                ? "hover:bg-white/5 text-slate-400"
                : "hover:bg-slate-100 text-slate-600"
            }`}
          >
            pass
          </button>
          <button
            onClick={handleSubmit}
            className={`flex-1 px-4 py-2 rounded-lg text-sm font-medium transition-colors ${
              feedbackType === "like"
                ? "bg-green-500 hover:bg-green-600 text-white"
                : "bg-red-500 hover:bg-red-600 text-white"
            }`}
          >
            send
          </button>
        </div>
      </div>
    </div>
  )
}