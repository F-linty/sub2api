<template>
  <div class="beacon-mark" :class="[`beacon-mark-${size}`, { 'beacon-mark-inverted': inverted }]">
    <span class="beacon-dot"></span>
    <span class="beacon-beam"></span>
    <span class="beacon-base"></span>
  </div>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    size?: 'sm' | 'md' | 'lg'
    inverted?: boolean
  }>(),
  {
    size: 'md',
    inverted: false
  }
)
</script>

<style scoped>
.beacon-mark {
  --beacon-size: 2.25rem;
  --beacon-dot: #c45b45;
  --beacon-ink: #151515;
  position: relative;
  display: inline-flex;
  height: var(--beacon-size);
  width: var(--beacon-size);
  align-items: center;
  justify-content: center;
}

.beacon-mark-sm {
  --beacon-size: 1.75rem;
}

.beacon-mark-lg {
  --beacon-size: 4.5rem;
}

.beacon-mark-inverted {
  --beacon-ink: #f8fafc;
}

.beacon-dot {
  position: absolute;
  top: 12%;
  left: 50%;
  height: 18%;
  width: 18%;
  transform: translateX(-50%);
  border-radius: 999px;
  background: var(--beacon-dot);
  box-shadow: 0 0 18px rgba(196, 91, 69, 0.28);
}

.beacon-beam {
  position: absolute;
  top: 31%;
  left: 50%;
  height: 38%;
  width: 23%;
  transform: translateX(-50%);
  background: linear-gradient(180deg, rgba(196, 91, 69, 0.52), rgba(196, 91, 69, 0));
  clip-path: polygon(50% 0, 100% 100%, 0 100%);
}

.beacon-base {
  position: absolute;
  bottom: 18%;
  left: 50%;
  height: 5%;
  width: 45%;
  transform: translateX(-50%);
  border-radius: 999px;
  background: var(--beacon-ink);
}
</style>
