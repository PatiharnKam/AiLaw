"use client"

import { useState, useEffect } from "react"

type ModelType = "NORMAL" | "COT"

const MODEL_TYPE_KEY = "chatbot-model-type"
const DEFAULT_MODEL_TYPE: ModelType = "NORMAL"

export function useModelType() {
  const [modelType, setModelTypeState] = useState<ModelType>(DEFAULT_MODEL_TYPE)

  useEffect(() => {
    const savedModelType = localStorage.getItem(MODEL_TYPE_KEY)
    if (savedModelType === "NORMAL" || savedModelType === "COT") {
      setModelTypeState(savedModelType)
    }
  }, [])

  const setModelType = (newModelType: ModelType) => {
    setModelTypeState(newModelType)
    localStorage.setItem(MODEL_TYPE_KEY, newModelType)
  }

  return {
    modelType,
    setModelType
  }
}