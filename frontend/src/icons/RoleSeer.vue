<template>
  <svg :width="size" :height="size" viewBox="0 0 120 120" fill="none" xmlns="http://www.w3.org/2000/svg">
    <!-- Hood / cloak -->
    <path d="M60 12 Q20 30 18 72 Q18 96 38 106 L60 110 L82 106 Q102 96 102 72 Q100 30 60 12Z" fill="#5b4a9e" stroke="#3a2d6e" stroke-width="2.5"/>
    <path d="M60 16 Q26 34 24 70 Q24 90 40 100 L60 104 L80 100 Q96 90 96 70 Q94 34 60 16Z" fill="#7b68c8"/>
    <!-- Face -->
    <ellipse cx="60" cy="72" rx="30" ry="28" fill="#ffe4cc" stroke="#c9986a" stroke-width="1.5"/>
    <!-- Blush left -->
    <ellipse cx="40" cy="78" rx="6" ry="3.5" fill="#ffb8b8" opacity="0.5"/>
    <!-- Blush right -->
    <ellipse cx="80" cy="78" rx="6" ry="3.5" fill="#ffb8b8" opacity="0.5"/>
    <!-- Left eye -->
    <ellipse cx="48" cy="68" rx="7" ry="7.5" fill="white" stroke="#3a2d6e" stroke-width="1.5"/>
    <ellipse cx="49" cy="68" rx="4.5" ry="5" fill="#5b4a9e"/>
    <ellipse cx="50.5" cy="66.5" rx="1.8" ry="2" fill="white"/>
    <!-- Right eye -->
    <ellipse cx="72" cy="68" rx="7" ry="7.5" fill="white" stroke="#3a2d6e" stroke-width="1.5"/>
    <ellipse cx="73" cy="68" rx="4.5" ry="5" fill="#5b4a9e"/>
    <ellipse cx="74.5" cy="66.5" rx="1.8" ry="2" fill="white"/>
    <!-- Third eye / mystic gem -->
    <path d="M60 44 L54 52 L60 56 L66 52 Z" fill="#a8e6ff" stroke="#3a2d6e" stroke-width="1.5" stroke-linejoin="round"/>
    <ellipse cx="60" cy="50" rx="2.5" ry="3" fill="white" opacity="0.7"/>
    <!-- Sparkles around gem -->
    <line x1="50" y1="42" x2="48" y2="38" stroke="#a8e6ff" stroke-width="1.5" stroke-linecap="round"/>
    <line x1="70" y1="42" x2="72" y2="38" stroke="#a8e6ff" stroke-width="1.5" stroke-linecap="round"/>
    <line x1="60" y1="36" x2="60" y2="32" stroke="#a8e6ff" stroke-width="1.5" stroke-linecap="round"/>
    <circle cx="47" cy="37" r="1.2" fill="#a8e6ff"/>
    <circle cx="73" cy="37" r="1.2" fill="#a8e6ff"/>
    <!-- Small smile -->
    <path d="M54 82 Q60 88 66 82" stroke="#c9986a" stroke-width="2" fill="none" stroke-linecap="round"/>
    <!-- Nose -->
    <ellipse cx="60" cy="77" rx="2.5" ry="2" fill="#deb89a"/>
    <!-- Stars on cloak -->
    <circle cx="30" cy="58" r="1.8" fill="#e8d5ff"/>
    <circle cx="90" cy="58" r="1.8" fill="#e8d5ff"/>
    <circle cx="34" cy="88" r="1.3" fill="#e8d5ff"/>
    <circle cx="86" cy="88" r="1.3" fill="#e8d5ff"/>
    <circle cx="26" cy="74" r="1" fill="#e8d5ff"/>
    <circle cx="94" cy="74" r="1" fill="#e8d5ff"/>
  </svg>
</template>

<script setup>
defineProps({ size: { type: [Number, String], default: 48 } })
</script>
