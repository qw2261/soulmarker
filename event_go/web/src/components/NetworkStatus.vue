<template>
  <el-alert
    v-if="!online"
    class="network-status"
    title="网络连接已断开，恢复连接后可重新加载"
    type="warning"
    show-icon
    :closable="false"
  />
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'

const online = ref(typeof navigator === 'undefined' || navigator.onLine)
let wasOffline = !online.value

function syncNetworkStatus() {
  const nextOnline = navigator.onLine
  if (nextOnline && wasOffline) ElMessage.success('网络连接已恢复')
  online.value = nextOnline
  wasOffline = !nextOnline
}

onMounted(() => {
  window.addEventListener('online', syncNetworkStatus)
  window.addEventListener('offline', syncNetworkStatus)
})

onBeforeUnmount(() => {
  window.removeEventListener('online', syncNetworkStatus)
  window.removeEventListener('offline', syncNetworkStatus)
})
</script>

<style scoped>
.network-status {
  position: sticky;
  top: 0;
  z-index: 3000;
  border-radius: 0;
}
</style>
