<template>
  <div class="page"><NavBar /><el-main class="main"><div class="container"><WorkspaceToolbar />
    <div class="header"><el-button text :icon="ArrowLeft" @click="$router.push(base)">返回</el-button><div class="title"><h1>{{ canRegistrations ? '报名与核销' : '入场核销' }}</h1><p>{{ event?.title }}</p></div><el-button v-if="canExport" :icon="Download" :loading="exporting" @click="exportCSV">导出 CSV</el-button></div>
    <section v-if="canCheckins" class="checkin"><div><h2>入场核销</h2><p>扫描、粘贴完整凭证或输入 32 位凭证码。</p></div><div class="checkin-action"><el-input v-model="credential" placeholder="soulmark:admission:..." clearable @keyup.enter="submitCheckin" /><el-button type="primary" :icon="Aim" :loading="checkingIn" @click="submitCheckin">核销</el-button></div><el-alert v-if="checkinMessage" :title="checkinMessage" :type="duplicate ? 'warning' : 'success'" show-icon closable @close="checkinMessage = ''" /></section>
    <el-tabs v-model="tab">
      <el-tab-pane v-if="canRegistrations" label="报名名单" name="registrations"><div class="table-scroll"><el-table :data="registrations" v-loading="loading" stripe class="data-table"><el-table-column prop="name" label="姓名" min-width="130" /><el-table-column prop="contact" label="联系方式" min-width="220" /><el-table-column label="票种" min-width="120"><template #default="{ row }">{{ row.ticket_name || '-' }}</template></el-table-column><el-table-column label="报名时间" min-width="180"><template #default="{ row }">{{ formatDateTime(row.created_at) }}</template></el-table-column></el-table></div></el-tab-pane>
      <el-tab-pane v-if="canCheckins" :label="`核销记录 (${checkinTotal})`" name="checkins"><div class="table-scroll"><el-table :data="checkins" v-loading="checkinsLoading" stripe class="data-table"><el-table-column prop="user_name" label="姓名" min-width="130" /><el-table-column prop="user_contact" label="联系方式" min-width="220" /><el-table-column label="核销时间" min-width="180"><template #default="{ row }">{{ formatDateTime(row.checked_in_at) }}</template></el-table-column><el-table-column prop="checked_in_by" label="操作方" width="130" /></el-table></div></el-tab-pane>
    </el-tabs>
    <Pagination v-if="tab === 'registrations'" :total="total" :current-page="page" :page-size="pageSize" @page-change="changePage" @size-change="changeSize" />
    <Pagination v-else :total="checkinTotal" :current-page="checkinPage" :page-size="pageSize" @page-change="changeCheckinPage" @size-change="changeCheckinSize" />
  </div></el-main></div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { Aim, ArrowLeft, Download } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { useRoute } from 'vue-router'
import NavBar from '@/components/NavBar.vue'
import Pagination from '@/components/Pagination.vue'
import WorkspaceToolbar from '@/components/WorkspaceToolbar.vue'
import { checkInOrganizationAdmission, getOrganizationEvent, listOrganizationCheckins, listOrganizationRegistrations } from '@/api/organizations'
import type { Checkin, Event, Registration } from '@/api/types'
import { useWorkspaceStore } from '@/stores/workspace'
import { downloadRegistrationCSV } from '@/utils/registration-export'
import { formatDateTime } from '@/utils/format'

const route = useRoute(); const organizationId = Number(route.params.organizationId); const eventId = Number(route.params.id); const base = `/workspace/${organizationId}`; const workspace = useWorkspaceStore()
const canRegistrations = computed(() => workspace.can('registrations.read')); const canCheckins = computed(() => workspace.can('checkins.manage')); const canExport = computed(() => workspace.can('registrations.export'))
const event = ref<Event>(); const registrations = ref<Registration[]>([]); const checkins = ref<Checkin[]>([]); const loading = ref(false); const checkinsLoading = ref(false); const total = ref(0); const checkinTotal = ref(0); const page = ref(1); const checkinPage = ref(1); const pageSize = ref(20); const tab = ref(canRegistrations.value ? 'registrations' : 'checkins'); const credential = ref(''); const checkingIn = ref(false); const checkinMessage = ref(''); const duplicate = ref(false); const exporting = ref(false)
async function loadRegistrations() { if (!canRegistrations.value) return; loading.value = true; try { const response = await listOrganizationRegistrations(organizationId, eventId, { page: page.value, page_size: pageSize.value }); registrations.value = response.data || []; total.value = response.total || 0 } finally { loading.value = false } }
async function loadCheckins() { if (!canCheckins.value) return; checkinsLoading.value = true; try { const response = await listOrganizationCheckins(organizationId, eventId, { page: checkinPage.value, page_size: pageSize.value }); checkins.value = response.data || []; checkinTotal.value = response.total || 0 } finally { checkinsLoading.value = false } }
async function submitCheckin() { const value = credential.value.trim(); if (!value) { ElMessage.warning('请输入入场凭证'); return }; checkingIn.value = true; try { const response = await checkInOrganizationAdmission(organizationId, eventId, value); duplicate.value = response.data?.already_checked_in || false; checkinMessage.value = duplicate.value ? '该凭证此前已核销，未重复记录' : '核销成功'; credential.value = ''; await loadCheckins() } finally { checkingIn.value = false } }
async function exportCSV() { exporting.value = true; try { const all: Registration[] = []; let currentPage = 1; while (true) { const response = await listOrganizationRegistrations(organizationId, eventId, { page: currentPage, page_size: 100 }); all.push(...(response.data || [])); if (all.length >= (response.total || 0) || !response.data?.length) break; currentPage++ }; downloadRegistrationCSV(all, eventId); ElMessage.success(`已导出 ${all.length} 条报名记录`) } finally { exporting.value = false } }
function changePage(value: number) { page.value = value; loadRegistrations() } function changeSize(value: number) { pageSize.value = value; page.value = 1; loadRegistrations() } function changeCheckinPage(value: number) { checkinPage.value = value; loadCheckins() } function changeCheckinSize(value: number) { pageSize.value = value; checkinPage.value = 1; loadCheckins() }
onMounted(async () => { event.value = (await getOrganizationEvent(organizationId, eventId)).data; await Promise.all([loadRegistrations(), loadCheckins()]) })
</script>

<style scoped>.page { min-height: 100vh; background: #f5f7fa; }.main { padding-top: 12px; }.container { max-width: 960px; margin: 0 auto; }.header { display: flex; align-items: center; gap: 16px; margin-bottom: 18px; }.title { flex: 1; }.header h1,.header p { margin: 0; }.header h1 { font-size: 24px; }.header p { margin-top: 4px; color: #606266; }.checkin { display: grid; gap: 14px; padding: 18px 0; margin-bottom: 18px; border-top: 1px solid #dcdfe6; border-bottom: 1px solid #dcdfe6; }.checkin h2,.checkin p { margin: 0; }.checkin h2 { font-size: 18px; }.checkin p { margin-top: 5px; color: #606266; }.checkin-action { display: grid; grid-template-columns: 1fr auto; gap: 10px; }.table-scroll { overflow-x: auto; }.data-table { min-width: 720px; }@media (max-width: 600px) { .main { padding: 8px; }.header { align-items: stretch; flex-direction: column; }.checkin-action { grid-template-columns: 1fr; } }</style>
