import type { ApiResponse } from './types'

// API 错误
export class ApiError extends Error {
  status: number
  code: number
  data: any
  constructor(status: number, code: number, message: string, data?: any) {
    super(message)
    this.status = status
    this.code = code
    this.data = data
  }
}

// 退出登录回调（由 auth store 注入，避免循环依赖）
let onUnauthorized: (() => void) | null = null
export function setUnauthorizedHandler(fn: () => void) {
  onUnauthorized = fn
}

// 通用请求
async function request<T>(
  method: string,
  url: string,
  body?: any,
  opts: { raw?: boolean } = {},
): Promise<T> {
  const init: RequestInit = {
    method,
    credentials: 'same-origin',
    headers: {},
  }
  if (body !== undefined) {
    if (body instanceof FormData) {
      init.body = body
    } else {
      ;(init.headers as Record<string, string>)['Content-Type'] = 'application/json'
      init.body = JSON.stringify(body)
    }
  }
  const resp = await fetch(url, init)
  if (opts.raw) {
    if (!resp.ok) {
      throw new ApiError(resp.status, 0, `request failed: ${resp.status}`)
    }
    return resp as unknown as T
  }
  // Guest-only 服务对不存在的接口返回纯文本 404（非 JSON），此处兜底避免 SyntaxError
  let json: ApiResponse<T>
  try {
    json = (await resp.json()) as ApiResponse<T>
  } catch {
    if (resp.status === 401 && onUnauthorized) onUnauthorized()
    throw new ApiError(resp.status, 0, `request failed: ${resp.status}`)
  }
  if (resp.status === 401) {
    if (onUnauthorized) onUnauthorized()
    throw new ApiError(401, json.code, json.message, json.data)
  }
  if (!resp.ok) {
    throw new ApiError(resp.status, json.code, json.message, json.data)
  }
  // 后端统一响应：HTTP 200 + code 字段。code 非 0 表示业务失败
  if (json.code !== 0) {
    throw new ApiError(resp.status, json.code, json.message, json.data)
  }
  return json.data
}

export function get<T>(url: string): Promise<T> {
  return request<T>('GET', url)
}

export function post<T>(url: string, body?: any): Promise<T> {
  return request<T>('POST', url, body)
}

export function put<T>(url: string, body?: any): Promise<T> {
  return request<T>('PUT', url, body)
}

export function del<T>(url: string): Promise<T> {
  return request<T>('DELETE', url)
}

// 原始响应（用于下载文件等）
export function getRaw(url: string, body?: any): Promise<Response> {
  if (body) {
    return request<Response>('POST', url, body, { raw: true })
  }
  return request<Response>('GET', url, undefined, { raw: true })
}

export function postRaw(url: string, body: any): Promise<Response> {
  return request<Response>('POST', url, body, { raw: true })
}
