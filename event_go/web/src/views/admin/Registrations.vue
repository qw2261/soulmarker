<template>
  <div class="page">
    <NavBar />
    <el-main class="main">
      <div class="container">
        <div class="header">
          <el-button text @click="$router.push('/admin/events')">← 返回管理</el-button>
          <h2>报名列表</h2>
        </div>

        <section class="checkin-tool">
          <div>
            <h3>入场核销</h3>
            <p>使用扫码枪扫描后自动填入，也可粘贴完整凭证或 32 位凭证码。</p>
          </div>
          <div class="checkin-action">
            <el-input
              v-model="credential"
              placeholder="soulmark:admission:..."
              clearable
              @keyup.enter="submitCheckin"
            />
            <el-button type="primary" :icon="Aim" :loading="checkingIn" @click="submitCheckin">
              核销
            </el-button>
          </div>
          <el-alert
            v-if="checkinMessage"
            :title="checkinMessage"
            :type="checkinDuplicate ? 'warning' : 'success'"
            show-icon
            closable
            @close="checkinMessage = ''"
          />
        </section>

        <el-tabs v-model="activeTab">
          <el-tab-pane label="报名名单" name="registrations">
            <el-table :data="registrations" v-loading="loading" stripe>
              <el-table-column prop="id" label="ID" width="60" />
              <el-table-column prop="name" label="姓名" min-width="130" />
              <el-table-column prop="contact" label="联系方式" min-width="220" />
              <el-table-column label="门票" min-width="120">
                <template #default="{ row }">
                  {{ row.ticket_name || '-' }}
                </template>
              </el-table-column>
              <el-table-column label="报名时间" width="180">
                <template #default="{ row }">
                  {{ formatDateTime(row.created_at) }}
                </template>
              </el-table-column>
            </el-table>
          </el-tab-pane>
          <el-tab-pane :label="`核销记录 (${checkinTotal})`" name="checkins">
            <el-table :data="checkins" v-loading="checkinsLoading" stripe>
              <el-table-column prop="user_name" label="姓名" min-width="130" />
              <el-table-column prop="user_contact" label="联系方式" min-width="220" />
              <el-table-column label="核销时间" min-width="180">
                <template #default="{ row }">{{ formatDateTime(row.checked_in_at) }}</template>
              </el-table-column>
              <el-table-column prop="checked_in_by" label="操作方" width="130" />
            </el-table>
          </el-tab-pane>
        </el-tabs>

        <Pagination
          v-if="activeTab === 'registrations'"
          :total="total"
          :current-page="page"
          :page-size="pageSize"
          @page-change="onPageChange"
          @size-change="onSizeChange"
        />
        <Pagination
          v-else
          :total="checkinTotal"
          :current-page="checkinPage"
          :page-size="pageSize"
          @page-change="onCheckinPageChange"
          @size-change="onCheckinSizeChange"
        />
      </div>
    </el-main>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { Aim } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import { checkInAdmission, listCheckins, listRegistrations } from '@/api/events'
import type { Checkin, Registration } from '@/api/types'
import { formatDateTime } from '@/utils/format'
import NavBar from '@/components/NavBar.vue'
import Pagination from '@/components/Pagination.vue'

const route = useRoute()
const eventId = Number(route.params.id)

const registrations = ref<Registration[]>([])
const checkins = ref<Checkin[]>([])
const loading = ref(false)
const checkinsLoading = ref(false)
const total = ref(0)
const checkinTotal = ref(0)
const page = ref(1)
const checkinPage = ref(1)
const pageSize = ref(20)
const activeTab = ref('registrations')
const credential = ref('')
const checkingIn = ref(false)
const checkinMessage = ref('')
const checkinDuplicate = ref(false)

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

async function fetchCheckins() {
  checkinsLoading.value = true
  try {
    const response = await listCheckins(eventId, { page: checkinPage.value, page_size: pageSize.value })
    checkins.value = response.data || []
    checkinTotal.value = response.total || 0
  } finally {
    checkinsLoading.value = false
  }
}

async function submitCheckin() {
  const value = credential.value.trim()
  if (!value) {
    ElMessage.warning('请输入入场凭证')
    return
  }
  checkingIn.value = true
  try {
    const response = await checkInAdmission(eventId, value)
    checkinDuplicate.value = response.data?.already_checked_in || false
    checkinMessage.value = checkinDuplicate.value ? '该凭证此前已核销，未重复记录' : '核销成功'
    credential.value = ''
    await fetchCheckins()
  } finally {
    checkingIn.value = false
  }
}

function onPageChange(p: number) { page.value = p; fetchData() }
function onSizeChange(s: number) { pageSize.value = s; page.value = 1; fetchData() }
function onCheckinPageChange(value: number) { checkinPage.value = value; fetchCheckins() }
function onCheckinSizeChange(value: number) { pageSize.value = value; checkinPage.value = 1; fetchCheckins() }

onMounted(() => Promise.all([fetchData(), fetchCheckins()]))
</script>

<style scoped>
.page { min-height: 100vh; background: #f5f7fa; }
.main { padding-top: 24px; }
.container { max-width: 960px; margin: 0 auto; }

.header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 16px;
}

.header h2 { margin: 0; }

.checkin-tool {
  display: grid;
  gap: 14px;
  margin-bottom: 20px;
  padding: 18px 0;
  border-top: 1px solid #dcdfe6;
  border-bottom: 1px solid #dcdfe6;
}

.checkin-tool h3,
.checkin-tool p {
  margin: 0;
}

.checkin-tool p {
  margin-top: 6px;
  color: #606266;
  font-size: 14px;
}

.checkin-action {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 10px;
}

@media (max-width: 600px) {
  .main { padding: 16px 8px; }
  .checkin-action { grid-template-columns: 1fr; }
}
</style>
