<template>
  <div class="page">
    <NavBar />
    <el-main class="main">
      <div class="container">
        <div class="header">
          <el-button text @click="$router.push('/admin/events')">← 返回管理</el-button>
          <h2>报名列表</h2>
        </div>

        <el-table :data="registrations" v-loading="loading" stripe>
          <el-table-column prop="id" label="ID" width="60" />
          <el-table-column prop="name" label="姓名" />
          <el-table-column prop="contact" label="联系方式" />
          <el-table-column label="门票" min-width="120">
            <template #default="{ row }">
              {{ row.ticket_name || '-' }}
            </template>
          </el-table-column>
          <el-table-column label="报名时间" width="160">
            <template #default="{ row }">
              {{ formatDateTime(row.created_at) }}
            </template>
          </el-table-column>
        </el-table>

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
import { useRoute } from 'vue-router'
import { listRegistrations } from '@/api/events'
import type { Registration } from '@/api/types'
import { formatDateTime } from '@/utils/format'
import NavBar from '@/components/NavBar.vue'
import Pagination from '@/components/Pagination.vue'

const route = useRoute()
const eventId = Number(route.params.id)

const registrations = ref<Registration[]>([])
const loading = ref(false)
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)

async function fetchData() {
  loading.value = true
  try {
    const res = await listRegistrations(eventId, { page: page.value, page_size: pageSize.value })
    registrations.value = res.data || []
    total.value = res.total || 0
  } finally {
    loading.value = false
  }
}

function onPageChange(p: number) { page.value = p; fetchData() }
function onSizeChange(s: number) { pageSize.value = s; page.value = 1; fetchData() }

onMounted(fetchData)
</script>

<style scoped>
.page { min-height: 100vh; background: #f5f7fa; }
.main { padding-top: 24px; }
.container { max-width: 800px; margin: 0 auto; }

.header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 16px;
}

.header h2 { margin: 0; }
</style>
