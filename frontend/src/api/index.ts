import { apiClient, unwrap } from './client'
import type {
  AuditLog, BatchStatus, DecisionType, InspectionSample, PackagingLine,
  PageResult, ProductionBatch, ReleaseDecision, User,
} from '../types/domain'

export interface PageParams { page?: number; pageSize?: number; search?: string }

export const authAPI = {
  login: (username: string, password: string) => unwrap<{ token: string; expiresAt: number; user: User }>(
    apiClient.post('/auth/login', { username, password }),
  ),
  me: () => unwrap<User>(apiClient.get('/auth/me')),
}

export const lineAPI = {
  list: (params: PageParams) => unwrap<PageResult<PackagingLine>>(apiClient.get('/lines', { params })),
  get: (id: number) => unwrap<PackagingLine>(apiClient.get(`/lines/${id}`)),
  create: (payload: Omit<PackagingLine, 'id' | 'createdAt' | 'updatedAt' | 'active' | 'batches'>) =>
    unwrap<PackagingLine>(apiClient.post('/lines', payload)),
  update: (id: number, payload: Partial<PackagingLine>) => unwrap<PackagingLine>(apiClient.patch(`/lines/${id}`, payload)),
}

export const batchAPI = {
  list: (params: PageParams & { status?: BatchStatus; lineId?: number }) => unwrap<PageResult<ProductionBatch>>(apiClient.get('/batches', { params })),
  get: (id: number) => unwrap<ProductionBatch>(apiClient.get(`/batches/${id}`)),
  create: (payload: Pick<ProductionBatch, 'batchNo' | 'specification' | 'responsibleTeam' | 'packagingLineId' | 'plannedQuantity' | 'producedQuantity'>) =>
    unwrap<ProductionBatch>(apiClient.post('/batches', payload)),
  update: (id: number, payload: Partial<ProductionBatch>) => unwrap<ProductionBatch>(apiClient.patch(`/batches/${id}`, payload)),
  transition: (id: number, status: BatchStatus, reason = '') => unwrap<ProductionBatch>(apiClient.post(`/batches/${id}/transition`, { status, reason })),
}

export const inspectionAPI = {
  list: (params: PageParams & { result?: string; batchId?: number }) => unwrap<PageResult<InspectionSample>>(apiClient.get('/inspections', { params })),
  create: (payload: Pick<InspectionSample, 'productionBatchId' | 'sampleCode' | 'samplingPosition' | 'inspectionItem' | 'acceptanceRange' | 'notes'>) =>
    unwrap<InspectionSample>(apiClient.post('/inspections', payload)),
  complete: (id: number, payload: { result: 'pass' | 'fail'; measuredValue: string; notes?: string; requestRetest?: boolean }) =>
    unwrap<InspectionSample>(apiClient.post(`/inspections/${id}/complete`, payload)),
  retest: (id: number, reason: string) => unwrap<InspectionSample>(apiClient.post(`/inspections/${id}/retest`, { reason })),
}

export const releaseAPI = {
  list: (params: PageParams & { decision?: DecisionType }) => unwrap<PageResult<ReleaseDecision>>(apiClient.get('/release-decisions', { params })),
  decide: (productionBatchId: number, decision: DecisionType, reason: string) =>
    unwrap<ReleaseDecision>(apiClient.post('/release-decisions', { productionBatchId, decision, reason })),
}

export const auditAPI = {
  list: (params: PageParams & { entityType?: string; actorId?: number }) => unwrap<PageResult<AuditLog>>(apiClient.get('/audit-logs', { params })),
}

