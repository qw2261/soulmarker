<template>
  <div class="page"><NavBar /><el-main class="main"><el-result :icon="resultIcon" :title="title" :sub-title="subtitle">
    <template #extra><el-button type="primary" @click="$router.push('/workspace')">进入组织工作台</el-button></template>
  </el-result></el-main></div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import NavBar from '@/components/NavBar.vue'
import { acceptOrganizationInvitation } from '@/api/organizations'

const route = useRoute()
const status = ref<'loading' | 'success' | 'error'>('loading')
const resultIcon = computed(() => status.value === 'success' ? 'success' : status.value === 'error' ? 'error' : 'info')
const title = computed(() => status.value === 'loading' ? '正在接受邀请' : status.value === 'success' ? '已加入组织' : '邀请无法接受')
const subtitle = computed(() => status.value === 'error' ? '链接可能已过期、被撤销或不属于当前登录邮箱。' : '')
onMounted(async () => {
  const token = typeof route.query.token === 'string' ? route.query.token : ''
  if (!token) { status.value = 'error'; return }
  try { await acceptOrganizationInvitation(token); status.value = 'success' } catch { status.value = 'error' }
})
</script>

<style scoped>.page { min-height: 100vh; background: #f5f7fa; }.main { max-width: 760px; margin: 0 auto; padding-top: 48px; }</style>
