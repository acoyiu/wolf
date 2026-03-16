<template>
  <svg :width="size" :height="size" viewBox="0 0 120 120" fill="none" xmlns="http://www.w3.org/2000/svg">
    <!-- Left ear -->
    <path d="M28 52 L18 18 L44 40 Z" fill="#8b8b8b" stroke="#4a4a4a" stroke-width="2.5" stroke-linejoin="round"/>
    <path d="M30 48 L23 24 L42 42 Z" fill="#c9a0b0"/>
    <!-- Right ear -->
    <path d="M92 52 L102 18 L76 40 Z" fill="#8b8b8b" stroke="#4a4a4a" stroke-width="2.5" stroke-linejoin="round"/>
    <path d="M90 48 L97 24 L78 42 Z" fill="#c9a0b0"/>
    <!-- Head -->
    <ellipse cx="60" cy="68" rx="38" ry="36" fill="#b0b0b0" stroke="#4a4a4a" stroke-width="2.5"/>
    <!-- Cheek tufts left -->
    <path d="M22 62 Q16 68 20 76 Q24 72 28 66 Z" fill="#d0d0d0" stroke="#4a4a4a" stroke-width="1.5"/>
    <path d="M24 70 Q17 77 22 84 Q26 79 30 73 Z" fill="#d0d0d0" stroke="#4a4a4a" stroke-width="1.5"/>
    <!-- Cheek tufts right -->
    <path d="M98 62 Q104 68 100 76 Q96 72 92 66 Z" fill="#d0d0d0" stroke="#4a4a4a" stroke-width="1.5"/>
    <path d="M96 70 Q103 77 98 84 Q94 79 90 73 Z" fill="#d0d0d0" stroke="#4a4a4a" stroke-width="1.5"/>
    <!-- Face patch -->
    <ellipse cx="60" cy="72" rx="26" ry="24" fill="#e8e8e8"/>
    <!-- Left eye -->
    <ellipse cx="45" cy="62" rx="7" ry="7.5" fill="white" stroke="#4a4a4a" stroke-width="1.5"/>
    <ellipse cx="46.5" cy="62" rx="4.5" ry="5" fill="#2d2d2d"/>
    <ellipse cx="48" cy="60" rx="1.8" ry="2" fill="white"/>
    <!-- Right eye -->
    <ellipse cx="75" cy="62" rx="7" ry="7.5" fill="white" stroke="#4a4a4a" stroke-width="1.5"/>
    <ellipse cx="76.5" cy="62" rx="4.5" ry="5" fill="#2d2d2d"/>
    <ellipse cx="78" cy="60" rx="1.8" ry="2" fill="white"/>
    <!-- Nose -->
    <ellipse cx="60" cy="76" rx="6" ry="4.5" fill="#3d3d3d"/>
    <ellipse cx="61.5" cy="75" rx="2" ry="1.2" fill="#6a6a6a"/>
    <!-- Mouth -->
    <path d="M54 80 Q57 85 60 80 Q63 85 66 80" stroke="#4a4a4a" stroke-width="2" fill="none" stroke-linecap="round"/>
    <!-- Brows -->
    <path d="M38 53 Q43 49 50 52" stroke="#4a4a4a" stroke-width="2.2" fill="none" stroke-linecap="round"/>
    <path d="M82 53 Q77 49 70 52" stroke="#4a4a4a" stroke-width="2.2" fill="none" stroke-linecap="round"/>
    <!-- Forehead mark -->
    <path d="M55 46 L60 38 L65 46" stroke="#8b8b8b" stroke-width="2" fill="none" stroke-linecap="round" stroke-linejoin="round"/>
  </svg>
</template>

<script setup>
defineProps({ size: { type: [Number, String], default: 48 } })
</script>
