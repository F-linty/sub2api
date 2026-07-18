<template>
  <div class="auth-shell relative flex min-h-screen items-center justify-center overflow-hidden p-4 dark:bg-dark-950 sm:p-6">
    <div
      class="auth-frame relative z-10 grid w-full max-w-5xl overflow-hidden rounded-[28px] border border-[#ebe4df] bg-white shadow-[0_24px_70px_-48px_rgba(25,24,23,0.55)] dark:border-dark-700 dark:bg-dark-900 lg:grid-cols-[0.95fr_1.05fr]"
      :class="{ 'auth-frame-slide-in': shouldSlideIn, 'auth-frame-slide-out': isReturningHome }"
    >
      <section class="auth-brand-panel hidden min-h-[640px] flex-col justify-between border-r border-[#ebe4df] bg-[#fffaf7] p-10 dark:border-dark-700 dark:bg-dark-900/80 lg:flex">
        <template v-if="settingsLoaded">
          <div class="flex flex-1 flex-col">
            <button
              type="button"
              class="auth-home-link inline-flex w-fit items-center gap-2 rounded-full border border-[#eaded6] bg-white/72 px-3.5 py-2 text-sm font-medium text-[#5f5b57] shadow-sm backdrop-blur transition hover:border-[#d6b6aa] hover:bg-white hover:text-[#9a3e2f] focus:outline-none focus:ring-2 focus:ring-[#c45b45]/20 dark:border-dark-700 dark:bg-dark-800/76 dark:text-dark-300 dark:hover:border-[#b75a47]/60 dark:hover:text-[#ffbea9]"
              @click="returnHome"
            >
              <Icon name="arrowLeft" size="sm" />
              <span>返回首页</span>
            </button>

            <div class="mt-12">
              <div class="mb-8 inline-flex h-16 w-16 items-center justify-center rounded-[22px] border border-[#eaded6] bg-white shadow-[0_12px_28px_-22px_rgba(25,24,23,0.7)] dark:border-dark-700 dark:bg-dark-800">
                <img v-if="siteLogo" :src="siteLogo" alt="Logo" class="h-11 w-11 object-contain" />
                <BeaconMark v-else size="md" />
              </div>
              <h1 class="brand-title text-[28px] font-semibold leading-tight text-[#191817] dark:text-white">
                {{ displayName }}
              </h1>
              <p class="mt-4 max-w-xs text-sm leading-6 text-[#74706d] dark:text-dark-300">
                {{ siteSubtitle }}
              </p>
            </div>
          </div>

          <div class="space-y-5">
            <div class="redshift-lines" aria-hidden="true">
              <span
                v-for="(byte, index) in redshiftBytes"
                :key="byte.color"
                :class="{ 'is-jumping': activeByteIndex === index }"
                :style="getByteStyle(byte, index)"
              ></span>
            </div>
            <div class="text-xs font-medium text-[#8d8782] dark:text-dark-400">
              &copy; {{ currentYear }} {{ siteName }}
            </div>
          </div>
        </template>
      </section>

      <main class="flex min-h-[640px] flex-col justify-center bg-white px-5 py-8 dark:bg-dark-900 sm:px-8 lg:px-14">
        <div class="mx-auto w-full max-w-[31rem]">
          <div class="mb-8 text-center lg:hidden">
            <template v-if="settingsLoaded">
              <div class="mb-5 flex justify-start">
                <button
                  type="button"
                  class="auth-home-link inline-flex items-center gap-2 rounded-full border border-[#eaded6] bg-[#fffaf7]/84 px-3.5 py-2 text-sm font-medium text-[#5f5b57] shadow-sm backdrop-blur transition hover:border-[#d6b6aa] hover:bg-white hover:text-[#9a3e2f] focus:outline-none focus:ring-2 focus:ring-[#c45b45]/20 dark:border-dark-700 dark:bg-dark-800/76 dark:text-dark-300 dark:hover:border-[#b75a47]/60 dark:hover:text-[#ffbea9]"
                  @click="returnHome"
                >
                  <Icon name="arrowLeft" size="sm" />
                  <span>返回首页</span>
                </button>
              </div>

              <div class="mb-4 inline-flex h-16 w-16 items-center justify-center rounded-2xl border border-[#eaded6] bg-white shadow-sm dark:border-dark-700 dark:bg-dark-800">
                <img v-if="siteLogo" :src="siteLogo" alt="Logo" class="h-11 w-11 object-contain" />
                <BeaconMark v-else size="md" />
              </div>
              <h1 class="brand-title text-2xl font-semibold text-[#191817] dark:text-white">
                {{ displayName }}
              </h1>
              <p class="mt-2 text-sm text-[#74706d] dark:text-dark-400">
                {{ siteSubtitle }}
              </p>
            </template>
          </div>

          <div class="auth-card rounded-2xl border border-[#ebe4df] bg-white p-7 shadow-[0_14px_34px_-30px_rgba(25,24,23,0.9)] dark:border-dark-700 dark:bg-dark-800 sm:p-8">
            <slot />
          </div>

          <div class="mt-6 text-center text-sm">
            <slot name="footer" />
          </div>

          <div class="mt-7 text-center text-xs text-gray-400 dark:text-dark-500 lg:hidden">
            &copy; {{ currentYear }} {{ siteName }}. All rights reserved.
          </div>
        </div>
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useAppStore } from '@/stores'
import BeaconMark from '@/components/common/BeaconMark.vue'
import Icon from '@/components/icons/Icon.vue'
import { sanitizeUrl } from '@/utils/url'

const appStore = useAppStore()
const router = useRouter()
const shouldSlideIn = ref(
  typeof window !== 'undefined' && sessionStorage.getItem('redshifts-auth-entry') === 'slide'
)
const isReturningHome = ref(false)
const redshiftBytes = [
  { color: '#220a06', width: '64px', rest: 42, pulse: 60, duration: 2.15, delay: 0 },
  { color: '#873628', width: '112px', rest: 70, pulse: 92, duration: 2.55, delay: 0.18 },
  { color: '#c45b45', width: '80px', rest: 54, pulse: 76, duration: 2.28, delay: 0.34 },
  { color: '#f3d0c5', width: '144px', rest: 24, pulse: 42, duration: 2.7, delay: 0.52 },
]
const activeByteIndex = ref<number | null>(null)
const activeByteTravel = ref(0)
let byteTimer: number | undefined
let byteResetTimer: number | undefined

const siteName = computed(() => appStore.siteName || 'Sub2API')
const displayName = computed(() => (siteName.value || 'Redshift').toUpperCase())
const siteLogo = computed(() => sanitizeUrl(appStore.siteLogo || '', { allowRelative: true, allowDataUrl: true }))
const siteSubtitle = computed(() => appStore.cachedPublicSettings?.site_subtitle || 'Gateway control plane')
const settingsLoaded = computed(() => appStore.publicSettingsLoaded)

const currentYear = computed(() => new Date().getFullYear())

function prefersReducedMotion() {
  return window.matchMedia('(prefers-reduced-motion: reduce)').matches
}

function randomBetween(min: number, max: number) {
  return Math.round(min + Math.random() * (max - min))
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
    activeByteTravel.value = randomBetween(72, 118)

    window.clearTimeout(byteResetTimer)
    byteResetTimer = window.setTimeout(() => {
      activeByteIndex.value = null
      scheduleByteJump()
    }, randomBetween(620, 780))
  }, randomBetween(1150, 2600))
}

function returnHome() {
  if (isReturningHome.value) return

  if (prefersReducedMotion()) {
    router.push('/home')
    return
  }

  isReturningHome.value = true
  sessionStorage.setItem('redshifts-home-entry', 'slide')
  window.setTimeout(() => {
    router.push('/home')
  }, 280)
}

onMounted(() => {
  if (shouldSlideIn.value) {
    sessionStorage.removeItem('redshifts-auth-entry')
  }

  appStore.fetchPublicSettings()
  scheduleByteJump()
})

onUnmounted(() => {
  window.clearTimeout(byteTimer)
  window.clearTimeout(byteResetTimer)
})
</script>

<style scoped>
.auth-shell {
  background:
    linear-gradient(180deg, rgba(255, 250, 247, 0.92), rgba(246, 243, 239, 0.96)),
    #f6f3ef;
}

.auth-frame::before {
  position: absolute;
  top: 0;
  right: 0;
  left: 0;
  height: 3px;
  content: '';
  background: linear-gradient(90deg, #220a06 0%, #873628 34%, #c45b45 62%, #f3d0c5 100%);
}

.auth-frame-slide-in {
  animation: auth-frame-slide-in 0.42s cubic-bezier(0.22, 1, 0.36, 1) both;
}

.auth-frame-slide-out {
  animation: auth-frame-slide-out 0.3s cubic-bezier(0.4, 0, 0.2, 1) both;
}

@keyframes auth-frame-slide-in {
  from {
    opacity: 0;
    transform: translateX(88px) scale(0.985);
  }

  to {
    opacity: 1;
    transform: translateX(0) scale(1);
  }
}

@keyframes auth-frame-slide-out {
  from {
    opacity: 1;
    transform: translateX(0) scale(1);
  }

  to {
    opacity: 0;
    transform: translateX(88px) scale(0.985);
  }
}

.brand-title {
  letter-spacing: 0.16em;
}

.redshift-lines {
  display: flex;
  height: 96px;
  align-items: flex-end;
  gap: 10px;
}

.redshift-lines span {
  display: block;
  height: var(--byte-rest);
  border-radius: 999px;
  transform-origin: center bottom;
  animation: redshift-byte-pulse var(--pulse-duration) cubic-bezier(0.32, 0.72, 0.18, 1) var(--pulse-delay) infinite;
  will-change: height, transform, box-shadow;
}

.redshift-lines span.is-jumping {
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

@media (prefers-reduced-motion: reduce) {
  .auth-frame-slide-in,
  .auth-frame-slide-out,
  .redshift-lines span {
    animation: none;
  }
}
</style>
