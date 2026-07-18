<template>
  <div class="min-h-screen bg-transparent dark:bg-dark-950">
    <Teleport to="body">
      <TransitionGroup
        name="page-notice"
        appear
        tag="div"
        class="pointer-events-none fixed left-1/2 top-3 z-[10000] flex w-[min(520px,calc(100vw-2rem))] -translate-x-1/2 flex-col items-center gap-2 sm:top-4"
        aria-live="polite"
        aria-atomic="true"
      >
        <div
          v-for="notice in pageNotices"
          :key="notice.id"
          class="pointer-events-auto flex max-w-full items-center gap-2.5 rounded-full border border-[#eadfd9]/90 bg-[#fffaf7]/95 px-3.5 py-2.5 text-sm text-[#2a1814] shadow-[0_14px_36px_rgba(74,39,29,0.10)] backdrop-blur-md dark:border-dark-700/80 dark:bg-dark-900/92 dark:text-dark-100 sm:px-4"
        >
          <span
            class="flex h-6 w-6 flex-shrink-0 items-center justify-center rounded-full bg-[#f5e3dc] text-[#a94735] dark:bg-red-950/40 dark:text-red-200"
          >
            <Icon :name="getNoticeIconName(notice.type)" size="xs" />
          </span>
          <span class="min-w-0 truncate font-medium">{{ notice.message }}</span>
          <button
            type="button"
            class="-mr-1 rounded-full p-1 text-[#9a8b84] transition-colors hover:bg-[#f3e7df] hover:text-[#2a1814] dark:text-dark-400 dark:hover:bg-dark-800 dark:hover:text-white"
            aria-label="Close notification"
            @click="appStore.hidePageNotice(notice.id)"
          >
            <Icon name="x" size="xs" />
          </button>
        </div>
      </TransitionGroup>
    </Teleport>

    <!-- Sidebar -->
    <AppSidebar />

    <!-- Main Content Area -->
    <div
      class="relative min-h-screen transition-all duration-300"
      :class="[sidebarCollapsed ? 'lg:ml-[68px]' : 'lg:ml-[224px]']"
    >
      <!-- Header -->
      <AppHeader />

      <!-- Main Content -->
      <main class="px-5 py-8 md:px-8 lg:px-12 lg:py-9">
        <slot />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import '@/styles/onboarding.css'
import { computed, onMounted } from 'vue'
import { useAppStore } from '@/stores'
import { useAuthStore } from '@/stores/auth'
import { useOnboardingTour } from '@/composables/useOnboardingTour'
import { useOnboardingStore } from '@/stores/onboarding'
import AppSidebar from './AppSidebar.vue'
import AppHeader from './AppHeader.vue'
import Icon from '@/components/icons/Icon.vue'

const appStore = useAppStore()
const authStore = useAuthStore()
const sidebarCollapsed = computed(() => appStore.sidebarCollapsed)
const pageNotices = computed(() => appStore.pageNotices)
const isAdmin = computed(() => authStore.user?.role === 'admin')

function getNoticeIconName(type: string): 'checkCircle' | 'xCircle' | 'exclamationTriangle' | 'infoCircle' {
  switch (type) {
    case 'success':
      return 'checkCircle'
    case 'error':
      return 'xCircle'
    case 'warning':
      return 'exclamationTriangle'
    case 'info':
    default:
      return 'infoCircle'
  }
}

const { replayTour } = useOnboardingTour({
  storageKey: isAdmin.value ? 'admin_guide' : 'user_guide',
  autoStart: true
})

const onboardingStore = useOnboardingStore()

onMounted(() => {
  onboardingStore.setReplayCallback(replayTour)
})

defineExpose({ replayTour })
</script>

<style scoped>
.page-notice-enter-active,
.page-notice-leave-active {
  transition:
    opacity 0.52s ease,
    filter 0.52s ease,
    transform 0.52s cubic-bezier(0.16, 1, 0.3, 1);
}

.page-notice-enter-from,
.page-notice-leave-to {
  opacity: 0;
  filter: blur(3px);
  transform: translateY(-32px) scale(0.98);
}

.page-notice-enter-to,
.page-notice-leave-from {
  opacity: 1;
  filter: blur(0);
  transform: translateY(0) scale(1);
}

.page-notice-leave-active {
  transition-duration: 0.32s;
}
</style>
