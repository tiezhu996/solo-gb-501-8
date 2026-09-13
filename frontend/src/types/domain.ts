export type BatchStatus = 'draft' | 'running' | 'hold' | 'rework' | 'released'
export type DecisionType = 'release' | 'quarantine' | 'rework'
export type Role = 'admin' | 'inspector' | 'approver' | 'operator' | 'viewer'

export interface BaseEntity {
  id: number
  createdAt: string
  updatedAt: string
}

export interface PackagingLine extends BaseEntity {
  code: string
  name: string
  team: string
  equipmentStatus: 'ready' | 'running' | 'maintenance' | 'fault'
  location: string
  active: boolean
  batches?: ProductionBatch[]
}

export interface ProductionBatch extends BaseEntity {
  batchNo: string
  specification: string
  status: BatchStatus
  responsibleTeam: string
  packagingLineId: number
  packagingLine?: PackagingLine
  plannedQuantity: number
  producedQuantity: number
  startedAt?: string
  completedAt?: string
  holdReason?: string
  inspections?: InspectionSample[]
  decisions?: ReleaseDecision[]
}

export interface InspectionSample extends BaseEntity {
  productionBatchId: number
  productionBatch?: ProductionBatch
  sampleCode: string
  samplingPosition: string
  inspectionItem: string
  result: 'pending' | 'pass' | 'fail'
  measuredValue?: string
  acceptanceRange: string
  retestStatus: 'none' | 'requested' | 'completed'
  inspectorId?: number
  inspectorName?: string
  inspectedAt?: string
  notes?: string
}

export interface ReleaseDecision extends BaseEntity {
  productionBatchId: number
  productionBatch?: ProductionBatch
  decision: DecisionType
  approverId: number
  approverName: string
  reason: string
  effectiveAt: string
  inspectionSummary: string
}

export interface AuditLog {
  id: number
  createdAt: string
  requestId: string
  actorId: number
  actorName: string
  action: string
  entityType: string
  entityId: number
  beforeState: string
  afterState: string
  ipAddress: string
}

export interface User {
  id: number
  username: string
  displayName: string
  role: Role
  permissions: string[]
}

export interface PageResult<T> {
  items: T[]
  total: number
  page: number
  pageSize: number
}

export interface APIEnvelope<T> { data: T; requestId: string }

