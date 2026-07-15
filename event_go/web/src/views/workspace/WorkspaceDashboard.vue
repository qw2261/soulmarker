<template>
  <div class="page"><NavBar /><el-main class="main"><div class="container">
    <WorkspaceToolbar />
    <div class="header">
      <div><h1>活动运营</h1><p>{{ detail?.profile.name || workspace.current?.organization_name }}</p></div>
      <el-button v-if="workspace.can('events.manage')" type="primary" :icon="Plus" @click="$router.push(`${base}/events/new`)">创建活动</el-button>
    </div>
    <div class="table-scroll"><el-table :data="events" v-loading="loading" stripe class="data-table">
      <el-table-column prop="title" label="活动" min-width="190" />
      <el-table-column label="状态" width="90"><template #default="{ row }"><el-tag :type="EventStatusColors[row.status] as any" size="small">{{ EventStatusMap[row.status] || row.status }}</el-tag></template></el-table-column>
      <el-table-column label="时间" min-width="170"><template #default="{ row }">{{ formatDateTime(row.event_time) }}</template></el-table-column>
      <el-table-column label="操作" min-width="330" fixed="right"><template #default="{ row }">
        <el-button v-if="workspace.can('events.manage')" size="small" :icon="Edit" @click="$router.push(`${base}/events/${row.id}/edit`)">编辑</el-button>
        <el-button v-if="workspace.can('tickets.manage')" size="small" :icon="Ticket" @click="$router.push(`${base}/events/${row.id}/tickets`)">票种</el-button>
        <el-button v-if="canOperate" size="small" @click="$router.push(`${base}/events/${row.id}/operations`)">{{ operationsLabel }}</el-button>
        <el-button v-if="workspace.can('events.manage')" size="small" type="danger" :icon="Delete" @click="remove(row)">删除</el-button>
      </template></el-table-column>
    </el-table></div>
    <el-empty v-if="!loading && !events.length" description="暂无活动" />
    <Pagination :total="total" :current-page="page" :page-size="pageSize" @page-change="changePage" @size-change="changeSize" />
  </div></el-main></div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Delete, Edit, Plus, Ticket } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { useRoute } from 'vue-router'
import NavBar from '@/components/NavBar.vue'
import Pagination from '@/components/Pagination.vue'
import WorkspaceToolbar from '@/components/WorkspaceToolbar.vue'
import { deleteOrganizationEvent, getOrganizationWorkspace, listOrganizationEvents } from '@/api/organizations'
import type { Event, OrganizationWorkspace } from '@/api/types'
import { EventStatusColors, EventStatusMap } from '@/api/types'
import { useWorkspaceStore } from '@/stores/workspace'
import { confirmAction } from '@/utils/confirm'
import { formatDateTime } from '@/utils/format'

const organizationId = Number(useRoute().params.organizationId)
const base = `/workspace/${organizationId}`
const workspace = useWorkspaceStore()
const detail = ref<OrganizationWorkspace>()
const events = ref<Event[]>([])
const loading = ref(false)
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const canOperate = computed(() => workspace.can('registrations.read') || workspace.can('checkins.manage') || workspace.can('registrations.export'))
const operationsLabel = computed(() => workspace.can('checkins.manage') ? (workspace.can('registrations.read') ? '报名 / 核销' : '核销') : '报名 / 导出')
async function load() {
  loading.value = true
  try {
    const [workspaceResponse, eventResponse] = await Promise.all([getOrganizationWorkspace(organizationId), listOrganizationEvents(organizationId, { page: page.value, page_size: pageSize.value })])
    detail.value = workspaceResponse.data
    events.value = eventResponse.data || []
    total.value = eventResponse.total || 0
  } finally { loading.value = false }
}
async function remove(event: Event) {
  try {
    await confirmAction(`确定删除活动「${event.title}」？此操作不可恢复。`, '确认删除', { type: 'warning' })
    await deleteOrganizationEvent(organizationId, event.id)
    ElMessage.success('活动已删除')
    await load()
  } catch { /* cancelled */ }
}
function changePage(value: number) { page.value = value; load() }
function changeSize(value: number) { pageSize.value = value; page.value = 1; load() }
onMounted(load)
</script>

<style scoped>
.page { min-height: 100vh; background: #f5f7fa; }.main { padding-top: 12px; }.container { max-width: 1160px; margin: 0 auto; }.header { display: flex; align-items: center; justify-content: space-between; gap: 16px; margin-bottom: 18px; }.header h1,.header p { margin: 0; }.header h1 { font-size: 24px; }.header p { margin-top: 5px; color: #606266; }.table-scroll { max-width: 100%; overflow-x: auto; }.data-table { min-width: 850px; }
@media (max-width: 600px) { .main { padding: 8px; }.header { align-items: stretch; flex-direction: column; } }
</style>
