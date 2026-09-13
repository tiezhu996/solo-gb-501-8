import axios, { AxiosError } from 'axios'
import { message } from 'antd'
import type { APIEnvelope } from '../types/domain'

export const apiClient = axios.create({ baseURL: '/api', timeout: 15000 })

apiClient.interceptors.request.use((config) => {
  const token = localStorage.getItem('sterile.accessToken')
  if (token) config.headers.Authorization = `Bearer ${token}`
  return config
})

apiClient.interceptors.response.use(
  (response) => response,
  async (error: AxiosError<{ error?: { message?: string; code?: string } }>) => {
    if (error.response?.status === 401) {
      localStorage.removeItem('sterile.accessToken')
      localStorage.removeItem('sterile.user')
      if (location.pathname !== '/login') location.assign('/login')
    }
    const text = error.response?.data?.error?.message || '请求失败，请稍后重试'
    void message.error(text)
    return Promise.reject(error)
  },
)

export async function unwrap<T>(request: Promise<{ data: APIEnvelope<T> }>): Promise<T> {
  const response = await request
  return response.data.data
}

