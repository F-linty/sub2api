<template>
  <Teleport to="body">
    <div
      class="pointer-events-none fixed bottom-5 right-5 z-[9999] space-y-2"
      aria-live="polite"
      aria-atomic="true"
    >
      <TransitionGroup
        enter-active-class="transition ease-out duration-200"
        enter-from-class="opacity-0 translate-y-2"
        enter-to-class="opacity-100 translate-x-0"
        leave-active-class="transition ease-in duration-150"
        leave-from-class="opacity-100 translate-y-0"
        leave-to-class="opacity-0 translate-y-2"
      >
        <div
          v-for="toast in toasts"
          :key="toast.id"
          :class="[
            'pointer-events-auto w-[min(340px,calc(100vw-2.5rem))] overflow-hidden rounded-xl',
            'border bg-white/96 shadow-[0_16px_42px_rgba(42,24,20,0.10)] backdrop-blur dark:bg-dark-900/95',
            getToastSurfaceClass(toast.type)
          ]"
        >
          <div class="p-3.5">
            <div class="flex items-start gap-3">
              <!-- Icon -->
              <div class="mt-0.5 flex-shrink-0">
                <span
                  :class="[
                    'flex h-7 w-7 items-center justify-center rounded-full',
                    getIconSurfaceClass(toast.type)
                  ]"
                >
                  <Icon
                    :name="getToastIconName(toast.type)"
                    size="sm"
                    :class="getIconColor(toast.type)"
                    aria-hidden="true"
                  />
                </span>
              </div>

              <!-- Content -->
              <div class="min-w-0 flex-1">
                <p v-if="toast.title" class="text-sm font-semibold text-[#2a1814] dark:text-white">
                  {{ toast.title }}
                </p>
                <p
                  :class="[
                    'text-sm leading-relaxed',
                    toast.title
                      ? 'mt-1 text-[#6f625e] dark:text-gray-300'
                      : 'font-medium text-[#2a1814] dark:text-white'
                  ]"
                >
                  {{ toast.message }}
                </p>
              </div>

              <!-- Close button -->
              <button
                @click="removeToast(toast.id)"
                class="-m-1 flex-shrink-0 rounded-md p-1 text-[#9a8b84] transition-colors hover:bg-[#f5ebe5] hover:text-[#2a1814] dark:text-gray-500 dark:hover:bg-dark-700 dark:hover:text-gray-300"
                aria-label="Close notification"
              >
                <Icon name="x" size="sm" />
              </button>
            </div>
          </div>
        </div>
      </TransitionGroup>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import Icon from '@/components/icons/Icon.vue'
import { useAppStore } from '@/stores/app'

const appStore = useAppStore()

const toasts = computed(() => appStore.toasts)

const getToastIconName = (type: string): 'checkCircle' | 'xCircle' | 'exclamationTriangle' | 'infoCircle' => {
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

const getIconColor = (type: string): string => {
  const colors: Record<string, string> = {
    success: 'text-[#3f7f55]',
    error: 'text-[#a94735]',
    warning: 'text-[#a86a22]',
    info: 'text-[#5b5d8e]'
  }
  return colors[type] || colors.info
}

const getToastSurfaceClass = (type: string): string => {
  const colors: Record<string, string> = {
    success: 'border-[#d7e5d7] dark:border-emerald-900/60',
    error: 'border-[#ead6cf] dark:border-red-900/60',
    warning: 'border-[#eadfca] dark:border-amber-900/60',
    info: 'border-[#dddceb] dark:border-indigo-900/60'
  }
  return colors[type] || colors.info
}

const getIconSurfaceClass = (type: string): string => {
  const colors: Record<string, string> = {
    success: 'bg-[#eef7ee] dark:bg-emerald-950/40',
    error: 'bg-[#f8e7df] dark:bg-red-950/40',
    warning: 'bg-[#fbf1df] dark:bg-amber-950/40',
    info: 'bg-[#eeeefb] dark:bg-indigo-950/40'
  }
  return colors[type] || colors.info
}

const removeToast = (id: string) => {
  appStore.hideToast(id)
}
</script>
