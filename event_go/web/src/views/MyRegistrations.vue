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
        <el-empty v-else-if="admissions.length === 0 && otherRegistrations.length === 0" description="暂无活动" />
        <template v-else>
          <section v-for="admission in admissions" :key="`admission-${admission.id}`" class="activity-item">
            <button class="activity-header" type="button" @click="$router.push(`/events/${admission.event_id}`)">
              <span>
                <strong>{{ admission.event_title }}</strong>
                <small>{{ formatDateTime(admission.event_time) }} · {{ admission.location }}</small>
              </span>
              <el-tag :type="activityStatus(admission).type">{{ activityStatus(admission).label }}</el-tag>
            </button>
            <AdmissionCredential :admission="admission" />
          </section>

          <section
            v-for="registration in otherRegistrations"
            :key="`registration-${registration.id}`"
            class="activity-item compact"
          >
            <button class="activity-header" type="button" @click="$router.push(`/events/${registration.event_id}`)">
              <span>
                <strong>{{ registration.event_title }}</strong>
                <small>{{ formatDateTime(registration.event_time) }} · {{ registration.location }}</small>
              </span>
              <el-tag>{{ eventStatusLabel(registration) }}</el-tag>
            </button>
            <p v-if="registration.ticket_name" class="ticket-line">门票：{{ registration.ticket_name }}</p>
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
import { computed, onMounted, ref } from 'vue'
import { listMyAdmissions, listMyRegistrations } from '@/api/events'
import type { MyAdmission, MyRegistration } from '@/api/types'
import { formatDateTime } from '@/utils/format'
import NavBar from '@/components/NavBar.vue'
import Pagination from '@/components/Pagination.vue'
import AdmissionCredential from '@/components/AdmissionCredential.vue'

const admissions = ref<MyAdmission[]>([])
const registrations = ref<MyRegistration[]>([])
const loading = ref(true)
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)

const admissionEventIDs = computed(() => new Set(admissions.value.map((item) => item.event_id)))
const otherRegistrations = computed(() =>
  registrations.value.filter((item) => !admissionEventIDs.value.has(item.event_id))
)

async function fetchActivities() {
  loading.value = true
  try {
    const [admissionResponse, registrationResponse] = await Promise.all([
      listMyAdmissions({ page: page.value, page_size: pageSize.value }),
      listMyRegistrations({ page: page.value, page_size: pageSize.value }),
    ])
    admissions.value = admissionResponse.data || []
    registrations.value = registrationResponse.data || []
    total.value = Math.max(admissionResponse.total || 0, registrationResponse.total || 0)
  } finally {
    loading.value = false
  }
}

function activityStatus(admission: MyAdmission): { label: string; type: 'success' | 'warning' | 'danger' | 'info' | 'primary' } {
  if (admission.status === 'revoked') return { label: '已取消', type: 'danger' }
  if (admission.checked_in_at) return { label: '已入场', type: 'success' }
  if (admission.event_status === 'cancelled') return { label: '活动已取消', type: 'danger' }
  if (admission.event_status === 'ended' || new Date(admission.event_time).getTime() < Date.now()) {
    return { label: '已结束', type: 'info' }
  }
  return { label: '待参加', type: 'primary' }
}

function eventStatusLabel(registration: MyRegistration) {
  if (registration.event_status === 'cancelled') return '活动已取消'
  if (registration.event_status === 'ended' || new Date(registration.event_time).getTime() < Date.now()) return '已结束'
  return '待参加'
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
