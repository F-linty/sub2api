import { apiClient } from '../client'

export interface RTKCompressionFilterStats {
  filter: string
  hits: number
  before: number
  after: number
  bytes_saved: number
}

export interface RTKCompressionHitStat {
  path: string
  filter: string
  before: number
  after: number
  bytes_saved: number
  reference_hash?: string
  reference_path?: string
}

export interface RTKCompressionMissReasonStats {
  reason: string
  count: number
  bytes: number
}

export interface RTKCompressionMissStat {
  path: string
  before: number
  reason: string
  sample?: string
}

export interface RTKRequestPartStats {
  part: string
  count: number
  bytes: number
}

export interface RTKCompressionEvent {
  at: string
  account_id?: string | number
  before: number
  after: number
  bytes_saved: number
  hits: number
  filters?: string[]
  hit_stats?: RTKCompressionHitStat[]
  misses?: number
  miss_stats?: RTKCompressionMissStat[]
  request_parts?: RTKRequestPartStats[]
}

export interface CodexPromptCacheOutcomeStats {
  outcome: string
  count: number
}

export interface CodexPromptCacheSourceStats {
  source: string
  count: number
}

export interface CodexPromptCacheEvent {
  at: string
  account_id?: string | number
  model?: string
  outcome: string
  source?: string
  key_hash?: string
  account_type?: string
  compact?: boolean
}

export interface CodexPromptCacheSnapshot {
  started_at: string
  updated_at?: string
  requests: number
  injected: number
  already_present: number
  skipped: number
  by_outcome?: CodexPromptCacheOutcomeStats[]
  by_source?: CodexPromptCacheSourceStats[]
  recent_events?: CodexPromptCacheEvent[]
}

export interface CodexChainTransportStats {
  transport: string
  count: number
}

export interface CodexChainEvent {
  at: string
  account_id?: string | number
  account_type?: string
  model?: string
  transport: string
  compact?: boolean
  store_false: boolean
  stream_true: boolean
  has_previous_response_id: boolean
  previous_response_id_removed: boolean
  has_prompt_cache_key: boolean
  prompt_cache_key_hash?: string
  has_reasoning: boolean
  reasoning_effort?: string
  has_reasoning_encrypted_content: boolean
  header_session_id_hash?: string
  header_conversation_id_hash?: string
  upstream_session_id_hash?: string
  upstream_conversation_id_hash?: string
}

export interface CodexChainSnapshot {
  started_at: string
  updated_at?: string
  requests: number
  with_previous_response_id: number
  previous_response_id_removed: number
  store_false: number
  reasoning_encrypted_content: number
  with_prompt_cache_key: number
  by_transport?: CodexChainTransportStats[]
  recent_events?: CodexChainEvent[]
}

export interface RTKCompressionSnapshot {
  enabled: boolean
  min_bytes: number
  max_bytes: number
  started_at: string
  updated_at?: string
  requests: number
  hits: number
  misses: number
  missed_bytes: number
  before: number
  after: number
  bytes_saved: number
  save_ratio: number
  by_filter?: RTKCompressionFilterStats[]
  by_miss_reason?: RTKCompressionMissReasonStats[]
  by_request_part?: RTKRequestPartStats[]
  recent_events?: RTKCompressionEvent[]
  prompt_cache?: CodexPromptCacheSnapshot
  codex_chain?: CodexChainSnapshot
}

export async function getSnapshot(): Promise<RTKCompressionSnapshot> {
  const { data } = await apiClient.get<RTKCompressionSnapshot>('/admin/rtk-compression/snapshot')
  return data
}

export default {
  getSnapshot
}
