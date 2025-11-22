"use client"

import { useEffect } from "react"
import { useRouter } from "next/navigation"

export default function Home() {
  const router = useRouter()

  useEffect(() => {
    router.push("/welcome")
  }, [router])

  return (
    <div className="flex min-h-screen items-center justify-center bg-slate-950">
      <div className="text-slate-400">Loading...</div>
    </div>
  )
}
