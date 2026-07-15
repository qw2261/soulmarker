<template>
  <div class="page">
    <NavBar />
    <el-main class="main">
      <div class="container">
        <div class="page-header">
          <h2>我的活动</h2>
          <span>{{ total }} 项</span>
        </div>
        <el-skeleton v-if="loading" :rows="6" animated />
        <el-empty v-else-if="activities.length === 0" description="暂无活动" />
        <template v-else>
          <section
            v-for="activity in activities"
            :key="`${activity.kind}-${activity.id}`"
            class="activity-item"
            :class="{ compact: !activity.admission }"
          >
            <button class="activity-header" type="button" @click="$router.push(`/events/${activity.event_id}`)">
              <span>
                <strong>{{ activity.event_title }}</strong>
                <small>{{ formatDateTime(activity.event_time) }} · {{ activity.location }}</small>
              </span>
              <el-tag :type="activityStatus(activity).type">{{ activityStatus(activity).label }}</el-tag>
            </button>
            <AdmissionCredential v-if="activity.admission" :admission="activity.admission" />
            <p v-else-if="activity.ticket_name" class="ticket-line">门票：{{ activity.ticket_name }}</p>
          </section>
        </template>
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
import { listMyActivities } from '@/api/events'
import type { MyActivity } from '@/api/types'
import { formatDateTime } from '@/utils/format'
import NavBar from '@/components/NavBar.vue'
import Pagination from '@/components/Pagination.vue'
import AdmissionCredential from '@/components/AdmissionCredential.vue'

const activities = ref<MyActivity[]>([])
const loading = ref(true)
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)

async function fetchActivities() {
  loading.value = true
  try {
    const response = await listMyActivities({ page: page.value, page_size: pageSize.value })
    activities.value = response.data || []
    total.value = response.total || 0
  } finally {
    loading.value = false
  }
}

function activityStatus(activity: MyActivity): { label: string; type: 'success' | 'warning' | 'danger' | 'info' | 'primary' } {
  if (activity.admission?.status === 'revoked') return { label: '已取消', type: 'danger' }
  if (activity.admission?.checked_in_at) return { label: '已入场', type: 'success' }
  if (activity.event_status === 'cancelled') return { label: '活动已取消', type: 'danger' }
  if (activity.event_status === 'ended' || new Date(activity.event_time).getTime() < Date.now()) {
    return { label: '已结束', type: 'info' }
  }
  return { label: '待参加', type: 'primary' }
}

function onPageChange(value: number) {
  page.value = value
  fetchActivities()
}

function onSizeChange(value: number) {
  pageSize.value = value
  page.value = 1
  fetchActivities()
}

onMounted(fetchActivities)
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
  max-width: 780px;
  margin: 0 auto;
}

.page-header {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  margin-bottom: 20px;
}

.page-header h2 {
  margin: 0;
}

.page-header span {
  color: #909399;
  font-size: 14px;
}

.activity-item {
  padding: 20px 0;
  border-bottom: 1px solid #dcdfe6;
}

.activity-item:first-of-type {
  padding-top: 0;
}

.activity-header {
  display: flex;
  width: 100%;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin: 0 0 14px;
  padding: 0;
  border: 0;
  background: transparent;
  color: inherit;
  cursor: pointer;
  text-align: left;
}

.activity-header span {
  display: grid;
  min-width: 0;
  gap: 6px;
}

.activity-header strong {
  font-size: 17px;
}

.activity-header small,
.ticket-line {
  color: #606266;
}

.compact .activity-header {
  margin-bottom: 0;
}

.ticket-line {
  margin: 8px 0 0;
}

@media (max-width: 600px) {
  .main {
    padding: 16px 8px;
  }

  .activity-header {
    flex-direction: column;
  }
}
</style>
