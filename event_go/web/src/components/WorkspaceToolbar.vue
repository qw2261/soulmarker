<template>
  <nav class="toolbar" aria-label="组织工作台导航">
    <div class="identity">
      <strong>{{ workspace.current?.organization_name || '组织工作台' }}</strong>
      <el-tag v-if="workspace.current" size="small" effect="plain">{{ roleLabel }}</el-tag>
    </div>
    <div class="links">
      <router-link :to="`/workspace/${organizationId}`">活动</router-link>
      <router-link v-if="workspace.can('members.read')" :to="`/workspace/${organizationId}/members`">成员</router-link>
      <router-link to="/workspace">切换组织</router-link>
    </div>
  </nav>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRoute } from 'vue-router'
import { useWorkspaceStore } from '@/stores/workspace'

const route = useRoute()
const workspace = useWorkspaceStore()
const organizationId = computed(() => Number(route.params.organizationId))
const roleLabels: Record<string, string> = {
  owner: '所有者', admin: '管理员', editor: '编辑', checker: '核销', finance: '财务',
}
const roleLabel = computed(() => roleLabels[workspace.current?.role || ''] || workspace.current?.role)
</script>

<style scoped>
.toolbar { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: 12px 0; margin-bottom: 20px; border-bottom: 1px solid #dcdfe6; }
.identity, .links { display: flex; align-items: center; gap: 12px; }
.links a { color: #606266; text-decoration: none; font-size: 14px; }
.links a:hover, .links .router-link-active { color: #409eff; }
@media (max-width: 600px) { .toolbar { align-items: flex-start; flex-direction: column; } .links { width: 100%; justify-content: space-between; } }
</style>
