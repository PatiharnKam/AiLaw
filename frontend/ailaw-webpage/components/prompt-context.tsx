"use client"
import { createContext, useContext, useState } from "react"

const PromptContext = createContext<any>(null)

export function PromptProvider({ children }: { children: React.ReactNode }) {
  const [prompt, setPrompt] = useState<string | null>(null)
  return (
    <PromptContext.Provider value={{ prompt, setPrompt }}>
      {children}
    </PromptContext.Provider>
  )
}

export function usePrompt() {
  return useContext(PromptContext)
}
