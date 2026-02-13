export const ErrorCodes = {
  INVALID_REQUEST: "10000",
  PROMPT_TOO_LONG: "10001",
  QUOTA_EXCEEDED: "10002",
  UNAUTHORIZED: "10003",
  INTERNAL_SERVER: "99999",
} as const

export interface ErrorInfo {
  title: string
  message: string
  code: string
}

export function mapErrorCode(code: string): ErrorInfo {
  switch (code) {
    case ErrorCodes.INVALID_REQUEST:
      return {
        title: "คำขอไม่ถูกต้อง",
        message: "กรุณาตรวจสอบข้อมูลและลองใหม่อีกครั้ง",
        code,
      }
    
    case ErrorCodes.PROMPT_TOO_LONG:
      return {
        title: "ข้อความยาวเกินไป",
        message: "กรุณาลดความยาวของข้อความและลองใหม่อีกครั้ง",
        code,
      }
    
    case ErrorCodes.QUOTA_EXCEEDED:
      return {
        title: "ใช้งานเกินโควต้า",
        message: "คุณได้ใช้งานครบโควต้าแล้ว กรุณารอวันถัดไป",
        code,
      }
    
    case ErrorCodes.UNAUTHORIZED:
      return {
        title: "ไม่มีสิทธิ์เข้าถึง",
        message: "กรุณาเข้าสู่ระบบใหม่อีกครั้ง",
        code,
      }
    
    case ErrorCodes.INTERNAL_SERVER:
      return {
        title: "เกิดข้อผิดพลาดภายในระบบ",
        message: "กรุณาลองใหม่อีกครั้งในภายหลัง",
        code,
      }
    
    default:
      return {
        title: "เกิดข้อผิดพลาด",
        message: code || "กรุณาลองใหม่อีกครั้ง",
        code: code || "UNKNOWN",
      }
  }
}

export function parseApiError(error: any): ErrorInfo {
  if (error?.code) {
    return mapErrorCode(error.code)
  }
  
  if (typeof error === "string") {
    if (Object.values(ErrorCodes).includes(error as any)) {
      return mapErrorCode(error)
    }
  
    return {
      title: "เกิดข้อผิดพลาด",
      message: error,
      code: "UNKNOWN",
    }
  }
  
  const message = error?.message || error?.error?.message || "เกิดข้อผิดพลาดที่ไม่ทราบสาเหตุ"
  
  return {
    title: "เกิดข้อผิดพลาด",
    message,
    code: "UNKNOWN",
  }
}

export function parseWSError(wsResponse: any): ErrorInfo {
  if (wsResponse?.type === "error" && wsResponse?.error) {
    return parseApiError(wsResponse.error)
  }
  
  return parseApiError(wsResponse)
}