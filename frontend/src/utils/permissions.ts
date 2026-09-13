import type { User } from '../types/domain'

export const can = (user: User | null, permission: string) => Boolean(user?.permissions.includes(permission))

export const roleLabels = {
  admin: '质量管理员', inspector: '检验员', approver: '放行审批员', operator: '产线操作员', viewer: '只读用户',
} as const

