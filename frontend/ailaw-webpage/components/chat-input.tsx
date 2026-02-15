"use client"

import React, { useState, useEffect, useRef } from "react"
import Image from "next/image"

interface ChatInputProps {
  input: string
  setInput: (value: string) => void
  handleSubmit: (e: React.FormEvent) => void
  isSending?: boolean
  isDark: boolean
  modelType: "NORMAL" | "COT"
  setModelType: (type: "NORMAL" | "COT") => void
  placeholder?: string
}

export function ChatInput({
  input,
  setInput,
  handleSubmit,
  isSending = false,
  isDark,
  modelType,
  setModelType,
  placeholder = "Type your message..."
}: ChatInputProps) {
  const [isDropdownOpen, setIsDropdownOpen] = useState(false)
  const dropdownRef = useRef<HTMLDivElement>(null)

  // Close dropdown when clicking outside
  useEffect(() => {
    const handleClickOutside = (event: MouseEvent) => {
      if (dropdownRef.current && !dropdownRef.current.contains(event.target as Node)) {
        setIsDropdownOpen(false)
      }
    }

    if (isDropdownOpen) {
      document.addEventListener("mousedown", handleClickOutside)
      return () => document.removeEventListener("mousedown", handleClickOutside)
    }
  }, [isDropdownOpen])

  const getDisplayName = (type: "NORMAL" | "COT") => {
    return type === "COT" ? "REASONING" : "NORMAL"
  }

  return (
    <div className="p-4">
      <form onSubmit={handleSubmit} className="mx-auto max-w-3xl">
        <div className={`rounded-3xl overflow-hidden ${
          isDark
            ? "border border-[#EFF4FF]/30 bg-[#FFFFFF]/5"
            : "border-2 border-slate-200 bg-white shadow-sm"
        }`}>
          {/* Textarea Section */}
          <textarea
            ref={(el) => {
              if (el) {
                el.style.height = 'auto';
                const newHeight = Math.min(el.scrollHeight, 200)
                el.style.height = newHeight + 'px'
              }
            }}
            value={input}
            onChange={(e) => {
              setInput(e.target.value);
              const newHeight = Math.min(e.target.scrollHeight, 200)
              e.target.style.height = newHeight + 'px'
            }}
            onKeyDown={(e) => {
              if (e.key === "Enter" && !e.shiftKey) {
                e.preventDefault();
                handleSubmit(e);
              }
            }}
            disabled={isSending}
            placeholder={isSending ? "Sending..." : placeholder}
            rows={1}
            className={`custom-scroll w-full px-6 pt-4 pb-3 resize-none overflow-y-auto focus:outline-none disabled:opacity-50 disabled:cursor-not-allowed max-h-[200px] bg-transparent ${
              isDark
                ? "text-white placeholder-[#EFF4FF]/30"
                : "text-slate-900 placeholder-slate-400"
            }`}
          ></textarea>

          {/* Bottom Bar - Model Selector & Send Button */}
          <div className={`relative flex items-center justify-end gap-2 px-4 pb-3 pt-1`}>
            {/* Model Type Dropdown */}
            <div className="relative z-50" ref={dropdownRef}>
              <button
                type="button"
                onClick={() => setIsDropdownOpen(!isDropdownOpen)}
                disabled={isSending}
                className={`flex items-center gap-2 rounded-lg px-3 py-1.5 text-sm font-medium transition-all disabled:opacity-50 disabled:cursor-not-allowed ${
                  isDark
                    ? "text-slate-300 hover:bg-[#FFFFFF]/10"
                    : "text-slate-700 hover:bg-gray-100"
                }`}
              >
                <span>{getDisplayName(modelType)}</span>
                <svg 
                  className={`h-4 w-4 transition-transform ${isDropdownOpen ? "rotate-180" : ""}`} 
                  fill="none" 
                  stroke="currentColor" 
                  viewBox="0 0 24 24"
                >
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M19 9l-7 7-7-7" />
                </svg>
              </button>

              {/* Dropdown Menu - Positioned Outside */}
              {isDropdownOpen && (
                <div 
                  className={`fixed w-64 rounded-xl shadow-2xl border overflow-hidden ${
                    isDark 
                      ? "bg-[#2A2A2A] border-[#3C3C3C]" 
                      : "bg-white border-gray-200"
                  }`}
                  style={{
                    bottom: dropdownRef.current 
                      ? `${window.innerHeight - dropdownRef.current.getBoundingClientRect().top + 8}px`
                      : 'auto',
                    right: dropdownRef.current
                      ? `${window.innerWidth - dropdownRef.current.getBoundingClientRect().right}px`
                      : 'auto'
                  }}
                >
                  <button
                    type="button"
                    onClick={() => {
                      setModelType("NORMAL")
                      setIsDropdownOpen(false)
                    }}
                    className={`flex w-full items-start gap-3 px-4 py-3 text-sm transition-colors ${
                      modelType === "NORMAL"
                        ? isDark
                          ? "bg-[#3C3C3C]"
                          : "bg-[#E8EAF3]"
                        : ""
                    } ${
                      isDark 
                        ? "text-slate-200 hover:bg-[#3C3C3C]" 
                        : "text-slate-900 hover:bg-gray-100"
                    }`}
                  >
                    <div className="flex-shrink-0 mt-0.5">
                      <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M13 10V3L4 14h7v7l9-11h-7z" />
                      </svg>
                    </div>
                    <div className="flex-1 text-left">
                      <div className="font-semibold">NORMAL</div>
                      <div className={`text-xs mt-0.5 ${isDark ? "text-slate-400" : "text-slate-500"}`}>
                        Fast and efficient responses
                      </div>
                    </div>
                    {modelType === "NORMAL" && (
                      <svg className="h-5 w-5 text-blue-500 flex-shrink-0" fill="currentColor" viewBox="0 0 24 24">
                        <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/>
                      </svg>
                    )}
                  </button>
                  
                  <button
                    type="button"
                    onClick={() => {
                      setModelType("COT")
                      setIsDropdownOpen(false)
                    }}
                    className={`flex w-full items-start gap-3 px-4 py-3 text-sm transition-colors ${
                      modelType === "COT"
                        ? isDark
                          ? "bg-[#3C3C3C]"
                          : "bg-[#E8EAF3]"
                        : ""
                    } ${
                      isDark 
                        ? "text-slate-200 hover:bg-[#3C3C3C]" 
                        : "text-slate-900 hover:bg-gray-100"
                    }`}
                  >
                    <div className="flex-shrink-0 mt-0.5">
                      <svg className="h-5 w-5" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                        <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={2} d="M9 3v2m6-2v2M9 19v2m6-2v2M5 9H3m2 6H3m18-6h-2m2 6h-2M7 19h10a2 2 0 002-2V7a2 2 0 00-2-2H7a2 2 0 00-2 2v10a2 2 0 002 2zM9 9h6v6H9V9z" />
                      </svg>
                    </div>
                    <div className="flex-1 text-left">
                      <div className="font-semibold">REASONING</div>
                      <div className={`text-xs mt-0.5 ${isDark ? "text-slate-400" : "text-slate-500"}`}>
                        Chain of Thought reasoning
                      </div>
                    </div>
                    {modelType === "COT" && (
                      <svg className="h-5 w-5 text-blue-500 flex-shrink-0" fill="currentColor" viewBox="0 0 24 24">
                        <path d="M9 16.17L4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41z"/>
                      </svg>
                    )}
                  </button>
                </div>
              )}
            </div>

            {/* Send Button */}
            <button
              type="submit"
              disabled={!input.trim() || isSending}
              className={`flex h-8 w-8 items-center justify-center rounded-xl transition-all disabled:opacity-50 disabled:cursor-not-allowed ${
                input.trim() && !isSending
                  ? isDark 
                    ? "bg-[#F3F3F3] hover:bg-white" 
                    : "bg-[#485BA9] hover:bg-[#3d4d8f]"
                  : isDark
                    ? "bg-[#FFFFFF]/10"
                    : "bg-gray-200"
              }`}
            >
              <Image
                src="/send-logo.svg"
                width={18}
                height={18}
                alt="Send"
                className={isDark ? "" : "invert"}
              />
            </button>
          </div>
        </div>
      </form>
    </div>
  )
}