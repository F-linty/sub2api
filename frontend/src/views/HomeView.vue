<template>
  <!-- Custom Home Content: Full Page Mode -->
  <div v-if="homeContent" class="min-h-screen">
    <iframe
      v-if="isHomeContentUrl"
      :src="homeContent.trim()"
      class="h-screen w-full border-0"
      allowfullscreen
    ></iframe>
    <!-- SECURITY: homeContent is an admin-only setting. -->
    <div v-else v-html="homeContent"></div>
  </div>

  <!-- Default Home Page -->
  <div v-else class="home-shell relative flex min-h-screen flex-col overflow-hidden text-[#191817] dark:bg-dark-950 dark:text-white">
    <header class="relative z-20 px-4 py-5 sm:px-6">
      <nav class="mx-auto flex max-w-6xl items-center justify-between">
        <div class="inline-flex items-center gap-3">
          <div class="flex h-12 w-12 items-center justify-center rounded-2xl border border-[#eaded6] bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800">
            <img :src="siteLogo || '/logo.png'" alt="Logo" class="h-8 w-8 object-contain" />
          </div>
          <span class="hidden text-sm font-semibold tracking-[0.24em] text-[#383532] dark:text-dark-100 sm:inline">
            {{ displayName }}
          </span>
        </div>

        <div class="flex items-center gap-2 sm:gap-3">
          <LocaleSwitcher />

          <a
            v-if="docUrl"
            :href="docUrl"
            target="_blank"
            rel="noopener noreferrer"
            class="home-icon-button"
            :title="t('home.viewDocs')"
          >
            <Icon name="book" size="md" />
          </a>

          <button
            type="button"
            class="home-icon-button"
            :title="isDark ? t('home.switchToLight') : t('home.switchToDark')"
            @click="toggleTheme"
          >
            <Icon v-if="isDark" name="sun" size="md" />
            <Icon v-else name="moon" size="md" />
          </button>

          <router-link
            v-if="isAuthenticated"
            :to="dashboardPath"
            class="inline-flex h-10 items-center gap-2 rounded-full bg-[#202020] px-3 text-sm font-semibold text-white shadow-[0_10px_22px_-16px_rgba(0,0,0,0.8)] transition-colors hover:bg-[#2b2b2b] dark:bg-white dark:text-slate-950"
          >
            <span class="flex h-6 w-6 items-center justify-center rounded-full bg-[#fbe7df] text-xs font-bold text-[#873628]">
              {{ userInitial }}
            </span>
            <span class="hidden sm:inline">{{ t('home.dashboard') }}</span>
          </router-link>
        </div>
      </nav>
    </header>

    <main class="relative z-10 flex-1 px-4 pb-10 pt-6 sm:px-6 lg:pb-14">
      <section
        class="home-showcase mx-auto grid max-w-6xl overflow-hidden rounded-[28px] border border-[#ebe4df] bg-white shadow-[0_24px_70px_-48px_rgba(25,24,23,0.55)] dark:border-dark-700 dark:bg-dark-900 lg:grid-cols-[0.98fr_1.02fr]"
        :class="{
          'home-showcase-entering': shouldSlideInFromAuth,
          'home-showcase-leaving': isAuthTransitioning,
        }"
      >
        <div class="home-brand-panel flex min-h-[520px] flex-col justify-between border-b border-[#ebe4df] bg-[#fffaf7] p-7 dark:border-dark-700 dark:bg-dark-900/80 sm:p-10 lg:min-h-[620px] lg:border-b-0 lg:border-r">
          <div class="pt-8 sm:pt-10">
            <p class="mb-4 text-xs font-semibold uppercase tracking-[0.28em] text-[#a94735]">
              API Gateway
            </p>
            <h1 class="max-w-xl text-5xl font-semibold leading-none tracking-tight text-[#191817] dark:text-white sm:text-6xl">
              {{ siteName }}
            </h1>
            <p class="mt-6 max-w-md text-base leading-7 text-[#74706d] dark:text-dark-300 sm:text-lg">
              {{ siteSubtitle }}
            </p>

            <div class="mt-9 flex flex-wrap items-center gap-3">
              <button
                type="button"
                class="inline-flex h-12 items-center gap-2 rounded-2xl bg-[#202020] px-6 text-sm font-semibold text-white shadow-[0_16px_28px_-20px_rgba(0,0,0,0.85)] transition-colors hover:bg-[#2b2b2b] dark:bg-white dark:text-slate-950"
                @click="navigatePrimary"
              >
                {{ isAuthenticated ? t('home.goToDashboard') : t('home.getStarted') }}
                <Icon name="arrowRight" size="sm" :stroke-width="2" />
              </button>
              <a
                v-if="docUrl"
                :href="docUrl"
                target="_blank"
                rel="noopener noreferrer"
                class="inline-flex h-12 items-center gap-2 rounded-2xl border border-[#ebe4df] bg-white px-5 text-sm font-semibold text-[#5f5b57] transition-colors hover:border-[#d9cfc8] hover:bg-[#fbfaf8] dark:border-dark-700 dark:bg-dark-800 dark:text-dark-100"
              >
                <Icon name="book" size="sm" />
                {{ t('home.docs') }}
              </a>
            </div>
          </div>

          <div class="mt-8">
            <div ref="redshiftStageRef" class="redshift-stage" aria-hidden="true">
              <div class="redshift-bars">
                <span
                  v-for="(byte, index) in redshiftBytes"
                  :key="byte.color"
                  :class="{ 'is-jumping': activeByteIndex === index }"
                  :style="getByteStyle(byte, index)"
                ></span>
              </div>
            </div>
            <div class="mt-8 grid gap-3 text-sm text-[#5f5b57] dark:text-dark-300 sm:grid-cols-3 lg:grid-cols-1 xl:grid-cols-3">
              <div class="home-pill">
                <Icon name="swap" size="sm" />
                <span>{{ t('home.tags.subscriptionToApi') }}</span>
              </div>
              <div class="home-pill">
                <Icon name="shield" size="sm" />
                <span>{{ t('home.tags.stickySession') }}</span>
              </div>
              <div class="home-pill">
                <Icon name="chart" size="sm" />
                <span>{{ t('home.tags.realtimeBilling') }}</span>
              </div>
            </div>
          </div>
        </div>

        <div class="flex min-h-[520px] flex-col justify-center bg-white p-7 dark:bg-dark-900 sm:p-10 lg:min-h-[620px] lg:p-12">
          <div class="rounded-3xl border border-[#ebe4df] bg-[#fbfaf8] p-4 dark:border-dark-700 dark:bg-dark-800/80">
            <div class="rounded-2xl bg-[#172033] shadow-[0_24px_40px_-28px_rgba(23,32,51,0.9)]">
              <div class="flex items-center justify-between border-b border-white/5 px-5 py-4">
                <div class="flex items-center gap-2">
                  <span class="h-3 w-3 rounded-full bg-[#c45b45]"></span>
                  <span class="h-3 w-3 rounded-full bg-[#d6a13d]"></span>
                  <span class="h-3 w-3 rounded-full bg-[#0f8f6a]"></span>
                </div>
                <span class="font-mono text-xs text-slate-500">waystation</span>
              </div>
              <div class="space-y-4 px-5 py-6 font-mono text-sm leading-7">
                <div class="text-slate-300">
                  <span class="text-[#0f8f6a]">$</span>
                  <span class="ml-2 text-[#f3d0c5]">POST</span>
                  <span class="ml-2 text-slate-400">/v1/messages</span>
                </div>
                <div class="text-slate-500"># routing through RedShifts...</div>
                <div class="flex flex-wrap items-center gap-3">
                  <span class="rounded-md bg-[#eaf7f1] px-3 py-1 font-semibold text-[#0f8f6a]">200 OK</span>
                  <span class="text-[#f3d0c5]">{ "content": "Hello!" }</span>
                </div>
              </div>
            </div>

            <div class="mt-4 grid gap-3 sm:grid-cols-3">
              <div class="metric-tile">
                <span>{{ t('home.features.unifiedGateway') }}</span>
                <strong>1 API</strong>
              </div>
              <div class="metric-tile">
                <span>{{ t('home.features.multiAccount') }}</span>
                <strong>Pool</strong>
              </div>
              <div class="metric-tile">
                <span>{{ t('home.features.balanceQuota') }}</span>
                <strong>Quota</strong>
              </div>
            </div>
          </div>

          <div class="mt-8 grid gap-4 sm:grid-cols-3">
            <article class="feature-card">
              <div class="feature-icon bg-[#f3f1ee] text-[#5f5b57]">
                <Icon name="server" size="md" />
              </div>
              <h3>{{ t('home.features.unifiedGateway') }}</h3>
              <p>{{ t('home.features.unifiedGatewayDesc') }}</p>
            </article>
            <article class="feature-card">
              <div class="feature-icon bg-[#fbe7df] text-[#873628]">
                <Icon name="users" size="md" />
              </div>
              <h3>{{ t('home.features.multiAccount') }}</h3>
              <p>{{ t('home.features.multiAccountDesc') }}</p>
            </article>
            <article class="feature-card">
              <div class="feature-icon bg-[#f1efff] text-[#6b5ce7]">
                <Icon name="chart" size="md" />
              </div>
              <h3>{{ t('home.features.balanceQuota') }}</h3>
              <p>{{ t('home.features.balanceQuotaDesc') }}</p>
            </article>
          </div>
        </div>
      </section>
    </main>

    <footer class="relative z-10 px-6 pb-8">
      <div class="mx-auto flex max-w-6xl flex-col items-center justify-center gap-3 text-center text-sm text-[#8d8782] dark:text-dark-400 sm:flex-row">
        <p>&copy; {{ currentYear }} {{ siteName }}. {{ t('home.footer.allRightsReserved') }}</p>
        <a
          :href="githubUrl"
          target="_blank"
          rel="noopener noreferrer"
          class="font-medium text-[#5f5b57] transition-colors hover:text-[#a94735] dark:text-dark-300"
        >
          GitHub
        </a>
      </div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { useAuthStore, useAppStore } from '@/stores'
import LocaleSwitcher from '@/components/common/LocaleSwitcher.vue'
import Icon from '@/components/icons/Icon.vue'
import { sanitizeUrl } from '@/utils/url'

const { t } = useI18n()
const router = useRouter()

const authStore = useAuthStore()
const appStore = useAppStore()

const siteName = computed(() => appStore.cachedPublicSettings?.site_name || appStore.siteName || 'Sub2API')
const displayName = computed(() => siteName.value.toUpperCase())
const siteLogo = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.site_logo || appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'AI API Gateway Platform')
const docUrl = computed(() => sanitizeUrl(appStore.cachedPublicSettings?.doc_url || appStore.docUrl || ''))
const homeContent = computed(() => appStore.cachedPublicSettings?.home_content || '')

const isHomeContentUrl = computed(() => {
  const content = homeContent.value.trim()
  return content.startsWith('http://') || content.startsWith('https://')
})

const isDark = ref(document.documentElement.classList.contains('dark'))
const isAuthTransitioning = ref(false)
const shouldSlideInFromAuth = ref(
  typeof window !== 'undefined' && sessionStorage.getItem('redshifts-home-entry') === 'slide'
)
const githubUrl = 'https://github.com/Wei-Shaw/sub2api'
const redshiftBytes = [
  { color: '#220a06', width: '64px', rest: 28, pulse: 42, duration: 2.15, delay: 0 },
  { color: '#873628', width: '112px', rest: 56, pulse: 72, duration: 2.55, delay: 0.18 },
  { color: '#c45b45', width: '80px', rest: 42, pulse: 60, duration: 2.28, delay: 0.34 },
  { color: '#f3d0c5', width: '144px', rest: 22, pulse: 36, duration: 2.7, delay: 0.52 },
]
const activeByteIndex = ref<number | null>(null)
const activeByteTravel = ref(0)
const redshiftStageRef = ref<HTMLElement | null>(null)
let byteTimer: number | undefined
let byteResetTimer: number | undefined

const isAuthenticated = computed(() => authStore.isAuthenticated)
const isAdmin = computed(() => authStore.isAdmin)
const dashboardPath = computed(() => isAdmin.value ? '/admin/dashboard' : '/dashboard')
const userInitial = computed(() => {
  const user = authStore.user
  if (!user || !user.email) return ''
  return user.email.charAt(0).toUpperCase()
})

const currentYear = computed(() => new Date().getFullYear())

function toggleTheme() {
  isDark.value = !isDark.value
  document.documentElement.classList.toggle('dark', isDark.value)
  localStorage.setItem('theme', isDark.value ? 'dark' : 'light')
}

function prefersReducedMotion() {
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches
}

function randomBetween(min: number, max: number) {
  return Math.round(min + Math.random() * (max - min))
}

function getAdaptiveByteTravel(byte: typeof redshiftBytes[number]) {
  const stageHeight = redshiftStageRef.value?.clientHeight || 180
  const headroom = Math.max(stageHeight - byte.pulse - 12, 56)
  return randomBetween(Math.round(headroom * 0.72), Math.round(headroom * 0.96))
}

function getByteStyle(byte: typeof redshiftBytes[number], index: number) {
  const isActive = activeByteIndex.value === index
  return {
    width: byte.width,
    backgroundColor: byte.color,
    '--byte-rest': `${byte.rest}px`,
    '--byte-pulse': `${byte.pulse}px`,
    '--byte-travel': `${isActive ? -activeByteTravel.value : 0}px`,
    '--pulse-duration': `${byte.duration}s`,
    '--pulse-delay': `${byte.delay}s`,
  }
}

function scheduleByteJump() {
  if (prefersReducedMotion()) return

  byteTimer = window.setTimeout(() => {
    let nextIndex = randomBetween(0, redshiftBytes.length - 1)
    if (redshiftBytes.length > 1 && nextIndex === activeByteIndex.value) {
      nextIndex = (nextIndex + randomBetween(1, redshiftBytes.length - 1)) % redshiftBytes.length
    }

    activeByteIndex.value = nextIndex
    activeByteTravel.value = getAdaptiveByteTravel(redshiftBytes[nextIndex])

    window.clearTimeout(byteResetTimer)
    byteResetTimer = window.setTimeout(() => {
      activeByteIndex.value = null
      scheduleByteJump()
    }, randomBetween(620, 780))
  }, randomBetween(1150, 2600))
}

function navigateToAuth() {
  if (isAuthTransitioning.value) return

  if (prefersReducedMotion()) {
    router.push('/login')
    return
  }

  isAuthTransitioning.value = true
  sessionStorage.setItem('redshifts-auth-entry', 'slide')
  window.setTimeout(() => {
    router.push('/login')
  }, 280)
}

function navigatePrimary() {
  if (isAuthenticated.value) {
    router.push(dashboardPath.value)
    return
  }

  navigateToAuth()
}

function initTheme() {
  const savedTheme = localStorage.getItem('theme')
  if (
    savedTheme === 'dark' ||
    (!savedTheme && window.matchMedia('(prefers-color-scheme: dark)').matches)
  ) {
    isDark.value = true
    document.documentElement.classList.add('dark')
  }
}

onMounted(() => {
  if (shouldSlideInFromAuth.value) {
    sessionStorage.removeItem('redshifts-home-entry')
  }

  initTheme()
  scheduleByteJump()
  authStore.checkAuth()

  if (!appStore.publicSettingsLoaded) {
    appStore.fetchPublicSettings()
  }
})

onUnmounted(() => {
  window.clearTimeout(byteTimer)
  window.clearTimeout(byteResetTimer)
})
</script>

<style scoped>
.home-shell {
  background:
    linear-gradient(180deg, rgba(255, 250, 247, 0.92), rgba(246, 243, 239, 0.96)),
    #f6f3ef;
}

.home-shell::before {
  position: absolute;
  inset: 0;
  pointer-events: none;
  content: '';
  background:
    linear-gradient(90deg, rgba(235, 228, 223, 0.42) 1px, transparent 1px),
    linear-gradient(180deg, rgba(235, 228, 223, 0.42) 1px, transparent 1px);
  background-size: 72px 72px;
  mask-image: linear-gradient(to bottom, rgba(0, 0, 0, 0.75), transparent 72%);
}

.home-icon-button {
  display: inline-flex;
  width: 40px;
  height: 40px;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  color: #5f5b57;
  transition: background-color 0.15s ease, color 0.15s ease;
}

.home-icon-button:hover {
  color: #191817;
  background: #f3f1ee;
}

.dark .home-icon-button {
  color: #d1d5db;
}

.home-brand-panel {
  position: relative;
}

.home-showcase {
  transform-origin: center;
  transition:
    opacity 0.28s ease,
    transform 0.28s cubic-bezier(0.4, 0, 0.2, 1),
    filter 0.28s ease;
}

.home-showcase-leaving {
  opacity: 0;
  filter: blur(2px);
  transform: translateX(-72px) scale(0.985);
}

.home-showcase-entering {
  animation: home-showcase-slide-in 0.42s cubic-bezier(0.22, 1, 0.36, 1) both;
}

@keyframes home-showcase-slide-in {
  from {
    opacity: 0;
    filter: blur(2px);
    transform: translateX(-72px) scale(0.985);
  }

  to {
    opacity: 1;
    filter: blur(0);
    transform: translateX(0) scale(1);
  }
}

.home-brand-panel::before {
  position: absolute;
  top: 0;
  right: 0;
  left: 0;
  height: 3px;
  content: '';
  background: linear-gradient(90deg, #220a06 0%, #873628 34%, #c45b45 62%, #f3d0c5 100%);
}

.redshift-stage {
  position: relative;
  height: clamp(164px, 24vh, 236px);
  overflow: hidden;
  border-bottom: 1px solid rgba(218, 207, 200, 0.72);
}

.redshift-bars {
  position: absolute;
  right: 0;
  bottom: 22px;
  left: 0;
  display: flex;
  align-items: flex-end;
  gap: 10px;
}

.redshift-bars span {
  display: block;
  height: var(--byte-rest);
  border-radius: 999px;
  transform-origin: center bottom;
  animation: redshift-byte-pulse var(--pulse-duration) cubic-bezier(0.32, 0.72, 0.18, 1) var(--pulse-delay) infinite;
  will-change: height, transform, box-shadow;
}

.redshift-bars span.is-jumping {
  animation:
    redshift-byte-pulse var(--pulse-duration) cubic-bezier(0.32, 0.72, 0.18, 1) var(--pulse-delay) infinite,
    redshift-byte-hop 0.86s cubic-bezier(0.2, 0.86, 0.22, 1) both;
}

@keyframes redshift-byte-pulse {
  0%,
  100% {
    height: var(--byte-rest);
  }

  18% {
    height: var(--byte-pulse);
  }

  30% {
    height: calc(var(--byte-rest) - 4px);
  }

  42% {
    height: calc(var(--byte-rest) + 7px);
  }

  54% {
    height: var(--byte-rest);
  }
}

@keyframes redshift-byte-hop {
  0%,
  100% {
    transform: translateY(0);
    box-shadow: none;
  }

  34% {
    transform: translateY(var(--byte-travel));
    box-shadow: 0 34px 40px -32px rgba(154, 62, 47, 0.7);
  }

  58% {
    transform: translateY(0);
    box-shadow: none;
  }

  76% {
    transform: translateY(calc(var(--byte-travel) * 0.16));
  }
}

.home-pill {
  display: inline-flex;
  min-height: 44px;
  align-items: center;
  gap: 8px;
  border: 1px solid #ebe4df;
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.72);
  padding: 10px 12px;
}

.metric-tile {
  min-height: 84px;
  border: 1px solid #ebe4df;
  border-radius: 18px;
  background: #ffffff;
  padding: 14px;
}

.metric-tile span {
  display: block;
  color: #74706d;
  font-size: 12px;
  font-weight: 600;
}

.metric-tile strong {
  display: block;
  margin-top: 10px;
  color: #191817;
  font-size: 22px;
  font-weight: 700;
}

.feature-card {
  min-height: 188px;
  border: 1px solid #ebe4df;
  border-radius: 20px;
  background: #ffffff;
  padding: 18px;
}

.feature-icon {
  display: inline-flex;
  width: 42px;
  height: 42px;
  align-items: center;
  justify-content: center;
  border-radius: 14px;
}

.feature-card h3 {
  margin-top: 18px;
  color: #191817;
  font-size: 16px;
  font-weight: 700;
}

.feature-card p {
  margin-top: 8px;
  color: #74706d;
  font-size: 13px;
  line-height: 1.7;
}

.dark .home-shell {
  background: #020617;
}

.dark .home-pill,
.dark .metric-tile,
.dark .feature-card {
  border-color: rgb(51 65 85);
  background: rgb(15 23 42 / 0.78);
}

.dark .metric-tile strong,
.dark .feature-card h3 {
  color: #f8fafc;
}

.dark .metric-tile span,
.dark .feature-card p {
  color: #94a3b8;
}

@media (prefers-reduced-motion: reduce) {
  .home-showcase {
    transition: none;
  }

  .home-showcase-entering,
  .redshift-bars span {
    animation: none;
  }
}
</style>
