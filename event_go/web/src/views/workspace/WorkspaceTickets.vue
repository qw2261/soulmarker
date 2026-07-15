<template>
  <div class="page"><NavBar /><el-main class="main"><div class="container"><WorkspaceToolbar />
    <div class="header"><el-button text :icon="ArrowLeft" @click="$router.push(base)">返回</el-button><div><h1>票种管理</h1><p>{{ event?.title }}</p></div></div>
    <section class="editor"><h2>{{ editingId ? '编辑票种' : '创建票种' }}</h2><el-form ref="formRef" :model="form" :rules="rules" class="ticket-form" label-position="top"><el-form-item label="名称" prop="name"><el-input v-model="form.name" placeholder="例如：普通票" /></el-form-item><el-form-item label="价格" prop="price"><el-input-number v-model="form.price" :min="0" :precision="2" /></el-form-item><el-form-item label="库存" prop="stock"><el-input-number v-model="form.stock" :min="editingId ? 0 : 1" /></el-form-item><el-form-item class="actions"><el-button type="primary" :loading="submitting" @click="submit">{{ editingId ? '保存票种' : '创建票种' }}</el-button><el-button v-if="editingId" @click="reset">取消编辑</el-button></el-form-item></el-form></section>
    <div class="table-scroll"><el-table :data="tickets" v-loading="loading" stripe class="data-table"><el-table-column prop="name" label="票种" min-width="180" /><el-table-column label="价格" width="120"><template #default="{ row }">{{ formatPrice(row.price) }}</template></el-table-column><el-table-column prop="stock" label="库存" width="100" /><el-table-column label="操作" width="190"><template #default="{ row }"><el-button size="small" :icon="Edit" @click="edit(row)">编辑</el-button><el-button size="small" type="danger" :icon="Delete" @click="remove(row)">删除</el-button></template></el-table-column></el-table></div>
    <Pagination :total="total" :current-page="page" :page-size="pageSize" @page-change="changePage" @size-change="changeSize" />
  </div></el-main></div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ArrowLeft, Delete, Edit } from '@element-plus/icons-vue'
import { ElMessage, type FormInstance } from 'element-plus'
import { useRoute } from 'vue-router'
import NavBar from '@/components/NavBar.vue'
import Pagination from '@/components/Pagination.vue'
import WorkspaceToolbar from '@/components/WorkspaceToolbar.vue'
import { createOrganizationTicket, deleteOrganizationTicket, getOrganizationEvent, listOrganizationTickets, updateOrganizationTicket } from '@/api/organizations'
import type { Event, Ticket } from '@/api/types'
import { confirmAction } from '@/utils/confirm'
import { formatPrice } from '@/utils/format'

const route = useRoute(); const organizationId = Number(route.params.organizationId); const eventId = Number(route.params.id); const base = `/workspace/${organizationId}`
const event = ref<Event>(); const tickets = ref<Ticket[]>([]); const loading = ref(false); const submitting = ref(false); const editingId = ref<number>(); const formRef = ref<FormInstance>(); const form = reactive({ name: '', price: 0, stock: 20 }); const total = ref(0); const page = ref(1); const pageSize = ref(20)
const rules = { name: [{ required: true, message: '请输入票种名称', trigger: 'blur' }], price: [{ required: true, type: 'number', min: 0, message: '价格不能为负数', trigger: 'change' }], stock: [{ required: true, type: 'number', min: 0, message: '库存不能为负数', trigger: 'change' }] }
async function load() { loading.value = true; try { const response = await listOrganizationTickets(organizationId, eventId, { page: page.value, page_size: pageSize.value }); tickets.value = response.data || []; total.value = response.total || 0 } finally { loading.value = false } }
function reset() { editingId.value = undefined; Object.assign(form, { name: '', price: 0, stock: 20 }); formRef.value?.clearValidate() }
function edit(ticket: Ticket) { editingId.value = ticket.id; Object.assign(form, { name: ticket.name, price: ticket.price, stock: ticket.stock }); window.scrollTo({ top: 0, behavior: 'smooth' }) }
async function submit() { if (!await formRef.value?.validate().catch(() => false) || (!editingId.value && form.stock < 1)) return; submitting.value = true; try { if (editingId.value) { await updateOrganizationTicket(organizationId, eventId, editingId.value, { ...form }); ElMessage.success('票种已更新') } else { await createOrganizationTicket(organizationId, eventId, { ...form }); ElMessage.success('票种已创建') }; reset(); await load() } finally { submitting.value = false } }
async function remove(ticket: Ticket) { try { await confirmAction(`确定删除票种「${ticket.name}」？`, '确认删除', { type: 'warning' }); await deleteOrganizationTicket(organizationId, eventId, ticket.id); ElMessage.success('票种已删除'); await load() } catch { /* cancelled */ } }
function changePage(value: number) { page.value = value; load() } function changeSize(value: number) { pageSize.value = value; page.value = 1; load() }
onMounted(async () => { event.value = (await getOrganizationEvent(organizationId, eventId)).data; await load() })
</script>

<style scoped>.page { min-height: 100vh; background: #f5f7fa; }.main { padding-top: 12px; }.container { max-width: 960px; margin: 0 auto; }.header { display: flex; align-items: center; gap: 16px; }.header h1,.header p { margin: 0; }.header h1 { font-size: 24px; }.header p { margin-top: 4px; color: #606266; }.editor { margin: 22px 0; padding: 18px 0; border-top: 1px solid #dcdfe6; border-bottom: 1px solid #dcdfe6; }.editor h2 { margin: 0 0 14px; font-size: 18px; }.ticket-form { display: grid; grid-template-columns: 1fr 160px 160px auto; gap: 12px; align-items: end; }.ticket-form :deep(.el-form-item) { margin-bottom: 0; }.table-scroll { overflow-x: auto; }.data-table { min-width: 680px; }@media (max-width: 760px) { .main { padding: 8px; }.ticket-form { grid-template-columns: 1fr 1fr; }.ticket-form > :first-child,.actions { grid-column: 1 / -1; } }</style>
