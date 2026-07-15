<template>
  <div class="page">
    <NavBar />
    <el-main class="main">
      <div class="container">
        <AdminToolbar />
        <div class="header">
          <el-button text :icon="ArrowLeft" @click="$router.push('/admin/events')">返回</el-button>
          <div>
            <h2>票种管理</h2>
            <p>{{ event?.title || '加载活动中' }}</p>
          </div>
        </div>

        <section class="editor" aria-labelledby="ticket-editor-title">
          <h3 id="ticket-editor-title">{{ editingId ? '编辑票种' : '创建票种' }}</h3>
          <el-form ref="formRef" :model="form" :rules="rules" class="ticket-form" label-position="top">
            <el-form-item label="名称" prop="name">
              <el-input v-model="form.name" placeholder="例如：普通票" />
            </el-form-item>
            <el-form-item label="价格" prop="price">
              <el-input-number v-model="form.price" :min="0" :precision="2" :step="0.01" />
            </el-form-item>
            <el-form-item label="库存" prop="stock">
              <el-input-number v-model="form.stock" :min="editingId ? 0 : 1" :step="1" />
            </el-form-item>
            <el-form-item class="form-actions">
              <el-button type="primary" :loading="submitting" @click="submit">
                {{ editingId ? '保存票种' : '创建票种' }}
              </el-button>
              <el-button v-if="editingId" @click="resetForm">取消编辑</el-button>
            </el-form-item>
          </el-form>
        </section>

        <div class="table-scroll">
          <el-table :data="tickets" v-loading="loading" stripe class="data-table">
            <el-table-column prop="name" label="票种" min-width="180" />
            <el-table-column label="价格" width="120">
              <template #default="{ row }">{{ formatPrice(row.price) }}</template>
            </el-table-column>
            <el-table-column prop="stock" label="剩余库存" width="120" />
            <el-table-column label="操作" width="190" fixed="right">
              <template #default="{ row }">
                <el-button size="small" :icon="Edit" @click="startEdit(row)">编辑</el-button>
                <el-button size="small" type="danger" :icon="Delete" @click="handleDelete(row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>

        <Pagination
          :total="total"
          :current-page="page"
          :page-size="pageSize"
          @page-change="onPageChange"
          @size-change="onSizeChange"
        />
      </div>
    </el-main>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ArrowLeft, Delete, Edit } from '@element-plus/icons-vue'
import { ElMessage, type FormInstance } from 'element-plus'
import { useRoute } from 'vue-router'
import { getEvent } from '@/api/events'
import { createTicket, deleteTicket, listTickets, updateTicket } from '@/api/tickets'
import type { Event, Ticket } from '@/api/types'
import AdminToolbar from '@/components/AdminToolbar.vue'
import NavBar from '@/components/NavBar.vue'
import Pagination from '@/components/Pagination.vue'
import { formatPrice } from '@/utils/format'
import { confirmAction } from '@/utils/confirm'

const eventId = Number(useRoute().params.id)
const event = ref<Event>()
const tickets = ref<Ticket[]>([])
const loading = ref(false)
const submitting = ref(false)
const editingId = ref<number>()
const formRef = ref<FormInstance>()
const form = reactive({ name: '', price: 0, stock: 20 })
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const rules = {
  name: [{ required: true, message: '请输入票种名称', trigger: 'blur' }],
  price: [{ required: true, type: 'number', min: 0, message: '价格不能为负数', trigger: 'change' }],
  stock: [{ required: true, type: 'number', min: 0, message: '库存不能为负数', trigger: 'change' }],
}

async function fetchTickets() {
  loading.value = true
  try {
    const response = await listTickets(eventId, { page: page.value, page_size: pageSize.value })
    tickets.value = response.data || []
    total.value = response.total || 0
  } finally {
    loading.value = false
  }
}

function resetForm() {
  editingId.value = undefined
  Object.assign(form, { name: '', price: 0, stock: 20 })
  formRef.value?.clearValidate()
}

function startEdit(ticket: Ticket) {
  editingId.value = ticket.id
  Object.assign(form, { name: ticket.name, price: ticket.price, stock: ticket.stock })
  window.scrollTo({ top: 0, behavior: 'smooth' })
}

async function submit() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  if (!editingId.value && form.stock < 1) {
    ElMessage.warning('新票种库存必须大于 0')
    return
  }
  submitting.value = true
  try {
    if (editingId.value) {
      await updateTicket(eventId, editingId.value, { ...form })
      ElMessage.success('票种已更新')
    } else {
      await createTicket(eventId, { ...form })
      ElMessage.success('票种已创建')
    }
    resetForm()
    await fetchTickets()
  } finally {
    submitting.value = false
  }
}

async function handleDelete(ticket: Ticket) {
  try {
    await confirmAction(`确定删除票种「${ticket.name}」？历史报名会保留票种快照。`, '确认删除', { type: 'warning' })
    await deleteTicket(eventId, ticket.id)
    ElMessage.success('票种已删除')
    if (tickets.value.length === 1 && page.value > 1) page.value--
    await fetchTickets()
  } catch {
    // User cancelled the destructive action.
  }
}

function onPageChange(value: number) { page.value = value; fetchTickets() }
function onSizeChange(value: number) { pageSize.value = value; page.value = 1; fetchTickets() }

onMounted(async () => {
  const eventResponse = await getEvent(eventId)
  event.value = eventResponse.data
  await fetchTickets()
})
</script>

<style scoped>
.page { min-height: 100vh; background: #f5f7fa; }
.main { padding-top: 24px; }
.container { max-width: 960px; margin: 0 auto; }
.header { display: flex; align-items: center; gap: 12px; margin-bottom: 18px; }
.header h2, .header p { margin: 0; }
.header h2 { font-size: 22px; }
.header p { margin-top: 4px; color: #606266; font-size: 14px; }
.editor { margin-bottom: 24px; padding: 18px 0; border-top: 1px solid #dcdfe6; border-bottom: 1px solid #dcdfe6; }
.editor h3 { margin: 0 0 14px; font-size: 17px; }
.ticket-form { display: grid; grid-template-columns: minmax(180px, 1fr) 160px 160px auto; gap: 12px; align-items: end; }
.ticket-form :deep(.el-form-item) { margin-bottom: 0; }
.form-actions { min-width: 190px; }
.table-scroll { max-width: 100%; overflow-x: auto; }
.data-table { min-width: 680px; }

@media (max-width: 760px) {
  .main { padding: 16px 8px; }
  .ticket-form { grid-template-columns: 1fr 1fr; }
  .ticket-form > :first-child, .form-actions { grid-column: 1 / -1; }
}
</style>
