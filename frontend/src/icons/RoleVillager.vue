<template>
  <svg :width="size" :height="size" viewBox="0 0 120 120" fill="none" xmlns="http://www.w3.org/2000/svg">
    <!-- Hat -->
    <path d="M30 56 L60 20 L90 56 Z" fill="#7ab648" stroke="#4a8a2a" stroke-width="2.5" stroke-linejoin="round"/>
    <path d="M36 54 L60 26 L84 54 Z" fill="#9cd468"/>
    <!-- Hat band -->
    <rect x="28" y="53" width="64" height="7" rx="3" fill="#d4a24e" stroke="#b0842a" stroke-width="1.5"/>
    <!-- Head -->
    <ellipse cx="60" cy="76" rx="30" ry="28" fill="#ffe4cc" stroke="#c9986a" stroke-width="1.5"/>
    <!-- Blush left -->
    <ellipse cx="40" cy="82" rx="6" ry="3.5" fill="#ffb8b8" opacity="0.5"/>
    <!-- Blush right -->
    <ellipse cx="80" cy="82" rx="6" ry="3.5" fill="#ffb8b8" opacity="0.5"/>
    <!-- Left eye -->
    <ellipse cx="48" cy="72" rx="6" ry="6.5" fill="white" stroke="#4a4a4a" stroke-width="1.5"/>
    <ellipse cx="49" cy="72" rx="4" ry="4.5" fill="#4a3728"/>
    <ellipse cx="50.5" cy="70.5" rx="1.6" ry="1.8" fill="white"/>
    <!-- Right eye -->
    <ellipse cx="72" cy="72" rx="6" ry="6.5" fill="white" stroke="#4a4a4a" stroke-width="1.5"/>
    <ellipse cx="73" cy="72" rx="4" ry="4.5" fill="#4a3728"/>
    <ellipse cx="74.5" cy="70.5" rx="1.6" ry="1.8" fill="white"/>
    <!-- Nose -->
    <ellipse cx="60" cy="81" rx="2.5" ry="2" fill="#deb89a"/>
    <!-- Smile -->
    <path d="M52 86 Q56 92 60 88 Q64 92 68 86" stroke="#c9986a" stroke-width="2" fill="none" stroke-linecap="round"/>
    <!-- Little leaf on hat -->
    <path d="M60 20 Q66 14 70 20" stroke="#4a8a2a" stroke-width="2" fill="#7ab648" stroke-linecap="round"/>
    <line x1="64" y1="20" x2="64" y2="15" stroke="#4a8a2a" stroke-width="1.2" stroke-linecap="round"/>
    <!-- Freckles -->
    <circle cx="44" cy="78" r="1" fill="#deb89a"/>
    <circle cx="47" cy="80" r="1" fill="#deb89a"/>
    <circle cx="73" cy="78" r="1" fill="#deb89a"/>
    <circle cx="76" cy="80" r="1" fill="#deb89a"/>
  </svg>
</template>

<script setup>
defineProps({ size: { type: [Number, String], default: 48 } })
</script>
