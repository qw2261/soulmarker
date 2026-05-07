<template>
  <div class="page">
    <NavBar />

    <el-main class="main">
      <div class="container">
        <div class="search-bar">
          <el-input
            v-model="keyword"
            placeholder="搜索活动标题或描述..."
            clearable
            size="large"
            @keyup.enter="search"
            @clear="search"
          >
            <template #prefix>
              <el-icon><Search /></el-icon>
            </template>
          </el-input>
          <el-select v-model="statusFilter" placeholder="全部状态" clearable size="large" @change="search">
            <el-option label="草稿" value="draft" />
            <el-option label="已发布" value="published" />
            <el-option label="已取消" value="cancelled" />
            <el-option label="已结束" value="ended" />
          </el-select>
          <el-select v-model="priceType" placeholder="全部价格" clearable size="large" @change="search">
            <el-option label="免费" value="free" />
            <el-option label="付费" value="paid" />
          </el-select>
          <el-button type="primary" size="large" @click="search">搜索</el-button>
        </div>

        <div v-if="loading" class="loading">
          <el-skeleton :rows="3" animated />
        </div>

        <el-empty v-else-if="events.length === 0" description="暂无活动" />

        <template v-else>
          <EventCard
            v-for="event in events"
            :key="event.id"
            :event="event"
          />
          <Pagination
            :total="total"
            :current-page="page"
            :page-size="pageSize"
            @page-change="onPageChange"
            @size-change="onSizeChange"
          />
        </template>
      </div>
    </el-main>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Search } from '@element-plus/icons-vue'
import { listEvents } from '@/api/events'
import type { Event } from '@/api/types'
import NavBar from '@/components/NavBar.vue'
import EventCard from '@/components/EventCard.vue'
import Pagination from '@/components/Pagination.vue'

const events = ref<Event[]>([])
const loading = ref(false)
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)
const keyword = ref('')
const statusFilter = ref('')
const priceType = ref('')

async function fetchEvents() {
  loading.value = true
  try {
    const res = await listEvents({
      page: page.value,
      page_size: pageSize.value,
      q: keyword.value || undefined,
      status: statusFilter.value || undefined,
      price_type: priceType.value || undefined,
    })
    events.value = res.data || []
    total.value = res.total || 0
  } finally {
    loading.value = false
  }
}

function search() {
  page.value = 1
  fetchEvents()
}

function onPageChange(p: number) {
  page.value = p
  fetchEvents()
}

function onSizeChange(s: number) {
  pageSize.value = s
  page.value = 1
  fetchEvents()
}

onMounted(fetchEvents)
</script>

<style scoped>
.page {
  min-height: 100vh;
  background: #f5f7fa;
}

.main {
  padding-top: 24px;
}

.container {
  max-width: 800px;
  margin: 0 auto;
}

.search-bar {
  display: flex;
  gap: 12px;
  margin-bottom: 20px;
}

.search-bar .el-input {
  flex: 1;
}

.search-bar .el-select {
  width: 140px;
}

.loading {
  padding: 24px;
  background: #fff;
  border-radius: 8px;
}
</style>
