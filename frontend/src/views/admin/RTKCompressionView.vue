<template>
  <AppLayout>
    <div class="space-y-6">
      <div class="flex flex-wrap items-center justify-end gap-3">
          <label class="inline-flex items-center gap-2 text-sm text-gray-600 dark:text-gray-300">
            <input v-model="autoRefresh" type="checkbox" class="rounded border-gray-300 text-primary-600 focus:ring-primary-500" />
            <span>{{ t('admin.rtkCompression.autoRefresh') }}</span>
          </label>
          <button type="button" class="btn btn-primary btn-sm" :disabled="loading" @click="loadSnapshot">
            {{ loading ? t('common.loading') : t('admin.rtkCompression.refresh') }}
          </button>
      </div>

      <div v-if="errorMessage" class="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700 dark:border-red-900/60 dark:bg-red-950/30 dark:text-red-300">
        {{ errorMessage }}
      </div>

      <div class="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-5">
        <div v-for="card in summaryCards" :key="card.label" class="card p-5">
          <p class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ card.label }}</p>
          <p class="mt-2 text-2xl font-semibold text-gray-950 dark:text-white">{{ card.value }}</p>
          <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ card.hint }}</p>
        </div>
      </div>

      <div class="card p-5">
        <div class="flex flex-wrap items-center justify-between gap-3">
          <div>
            <h2 class="text-base font-semibold text-gray-950 dark:text-white">{{ t('admin.rtkCompression.config') }}</h2>
            <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">
              {{ snapshot?.enabled ? t('admin.rtkCompression.enabled') : t('admin.rtkCompression.disabled') }}
            </p>
          </div>
          <div class="grid grid-cols-1 gap-3 text-sm sm:grid-cols-3">
            <div>
              <span class="text-gray-500 dark:text-gray-400">{{ t('admin.rtkCompression.minBytes') }}</span>
              <p class="font-medium text-gray-950 dark:text-white">{{ formatBytes(snapshot?.min_bytes ?? 0) }}</p>
            </div>
            <div>
              <span class="text-gray-500 dark:text-gray-400">{{ t('admin.rtkCompression.maxBytes') }}</span>
              <p class="font-medium text-gray-950 dark:text-white">{{ formatBytes(snapshot?.max_bytes ?? 0) }}</p>
            </div>
            <div>
              <span class="text-gray-500 dark:text-gray-400">{{ t('admin.rtkCompression.lastUpdated') }}</span>
              <p class="font-medium text-gray-950 dark:text-white">{{ formatDateTime(snapshot?.updated_at) || t('admin.rtkCompression.noHits') }}</p>
            </div>
          </div>
        </div>
      </div>

      <div class="card overflow-hidden">
        <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
          <h2 class="text-base font-semibold text-gray-950 dark:text-white">{{ t('admin.rtkCompression.filters.title') }}</h2>
        </div>
        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-100 text-sm dark:divide-dark-700">
            <thead class="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500 dark:bg-dark-800/70 dark:text-gray-400">
              <tr>
                <th class="px-5 py-3">{{ t('admin.rtkCompression.filters.filter') }}</th>
                <th class="px-5 py-3 text-right">{{ t('admin.rtkCompression.filters.hits') }}</th>
                <th class="px-5 py-3 text-right">{{ t('admin.rtkCompression.filters.before') }}</th>
                <th class="px-5 py-3 text-right">{{ t('admin.rtkCompression.filters.after') }}</th>
                <th class="px-5 py-3 text-right">{{ t('admin.rtkCompression.filters.saved') }}</th>
                <th class="px-5 py-3 text-right">{{ t('admin.rtkCompression.filters.ratio') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="item in filterRows" :key="item.filter" class="text-gray-700 dark:text-gray-200">
                <td class="px-5 py-3 font-medium">{{ item.filter }}</td>
                <td class="px-5 py-3 text-right">{{ formatNumber(item.hits) }}</td>
                <td class="px-5 py-3 text-right">{{ formatBytes(item.before) }}</td>
                <td class="px-5 py-3 text-right">{{ formatBytes(item.after) }}</td>
                <td class="px-5 py-3 text-right text-green-600 dark:text-green-400">{{ formatBytes(item.bytes_saved) }}</td>
                <td class="px-5 py-3 text-right">{{ formatPercent(safeRatio(item.bytes_saved, item.before)) }}</td>
              </tr>
              <tr v-if="filterRows.length === 0">
                <td colspan="6" class="px-5 py-8 text-center text-sm text-gray-500 dark:text-gray-400">
                  {{ t('admin.rtkCompression.filters.empty') }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <div class="card overflow-hidden">
        <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
          <h2 class="text-base font-semibold text-gray-950 dark:text-white">{{ t('admin.rtkCompression.misses.title') }}</h2>
        </div>
        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-100 text-sm dark:divide-dark-700">
            <thead class="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500 dark:bg-dark-800/70 dark:text-gray-400">
              <tr>
                <th class="px-5 py-3">{{ t('admin.rtkCompression.misses.reason') }}</th>
                <th class="px-5 py-3 text-right">{{ t('admin.rtkCompression.misses.count') }}</th>
                <th class="px-5 py-3 text-right">{{ t('admin.rtkCompression.misses.bytes') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="item in missedRows" :key="item.reason" class="text-gray-700 dark:text-gray-200">
                <td class="px-5 py-3 font-medium">{{ formatMissReason(item.reason) }}</td>
                <td class="px-5 py-3 text-right">{{ formatNumber(item.count) }}</td>
                <td class="px-5 py-3 text-right">{{ formatBytes(item.bytes) }}</td>
              </tr>
              <tr v-if="missedRows.length === 0">
                <td colspan="3" class="px-5 py-8 text-center text-sm text-gray-500 dark:text-gray-400">
                  {{ t('admin.rtkCompression.misses.empty') }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <div class="card overflow-hidden">
        <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
          <h2 class="text-base font-semibold text-gray-950 dark:text-white">{{ t('admin.rtkCompression.requestParts.title') }}</h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.rtkCompression.requestParts.description') }}</p>
        </div>
        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-100 text-sm dark:divide-dark-700">
            <thead class="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500 dark:bg-dark-800/70 dark:text-gray-400">
              <tr>
                <th class="px-5 py-3">{{ t('admin.rtkCompression.requestParts.part') }}</th>
                <th class="px-5 py-3 text-right">{{ t('admin.rtkCompression.requestParts.count') }}</th>
                <th class="px-5 py-3 text-right">{{ t('admin.rtkCompression.requestParts.bytes') }}</th>
                <th class="px-5 py-3 text-right">{{ t('admin.rtkCompression.requestParts.ratio') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="item in requestPartRows" :key="item.part" class="text-gray-700 dark:text-gray-200">
                <td class="px-5 py-3 font-medium">{{ formatRequestPart(item.part) }}</td>
                <td class="px-5 py-3 text-right">{{ formatNumber(item.count) }}</td>
                <td class="px-5 py-3 text-right">{{ formatBytes(item.bytes) }}</td>
                <td class="px-5 py-3 text-right">{{ formatPercent(safeRatio(item.bytes, snapshot?.before ?? 0)) }}</td>
              </tr>
              <tr v-if="requestPartRows.length === 0">
                <td colspan="4" class="px-5 py-8 text-center text-sm text-gray-500 dark:text-gray-400">
                  {{ t('admin.rtkCompression.requestParts.empty') }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <div class="card overflow-hidden">
        <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
          <h2 class="text-base font-semibold text-gray-950 dark:text-white">{{ t('admin.rtkCompression.promptCache.title') }}</h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.rtkCompression.promptCache.description') }}</p>
        </div>
        <div class="grid grid-cols-1 gap-4 border-b border-gray-100 p-5 dark:border-dark-700 md:grid-cols-4">
          <div>
            <p class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('admin.rtkCompression.promptCache.requests') }}</p>
            <p class="mt-2 text-xl font-semibold text-gray-950 dark:text-white">{{ formatNumber(promptCache?.requests ?? 0) }}</p>
          </div>
          <div>
            <p class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('admin.rtkCompression.promptCache.injected') }}</p>
            <p class="mt-2 text-xl font-semibold text-green-600 dark:text-green-400">{{ formatNumber(promptCache?.injected ?? 0) }}</p>
          </div>
          <div>
            <p class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('admin.rtkCompression.promptCache.alreadyPresent') }}</p>
            <p class="mt-2 text-xl font-semibold text-gray-950 dark:text-white">{{ formatNumber(promptCache?.already_present ?? 0) }}</p>
          </div>
          <div>
            <p class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('admin.rtkCompression.promptCache.skipped') }}</p>
            <p class="mt-2 text-xl font-semibold text-amber-600 dark:text-amber-400">{{ formatNumber(promptCache?.skipped ?? 0) }}</p>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ formatDateTime(promptCache?.updated_at) || t('admin.rtkCompression.promptCache.noEvents') }}</p>
          </div>
        </div>
        <div class="grid grid-cols-1 gap-0 lg:grid-cols-2">
          <div class="overflow-x-auto border-b border-gray-100 dark:border-dark-700 lg:border-b-0 lg:border-r">
            <table class="min-w-full divide-y divide-gray-100 text-sm dark:divide-dark-700">
              <thead class="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500 dark:bg-dark-800/70 dark:text-gray-400">
                <tr>
                  <th colspan="2" class="px-5 py-3 normal-case tracking-normal text-gray-700 dark:text-gray-200">{{ t('admin.rtkCompression.promptCache.byOutcome') }}</th>
                </tr>
                <tr>
                  <th class="px-5 py-3">{{ t('admin.rtkCompression.promptCache.outcome') }}</th>
                  <th class="px-5 py-3 text-right">{{ t('admin.rtkCompression.promptCache.count') }}</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                <tr v-for="item in promptCacheOutcomes" :key="item.outcome" class="text-gray-700 dark:text-gray-200">
                  <td class="px-5 py-3 font-medium">{{ formatPromptCacheOutcome(item.outcome) }}</td>
                  <td class="px-5 py-3 text-right">{{ formatNumber(item.count) }}</td>
                </tr>
                <tr v-if="promptCacheOutcomes.length === 0">
                  <td colspan="2" class="px-5 py-8 text-center text-sm text-gray-500 dark:text-gray-400">{{ t('admin.rtkCompression.promptCache.empty') }}</td>
                </tr>
              </tbody>
            </table>
          </div>
          <div class="overflow-x-auto">
            <table class="min-w-full divide-y divide-gray-100 text-sm dark:divide-dark-700">
              <thead class="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500 dark:bg-dark-800/70 dark:text-gray-400">
                <tr>
                  <th colspan="2" class="px-5 py-3 normal-case tracking-normal text-gray-700 dark:text-gray-200">{{ t('admin.rtkCompression.promptCache.bySource') }}</th>
                </tr>
                <tr>
                  <th class="px-5 py-3">{{ t('admin.rtkCompression.promptCache.source') }}</th>
                  <th class="px-5 py-3 text-right">{{ t('admin.rtkCompression.promptCache.count') }}</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
                <tr v-for="item in promptCacheSources" :key="item.source" class="text-gray-700 dark:text-gray-200">
                  <td class="px-5 py-3 font-medium">{{ formatPromptCacheSource(item.source) }}</td>
                  <td class="px-5 py-3 text-right">{{ formatNumber(item.count) }}</td>
                </tr>
                <tr v-if="promptCacheSources.length === 0">
                  <td colspan="2" class="px-5 py-8 text-center text-sm text-gray-500 dark:text-gray-400">{{ t('admin.rtkCompression.promptCache.empty') }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
        <div class="overflow-x-auto border-t border-gray-100 dark:border-dark-700">
          <table class="min-w-full divide-y divide-gray-100 text-sm dark:divide-dark-700">
            <thead class="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500 dark:bg-dark-800/70 dark:text-gray-400">
              <tr>
                <th class="px-5 py-3">{{ t('admin.rtkCompression.promptCache.time') }}</th>
                <th class="px-5 py-3">{{ t('admin.rtkCompression.promptCache.account') }}</th>
                <th class="px-5 py-3">{{ t('admin.rtkCompression.promptCache.model') }}</th>
                <th class="px-5 py-3">{{ t('admin.rtkCompression.promptCache.outcome') }}</th>
                <th class="px-5 py-3">{{ t('admin.rtkCompression.promptCache.source') }}</th>
                <th class="px-5 py-3">{{ t('admin.rtkCompression.promptCache.accountType') }}</th>
                <th class="px-5 py-3">{{ t('admin.rtkCompression.promptCache.compact') }}</th>
                <th class="px-5 py-3">{{ t('admin.rtkCompression.promptCache.keyHash') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="event in promptCacheEvents" :key="`${event.at}-${event.account_id}-${event.outcome}`" class="text-gray-700 dark:text-gray-200">
                <td class="whitespace-nowrap px-5 py-3">{{ formatDateTime(event.at) }}</td>
                <td class="whitespace-nowrap px-5 py-3">{{ event.account_id || '-' }}</td>
                <td class="whitespace-nowrap px-5 py-3">{{ event.model || '-' }}</td>
                <td class="whitespace-nowrap px-5 py-3">{{ formatPromptCacheOutcome(event.outcome) }}</td>
                <td class="whitespace-nowrap px-5 py-3">{{ event.source ? formatPromptCacheSource(event.source) : '-' }}</td>
                <td class="whitespace-nowrap px-5 py-3">{{ event.account_type || '-' }}</td>
                <td class="whitespace-nowrap px-5 py-3">{{ event.compact ? 'yes' : 'no' }}</td>
                <td class="max-w-[240px] truncate px-5 py-3 font-mono text-xs">{{ event.key_hash || '-' }}</td>
              </tr>
              <tr v-if="promptCacheEvents.length === 0">
                <td colspan="8" class="px-5 py-8 text-center text-sm text-gray-500 dark:text-gray-400">{{ t('admin.rtkCompression.promptCache.noEvents') }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <div class="card overflow-hidden">
        <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
          <h2 class="text-base font-semibold text-gray-950 dark:text-white">{{ t('admin.rtkCompression.codexChain.title') }}</h2>
          <p class="mt-1 text-sm text-gray-500 dark:text-gray-400">{{ t('admin.rtkCompression.codexChain.description') }}</p>
        </div>
        <div class="grid grid-cols-1 gap-4 border-b border-gray-100 p-5 dark:border-dark-700 md:grid-cols-5">
          <div>
            <p class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('admin.rtkCompression.codexChain.requests') }}</p>
            <p class="mt-2 text-xl font-semibold text-gray-950 dark:text-white">{{ formatNumber(codexChain?.requests ?? 0) }}</p>
          </div>
          <div>
            <p class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('admin.rtkCompression.codexChain.prevPresent') }}</p>
            <p class="mt-2 text-xl font-semibold text-gray-950 dark:text-white">{{ formatNumber(codexChain?.with_previous_response_id ?? 0) }}</p>
          </div>
          <div>
            <p class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('admin.rtkCompression.codexChain.prevRemoved') }}</p>
            <p class="mt-2 text-xl font-semibold text-amber-600 dark:text-amber-400">{{ formatNumber(codexChain?.previous_response_id_removed ?? 0) }}</p>
          </div>
          <div>
            <p class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('admin.rtkCompression.codexChain.storeFalse') }}</p>
            <p class="mt-2 text-xl font-semibold text-green-600 dark:text-green-400">{{ formatNumber(codexChain?.store_false ?? 0) }}</p>
          </div>
          <div>
            <p class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('admin.rtkCompression.codexChain.encrypted') }}</p>
            <p class="mt-2 text-xl font-semibold text-gray-950 dark:text-white">{{ formatNumber(codexChain?.reasoning_encrypted_content ?? 0) }}</p>
            <p class="mt-1 text-xs text-gray-500 dark:text-gray-400">{{ formatNumber(codexChain?.with_prompt_cache_key ?? 0) }} {{ t('admin.rtkCompression.codexChain.promptCache') }}</p>
          </div>
        </div>
        <div class="overflow-x-auto border-b border-gray-100 dark:border-dark-700">
          <table class="min-w-full divide-y divide-gray-100 text-sm dark:divide-dark-700">
            <thead class="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500 dark:bg-dark-800/70 dark:text-gray-400">
              <tr>
                <th colspan="2" class="px-5 py-3 normal-case tracking-normal text-gray-700 dark:text-gray-200">{{ t('admin.rtkCompression.codexChain.byTransport') }}</th>
              </tr>
              <tr>
                <th class="px-5 py-3">{{ t('admin.rtkCompression.codexChain.transport') }}</th>
                <th class="px-5 py-3 text-right">{{ t('admin.rtkCompression.codexChain.count') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="item in codexChainTransports" :key="item.transport" class="text-gray-700 dark:text-gray-200">
                <td class="px-5 py-3 font-medium">{{ item.transport || '-' }}</td>
                <td class="px-5 py-3 text-right">{{ formatNumber(item.count) }}</td>
              </tr>
              <tr v-if="codexChainTransports.length === 0">
                <td colspan="2" class="px-5 py-8 text-center text-sm text-gray-500 dark:text-gray-400">{{ t('admin.rtkCompression.codexChain.empty') }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-100 text-sm dark:divide-dark-700">
            <thead class="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500 dark:bg-dark-800/70 dark:text-gray-400">
              <tr>
                <th class="px-5 py-3">{{ t('admin.rtkCompression.codexChain.time') }}</th>
                <th class="px-5 py-3">{{ t('admin.rtkCompression.codexChain.account') }}</th>
                <th class="px-5 py-3">{{ t('admin.rtkCompression.codexChain.model') }}</th>
                <th class="px-5 py-3">{{ t('admin.rtkCompression.codexChain.transport') }}</th>
                <th class="px-5 py-3">{{ t('admin.rtkCompression.codexChain.storeFalse') }}</th>
                <th class="px-5 py-3">{{ t('admin.rtkCompression.codexChain.previous') }}</th>
                <th class="px-5 py-3">{{ t('admin.rtkCompression.codexChain.reasoning') }}</th>
                <th class="px-5 py-3">{{ t('admin.rtkCompression.codexChain.session') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="event in codexChainEvents" :key="`${event.at}-${event.account_id}-${event.model}-${event.transport}`" class="align-top text-gray-700 dark:text-gray-200">
                <td class="whitespace-nowrap px-5 py-3">{{ formatDateTime(event.at) }}</td>
                <td class="whitespace-nowrap px-5 py-3">{{ event.account_id || '-' }}</td>
                <td class="whitespace-nowrap px-5 py-3">{{ event.model || '-' }}</td>
                <td class="whitespace-nowrap px-5 py-3">{{ event.transport || '-' }}</td>
                <td class="whitespace-nowrap px-5 py-3">{{ formatBool(event.store_false) }}</td>
                <td class="whitespace-nowrap px-5 py-3">
                  <div>{{ t('admin.rtkCompression.codexChain.previous') }}: {{ formatBool(event.has_previous_response_id) }}</div>
                  <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.rtkCompression.codexChain.removed') }}: {{ formatBool(event.previous_response_id_removed) }}</div>
                </td>
                <td class="whitespace-nowrap px-5 py-3">
                  <div>{{ event.reasoning_effort || '-' }}</div>
                  <div class="text-xs text-gray-500 dark:text-gray-400">{{ t('admin.rtkCompression.codexChain.encrypted') }}: {{ formatBool(event.has_reasoning_encrypted_content) }}</div>
                </td>
                <td class="min-w-[260px] px-5 py-3 font-mono text-xs">
                  <div>pc: {{ event.prompt_cache_key_hash || '-' }}</div>
                  <div>h.sid: {{ event.header_session_id_hash || '-' }}</div>
                  <div>h.cid: {{ event.header_conversation_id_hash || '-' }}</div>
                  <div>up.sid: {{ event.upstream_session_id_hash || '-' }}</div>
                  <div>up.cid: {{ event.upstream_conversation_id_hash || '-' }}</div>
                </td>
              </tr>
              <tr v-if="codexChainEvents.length === 0">
                <td colspan="8" class="px-5 py-8 text-center text-sm text-gray-500 dark:text-gray-400">{{ t('admin.rtkCompression.codexChain.noEvents') }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <div class="card overflow-hidden">
        <div class="border-b border-gray-100 px-5 py-4 dark:border-dark-700">
          <h2 class="text-base font-semibold text-gray-950 dark:text-white">{{ t('admin.rtkCompression.events.title') }}</h2>
        </div>
        <div class="overflow-x-auto">
          <table class="min-w-full divide-y divide-gray-100 text-sm dark:divide-dark-700">
            <thead class="bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500 dark:bg-dark-800/70 dark:text-gray-400">
              <tr>
                <th class="px-5 py-3">{{ t('admin.rtkCompression.events.time') }}</th>
                <th class="px-5 py-3">{{ t('admin.rtkCompression.events.account') }}</th>
                <th class="px-5 py-3 text-right">{{ t('admin.rtkCompression.events.hits') }}</th>
                <th class="px-5 py-3">{{ t('admin.rtkCompression.events.filters') }}</th>
                <th class="px-5 py-3 text-right">{{ t('admin.rtkCompression.events.before') }}</th>
                <th class="px-5 py-3 text-right">{{ t('admin.rtkCompression.events.after') }}</th>
                <th class="px-5 py-3 text-right">{{ t('admin.rtkCompression.events.saved') }}</th>
                <th class="px-5 py-3">{{ t('admin.rtkCompression.events.details') }}</th>
              </tr>
            </thead>
            <tbody class="divide-y divide-gray-100 dark:divide-dark-700">
              <tr v-for="event in recentEvents" :key="`${event.at}-${event.before}-${event.after}`" class="align-top text-gray-700 dark:text-gray-200">
                <td class="whitespace-nowrap px-5 py-3">{{ formatDateTime(event.at) }}</td>
                <td class="whitespace-nowrap px-5 py-3">{{ event.account_id || '-' }}</td>
                <td class="px-5 py-3 text-right">{{ formatNumber(event.hits) }}</td>
                <td class="px-5 py-3">{{ (event.filters || []).join(', ') || '-' }}</td>
                <td class="px-5 py-3 text-right">{{ formatBytes(event.before) }}</td>
                <td class="px-5 py-3 text-right">{{ formatBytes(event.after) }}</td>
                <td class="px-5 py-3 text-right text-green-600 dark:text-green-400">{{ formatBytes(event.bytes_saved) }}</td>
                <td class="whitespace-nowrap px-5 py-3">
                  <button
                    v-if="eventDetailCount(event) > 0"
                    type="button"
                    class="rounded-lg bg-gray-100 px-3 py-1.5 text-xs font-bold text-gray-700 hover:bg-gray-200 dark:bg-dark-700 dark:text-gray-200 dark:hover:bg-dark-600"
                    @click="openCompressionEventDetails(event)"
                  >
                    {{ t('admin.rtkCompression.events.viewDetails', { count: eventDetailCount(event) }) }}
                  </button>
                  <span v-else class="text-sm text-gray-400">-</span>
                </td>
              </tr>
              <tr v-if="recentEvents.length === 0">
                <td colspan="8" class="px-5 py-8 text-center text-sm text-gray-500 dark:text-gray-400">
                  {{ t('admin.rtkCompression.events.empty') }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <BaseDialog
        :show="showCompressionEventDetails"
        :title="t('admin.rtkCompression.eventDetails.title')"
        width="full"
        :close-on-click-outside="true"
        @close="closeCompressionEventDetails"
      >
        <template #default>
          <div v-if="selectedCompressionEvent" class="flex h-full min-h-0 flex-col">
            <div class="mb-4 grid grid-cols-1 gap-3 rounded-xl border border-gray-200 bg-gray-50 p-4 text-sm dark:border-dark-700 dark:bg-dark-800 md:grid-cols-4">
              <div>
                <span class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('admin.rtkCompression.events.time') }}</span>
                <p class="mt-1 font-medium text-gray-950 dark:text-white">{{ formatDateTime(selectedCompressionEvent.at) }}</p>
              </div>
              <div>
                <span class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('admin.rtkCompression.events.account') }}</span>
                <p class="mt-1 font-medium text-gray-950 dark:text-white">{{ selectedCompressionEvent.account_id || '-' }}</p>
              </div>
              <div>
                <span class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('admin.rtkCompression.events.hits') }}</span>
                <p class="mt-1 font-medium text-gray-950 dark:text-white">{{ formatNumber(selectedCompressionEvent.hits) }}</p>
              </div>
              <div>
                <span class="text-xs font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">{{ t('admin.rtkCompression.events.saved') }}</span>
                <p class="mt-1 font-medium text-green-600 dark:text-green-400">{{ formatBytes(selectedCompressionEvent.bytes_saved) }}</p>
              </div>
            </div>

            <div class="min-h-0 flex-1 overflow-hidden rounded-xl border border-gray-200 dark:border-dark-700">
              <div class="min-h-0 max-h-[70vh] overflow-auto">
                <table class="min-w-full divide-y divide-gray-200 text-sm dark:divide-dark-700">
                  <thead class="sticky top-0 z-10 bg-gray-50 text-left text-xs uppercase tracking-wide text-gray-500 dark:bg-dark-900 dark:text-gray-400">
                    <tr>
                      <th class="px-4 py-3">{{ t('admin.rtkCompression.eventDetails.type') }}</th>
                      <th class="px-4 py-3">{{ t('admin.rtkCompression.filters.filter') }}</th>
                      <th class="px-4 py-3">{{ t('admin.rtkCompression.eventDetails.path') }}</th>
                      <th class="px-4 py-3 text-right">{{ t('admin.rtkCompression.filters.before') }}</th>
                      <th class="px-4 py-3 text-right">{{ t('admin.rtkCompression.filters.after') }}</th>
                      <th class="px-4 py-3 text-right">{{ t('admin.rtkCompression.filters.saved') }}</th>
                      <th class="px-4 py-3">{{ t('admin.rtkCompression.eventDetails.reference') }}</th>
                      <th class="px-4 py-3">{{ t('admin.rtkCompression.eventDetails.sample') }}</th>
                    </tr>
                  </thead>
                  <tbody class="divide-y divide-gray-200 bg-white dark:divide-dark-700 dark:bg-dark-800">
                    <tr v-for="row in selectedCompressionEventRows" :key="row.key" class="hover:bg-gray-50 dark:hover:bg-dark-700/50">
                      <td class="whitespace-nowrap px-4 py-3">
                        <span :class="row.type === 'hit' ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-300' : 'bg-amber-100 text-amber-700 dark:bg-amber-900/30 dark:text-amber-300'" class="rounded-full px-2 py-1 text-[10px] font-bold">
                          {{ row.type === 'hit' ? t('admin.rtkCompression.eventDetails.hit') : t('admin.rtkCompression.eventDetails.miss') }}
                        </span>
                      </td>
                      <td class="whitespace-nowrap px-4 py-3 font-medium text-gray-800 dark:text-gray-100">{{ row.filter }}</td>
                      <td class="max-w-[320px] truncate px-4 py-3 font-mono text-xs text-gray-600 dark:text-gray-300" :title="row.path">{{ row.path }}</td>
                      <td class="whitespace-nowrap px-4 py-3 text-right text-gray-700 dark:text-gray-200">{{ formatBytes(row.before) }}</td>
                      <td class="whitespace-nowrap px-4 py-3 text-right text-gray-700 dark:text-gray-200">{{ row.after != null ? formatBytes(row.after) : '-' }}</td>
                      <td class="whitespace-nowrap px-4 py-3 text-right text-green-600 dark:text-green-400">{{ row.bytesSaved != null ? formatBytes(row.bytesSaved) : '-' }}</td>
                      <td class="max-w-[300px] px-4 py-3 font-mono text-xs text-gray-500 dark:text-gray-400">
                        <div v-if="row.referenceHash">hash: {{ row.referenceHash }}</div>
                        <div v-if="row.referencePath" class="truncate" :title="row.referencePath">first: {{ row.referencePath }}</div>
                        <span v-if="!row.referenceHash && !row.referencePath">-</span>
                      </td>
                      <td class="max-w-[360px] truncate px-4 py-3 text-xs text-gray-500 dark:text-gray-400" :title="row.sample || ''">{{ row.sample || '-' }}</td>
                    </tr>
                    <tr v-if="selectedCompressionEventRows.length === 0">
                      <td colspan="8" class="px-5 py-8 text-center text-sm text-gray-500 dark:text-gray-400">
                        {{ t('admin.rtkCompression.eventDetails.empty') }}
                      </td>
                    </tr>
                  </tbody>
                </table>
              </div>
            </div>
          </div>
        </template>
      </BaseDialog>
    </div>
  </AppLayout>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import AppLayout from '@/components/layout/AppLayout.vue'
import BaseDialog from '@/components/common/BaseDialog.vue'
import { adminAPI } from '@/api/admin'
import type { RTKCompressionEvent, RTKCompressionSnapshot } from '@/api/admin/rtkCompression'

const { t } = useI18n()

const loading = ref(false)
const errorMessage = ref('')
const snapshot = ref<RTKCompressionSnapshot | null>(null)
const autoRefresh = ref(true)
const selectedCompressionEvent = ref<RTKCompressionEvent | null>(null)
let refreshTimer: number | undefined

const filterRows = computed(() => snapshot.value?.by_filter ?? [])
const missedRows = computed(() => snapshot.value?.by_miss_reason ?? [])
const requestPartRows = computed(() => snapshot.value?.by_request_part ?? [])
const recentEvents = computed(() => snapshot.value?.recent_events ?? [])
const showCompressionEventDetails = computed(() => selectedCompressionEvent.value !== null)
const promptCache = computed(() => snapshot.value?.prompt_cache)
const promptCacheOutcomes = computed(() => promptCache.value?.by_outcome ?? [])
const promptCacheSources = computed(() => promptCache.value?.by_source ?? [])
const promptCacheEvents = computed(() => promptCache.value?.recent_events ?? [])
const codexChain = computed(() => snapshot.value?.codex_chain)
const codexChainTransports = computed(() => codexChain.value?.by_transport ?? [])
const codexChainEvents = computed(() => codexChain.value?.recent_events ?? [])
const selectedCompressionEventRows = computed(() => {
  const event = selectedCompressionEvent.value
  if (!event) return []
  const hits = (event.hit_stats ?? []).map((hit, index) => ({
    key: `hit-${index}-${hit.path}-${hit.before}-${hit.after}`,
    type: 'hit' as const,
    filter: hit.filter,
    path: hit.path,
    before: hit.before,
    after: hit.after,
    bytesSaved: hit.bytes_saved,
    referenceHash: hit.reference_hash || '',
    referencePath: hit.reference_path || '',
    sample: '',
  }))
  const misses = (event.miss_stats ?? []).map((miss, index) => ({
    key: `miss-${index}-${miss.path}-${miss.before}-${miss.reason}`,
    type: 'miss' as const,
    filter: formatMissReason(miss.reason),
    path: miss.path,
    before: miss.before,
    after: null,
    bytesSaved: null,
    referenceHash: '',
    referencePath: '',
    sample: miss.sample || '',
  }))
  return [...hits, ...misses]
})

const summaryCards = computed(() => [
  {
    label: t('admin.rtkCompression.cards.requests'),
    value: formatNumber(snapshot.value?.requests ?? 0),
    hint: `${formatNumber(snapshot.value?.hits ?? 0)} ${t('admin.rtkCompression.cards.hits')}`,
  },
  {
    label: t('admin.rtkCompression.cards.saved'),
    value: formatBytes(snapshot.value?.bytes_saved ?? 0),
    hint: `${formatBytes(snapshot.value?.before ?? 0)} → ${formatBytes(snapshot.value?.after ?? 0)}`,
  },
  {
    label: t('admin.rtkCompression.cards.ratio'),
    value: formatPercent(snapshot.value?.save_ratio ?? 0),
    hint: snapshot.value?.updated_at ? formatDateTime(snapshot.value.updated_at) : t('admin.rtkCompression.noHits'),
  },
  {
    label: t('admin.rtkCompression.cards.missed'),
    value: formatNumber(snapshot.value?.misses ?? 0),
    hint: formatBytes(snapshot.value?.missed_bytes ?? 0),
  },
  {
    label: t('admin.rtkCompression.config'),
    value: snapshot.value?.enabled ? t('admin.rtkCompression.enabled') : t('admin.rtkCompression.disabled'),
    hint: `${formatBytes(snapshot.value?.min_bytes ?? 0)} / ${formatBytes(snapshot.value?.max_bytes ?? 0)}`,
  },
])

async function loadSnapshot() {
  loading.value = true
  errorMessage.value = ''
  try {
    snapshot.value = await adminAPI.rtkCompression.getSnapshot()
  } catch (error: any) {
    errorMessage.value = error?.message || 'Failed to load RTK compression stats'
  } finally {
    loading.value = false
  }
}

function setupAutoRefresh() {
  if (refreshTimer) {
    window.clearInterval(refreshTimer)
    refreshTimer = undefined
  }
  if (autoRefresh.value) {
    refreshTimer = window.setInterval(() => {
      void loadSnapshot()
    }, 5000)
  }
}

function formatBytes(value: number): string {
  if (!Number.isFinite(value) || value <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB']
  let size = value
  let unit = 0
  while (size >= 1024 && unit < units.length - 1) {
    size /= 1024
    unit++
  }
  return `${size >= 10 || unit === 0 ? size.toFixed(0) : size.toFixed(1)} ${units[unit]}`
}

function formatNumber(value: number): string {
  if (!Number.isFinite(value)) return '0'
  return new Intl.NumberFormat().format(value)
}

function formatPercent(value: number): string {
  if (!Number.isFinite(value)) return '0%'
  return `${(value * 100).toFixed(1)}%`
}

function safeRatio(saved: number, before: number): number {
  return before > 0 ? saved / before : 0
}

function formatMissReason(reason: string): string {
  return t(`admin.rtkCompression.missReasons.${reason}`, reason)
}

function formatRequestPart(part: string): string {
  return t(`admin.rtkCompression.requestPartNames.${part}`, part)
}

function formatPromptCacheOutcome(outcome: string): string {
  return t(`admin.rtkCompression.promptCache.outcomes.${outcome}`, outcome)
}

function formatPromptCacheSource(source: string): string {
  return t(`admin.rtkCompression.promptCache.sources.${source}`, source)
}

function formatBool(value?: boolean): string {
  return value ? t('admin.rtkCompression.codexChain.yes') : t('admin.rtkCompression.codexChain.no')
}

function eventDetailCount(event: RTKCompressionEvent): number {
  return (event.hit_stats?.length ?? 0) + (event.miss_stats?.length ?? 0)
}

function openCompressionEventDetails(event: RTKCompressionEvent) {
  selectedCompressionEvent.value = event
}

function closeCompressionEventDetails() {
  selectedCompressionEvent.value = null
}

function formatDateTime(value?: string): string {
  if (!value) return ''
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return new Intl.DateTimeFormat(undefined, {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  }).format(date)
}

watch(autoRefresh, setupAutoRefresh)

onMounted(() => {
  void loadSnapshot()
  setupAutoRefresh()
})

onUnmounted(() => {
  if (refreshTimer) {
    window.clearInterval(refreshTimer)
  }
})
</script>
