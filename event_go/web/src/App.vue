<template>
  <NetworkStatus />
  <router-view />
</template>

<script setup lang="ts">
import NetworkStatus from '@/components/NetworkStatus.vue'
</script>

<style>
:root {
  font-family: Inter, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
  color: #303133;
  background: #f5f7fa;
  letter-spacing: 0;
}

* {
  box-sizing: border-box;
}

html,
body,
#app {
  min-width: 320px;
  min-height: 100%;
  margin: 0;
}

button,
input,
textarea,
select {
  font: inherit;
}

img {
  max-width: 100%;
}
</style>
