<template>
  <div class="page">
    <NavBar />
    <el-main class="main">
      <div class="container">
        <AdminToolbar />
        <div class="header">
          <div>
            <h2>门店管理</h2>
            <p>维护活动对外展示的主办方资料。</p>
          </div>
          <el-button type="primary" :icon="Plus" @click="$router.push('/admin/organizers/new')">
            创建门店
          </el-button>
        </div>

        <div class="table-scroll">
          <el-table :data="organizers" v-loading="loading" stripe class="data-table">
            <el-table-column prop="name" label="名称" min-width="160" />
            <el-table-column prop="contact" label="联系方式" min-width="180">
              <template #default="{ row }">{{ row.contact || '-' }}</template>
            </el-table-column>
            <el-table-column prop="address" label="地址" min-width="200">
              <template #default="{ row }">{{ row.address || '-' }}</template>
            </el-table-column>
            <el-table-column prop="event_count" label="活动数" width="90" />
            <el-table-column label="操作" width="190" fixed="right">
              <template #default="{ row }">
                <el-button size="small" :icon="Edit" @click="$router.push(`/admin/organizers/${row.id}/edit`)">
                  编辑
                </el-button>
                <el-button size="small" type="danger" :icon="Delete" @click="handleDelete(row)">
                  删除
                </el-button>
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
import { onMounted, ref } from 'vue'
import { Delete, Edit, Plus } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { deleteOrganizer, listOrganizers } from '@/api/organizers'
import type { Organizer } from '@/api/types'
import AdminToolbar from '@/components/AdminToolbar.vue'
import NavBar from '@/components/NavBar.vue'
import Pagination from '@/components/Pagination.vue'
import { confirmAction } from '@/utils/confirm'

const organizers = ref<Organizer[]>([])
const loading = ref(false)
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)

async function fetchOrganizers() {
  loading.value = true
  try {
    const response = await listOrganizers({ page: page.value, page_size: pageSize.value })
    organizers.value = response.data || []
    total.value = response.total || 0
  } finally {
    loading.value = false
  }
}

async function handleDelete(organizer: Organizer) {
  try {
    await confirmAction(
      `确定删除门店「${organizer.name}」？已有活动会保留并转为未归属状态。`,
      '确认删除',
      { type: 'warning' },
    )
    await deleteOrganizer(organizer.id)
    ElMessage.success('门店已删除，历史活动已保留')
    if (organizers.value.length === 1 && page.value > 1) page.value--
    await fetchOrganizers()
  } catch {
    // User cancelled the destructive action.
  }
}

function onPageChange(value: number) { page.value = value; fetchOrganizers() }
function onSizeChange(value: number) { pageSize.value = value; page.value = 1; fetchOrganizers() }

onMounted(fetchOrganizers)
</script>

<style scoped>
.page { min-height: 100vh; background: #f5f7fa; }
.main { padding-top: 24px; }
.container { max-width: 1080px; margin: 0 auto; }
.header { display: flex; align-items: flex-end; justify-content: space-between; gap: 16px; margin-bottom: 16px; }
.header h2, .header p { margin: 0; }
.header p { margin-top: 6px; color: #606266; font-size: 14px; }
.table-scroll { max-width: 100%; overflow-x: auto; }
.data-table { min-width: 820px; }

@media (max-width: 600px) {
  .main { padding: 16px 8px; }
  .header { align-items: stretch; flex-direction: column; }
}
</style>
