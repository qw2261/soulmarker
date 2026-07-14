<template>
  <div class="page">
    <NavBar />
    <el-main class="main">
      <div class="container">
        <h2>我的报名</h2>
        <el-skeleton v-if="loading" :rows="4" animated />
        <el-empty v-else-if="registrations.length === 0" description="暂无报名记录" />
        <el-card
          v-for="registration in registrations"
          v-else
          :key="registration.id"
          class="registration-card"
          shadow="hover"
          @click="$router.push(`/events/${registration.event_id}`)"
        >
          <div class="card-header">
            <h3>{{ registration.event_title }}</h3>
            <el-tag>{{ registration.event_status }}</el-tag>
          </div>
          <p>时间：{{ formatDateTime(registration.event_time) }}</p>
          <p>地点：{{ registration.location }}</p>
          <p v-if="registration.ticket_name">门票：{{ registration.ticket_name }}</p>
        </el-card>
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
import { listMyRegistrations } from '@/api/events'
import type { MyRegistration } from '@/api/types'
import { formatDateTime } from '@/utils/format'
import NavBar from '@/components/NavBar.vue'
import Pagination from '@/components/Pagination.vue'

const registrations = ref<MyRegistration[]>([])
const loading = ref(true)
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)

async function fetchRegistrations() {
  loading.value = true
  try {
    const response = await listMyRegistrations({ page: page.value, page_size: pageSize.value })
    registrations.value = response.data || []
    total.value = response.total || 0
  } finally {
    loading.value = false
  }
}

function onPageChange(value: number) {
  page.value = value
  fetchRegistrations()
}

function onSizeChange(value: number) {
  pageSize.value = value
  page.value = 1
  fetchRegistrations()
}

onMounted(fetchRegistrations)
</script>

<style scoped>
.page { min-height: 100vh; background: #f5f7fa; }
.main { padding-top: 24px; }
.container { max-width: 760px; margin: 0 auto; }
.registration-card { margin-bottom: 12px; cursor: pointer; }
.registration-card p { margin: 6px 0; color: #606266; }
.card-header { display: flex; align-items: center; justify-content: space-between; }
.card-header h3 { margin: 0 0 8px; }
</style>
