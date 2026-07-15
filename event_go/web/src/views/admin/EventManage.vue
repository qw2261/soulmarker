<template>
  <div class="page">
    <NavBar />
    <el-main class="main">
      <div class="container">
        <AdminToolbar />
        <div class="header">
          <div>
            <h2>活动管理</h2>
            <p>发布活动并管理票种、报名和入场核销。</p>
          </div>
          <el-button type="primary" :icon="Plus" @click="$router.push('/admin/events/new')">创建活动</el-button>
        </div>

        <div class="table-scroll">
        <el-table :data="events" v-loading="loading" stripe class="data-table">
          <el-table-column prop="id" label="ID" width="60" />
          <el-table-column prop="title" label="标题" min-width="150" />
          <el-table-column prop="organizer_name" label="门店" min-width="130">
            <template #default="{ row }">{{ row.organizer_name || '未归属' }}</template>
          </el-table-column>
          <el-table-column label="状态" width="90">
            <template #default="{ row }">
              <el-tag :type="EventStatusColors[row.status] as any" size="small">
                {{ EventStatusMap[row.status] || row.status }}
              </el-tag>
            </template>
          </el-table-column>
          <el-table-column label="时间" width="160">
            <template #default="{ row }">
              {{ formatDateTime(row.event_time) }}
            </template>
          </el-table-column>
          <el-table-column label="价格" width="80">
            <template #default="{ row }">
              {{ formatPrice(row.price) }}
            </template>
          </el-table-column>
          <el-table-column label="操作" width="350" fixed="right">
            <template #default="{ row }">
              <el-button size="small" :icon="Edit" @click="$router.push(`/admin/events/${row.id}/edit`)">编辑</el-button>
              <el-button size="small" :icon="Ticket" @click="$router.push(`/admin/events/${row.id}/tickets`)">票种</el-button>
              <el-button size="small" @click="$router.push(`/admin/events/${row.id}/registrations`)">报名 / 核销</el-button>
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
import { ref, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { Delete, Edit, Plus, Ticket } from '@element-plus/icons-vue'
import { listEvents, deleteEvent } from '@/api/events'
import type { Event } from '@/api/types'
import { EventStatusMap, EventStatusColors } from '@/api/types'
import { formatDateTime, formatPrice } from '@/utils/format'
import NavBar from '@/components/NavBar.vue'
import Pagination from '@/components/Pagination.vue'
import AdminToolbar from '@/components/AdminToolbar.vue'
import { confirmAction } from '@/utils/confirm'

const events = ref<Event[]>([])
const loading = ref(false)
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)

async function fetchEvents() {
  loading.value = true
  try {
    const res = await listEvents({ page: page.value, page_size: pageSize.value })
    events.value = res.data || []
    total.value = res.total || 0
  } finally {
    loading.value = false
  }
}

async function handleDelete(row: Event) {
  try {
    await confirmAction(`确定删除活动「${row.title}」？此操作不可恢复。`, '确认删除', {
      type: 'warning',
    })
    await deleteEvent(row.id)
    ElMessage.success('已删除')
    fetchEvents()
  } catch {
    // cancelled
  }
}

function onPageChange(p: number) { page.value = p; fetchEvents() }
function onSizeChange(s: number) { pageSize.value = s; page.value = 1; fetchEvents() }

onMounted(fetchEvents)
</script>

<style scoped>
.page { min-height: 100vh; background: #f5f7fa; }
.main { padding-top: 24px; }
.container { max-width: 1160px; margin: 0 auto; }

.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 16px;
}

.header h2, .header p { margin: 0; }
.header p { margin-top: 6px; color: #606266; font-size: 14px; }
.table-scroll { max-width: 100%; overflow-x: auto; }
.data-table { min-width: 1050px; }

@media (max-width: 600px) {
  .main { padding: 16px 8px; }
  .header { align-items: stretch; flex-direction: column; }
}
</style>
