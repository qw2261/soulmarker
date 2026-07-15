<template>
  <div class="page">
    <NavBar />
    <el-main class="main">
      <div class="container">
        <div v-if="loading" class="loading">
          <el-skeleton :rows="5" animated />
        </div>

        <template v-else-if="event">
          <el-button text class="back-link" @click="$router.push('/')">
            ← 返回活动列表
          </el-button>
          <div class="event-header">
            <h1>{{ event.title }}</h1>
            <el-tag :type="EventStatusColors[event.status] as any" size="large">
              {{ EventStatusMap[event.status] || event.status }}
            </el-tag>
          </div>

          <div class="organizer-badge" v-if="event.organizer_name">
            <el-icon><Shop /></el-icon>
            主办方：
            <router-link :to="`/organizers/${event.organizer_id}`">
              {{ event.organizer_name }}
            </router-link>
          </div>

          <el-descriptions :column="1" border class="info">
            <el-descriptions-item label="时间">
              {{ formatDateTime(event.event_time) }}
            </el-descriptions-item>
            <el-descriptions-item label="地点">{{ event.location }}</el-descriptions-item>
            <el-descriptions-item label="价格">{{ formatPrice(event.price) }}</el-descriptions-item>
            <el-descriptions-item label="容量">{{ event.capacity }} 人</el-descriptions-item>
          </el-descriptions>

          <div class="description" v-if="event.description">
            <h3>活动介绍</h3>
            <p>{{ event.description }}</p>
          </div>

          <el-divider />

          <div class="section">
            <h3>报名</h3>
            <template v-if="registered">
              <el-alert
                title="你已成功报名该活动"
                type="success"
                show-icon
                :closable="false"
              />
              <div class="registered-actions">
                <el-button type="primary" @click="$router.push(`/events/${event.id}/discussion`)">
                  去讨论区
                </el-button>
                <el-button @click="handleCancel">取消报名</el-button>
              </div>
              <AdmissionCredential v-if="admission" :admission="admission" class="event-credential" />
            </template>
            <RegisterForm
              v-else-if="event.status === 'published' && userStore.isLoggedIn"
              :event-id="event.id"
              @registered="onRegistered"
            />
            <el-alert
              v-else-if="event.status === 'published'"
              title="登录后即可使用账户身份报名"
              type="info"
              show-icon
              :closable="false"
            >
              <template #default>
                <el-button type="primary" @click="$router.push('/login')">去登录</el-button>
              </template>
            </el-alert>
            <el-alert
              v-else
              :title="event.status === 'draft' ? '活动尚未发布，暂不可报名' : '活动已结束或取消，不可报名'"
              type="info"
              show-icon
              :closable="false"
            />
          </div>

          <el-divider />

          <div class="section">
            <div class="section-header">
              <h3>讨论区</h3>
              <el-button
                type="primary"
                size="small"
                @click="$router.push(`/events/${event.id}/discussion`)"
              >
                查看全部
              </el-button>
            </div>

            <div v-if="recentPosts.length > 0" class="post-list">
              <div
                v-for="post in recentPosts"
                :key="post.id"
                class="post-item"
                @click="$router.push(`/events/${event.id}/posts/${post.id}`)"
              >
                <h4>{{ post.title }}</h4>
                <p class="post-meta">
                  <span>{{ post.author_name }}</span>
                  <span>{{ post.reply_count }} 回复</span>
                  <span>{{ formatDate(post.created_at) }}</span>
                </p>
              </div>
            </div>
            <div v-else class="no-posts">
              暂无讨论，报名后可以发帖
            </div>
          </div>
        </template>
      </div>
    </el-main>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { Shop } from '@element-plus/icons-vue'
import { getEvent, getRegistrationStatus, cancelRegistration } from '@/api/events'
import { listPosts } from '@/api/posts'
import type { Admission, Event, Post } from '@/api/types'
import { EventStatusMap, EventStatusColors } from '@/api/types'
import { useUserStore } from '@/stores/user'
import { formatDateTime, formatPrice, formatDate } from '@/utils/format'
import NavBar from '@/components/NavBar.vue'
import RegisterForm from '@/components/RegisterForm.vue'
import AdmissionCredential from '@/components/AdmissionCredential.vue'
import { confirmAction } from '@/utils/confirm'

const route = useRoute()
const userStore = useUserStore()
const event = ref<Event | null>(null)
const loading = ref(true)
const registered = ref(false)
const admission = ref<Admission>()
const recentPosts = ref<Post[]>([])

async function fetchEvent() {
  const id = Number(route.params.id)
  try {
    const res = await getEvent(id)
    event.value = res.data || null
    if (userStore.isLoggedIn) {
      await checkRegistration(id)
    }
    await fetchRecentPosts(id)
  } finally {
    loading.value = false
  }
}

async function checkRegistration(eventId: number) {
  try {
    const res = await getRegistrationStatus(eventId)
    registered.value = res.data?.registered || false
    admission.value = res.data?.admission
  } catch {
    registered.value = false
    admission.value = undefined
  }
}

async function fetchRecentPosts(eventId: number) {
  try {
    const res = await listPosts(eventId, { page_size: 5 })
    recentPosts.value = res.data || []
  } catch {
    recentPosts.value = []
  }
}

function onRegistered(value?: Admission) {
  registered.value = true
  admission.value = value
}

async function handleCancel() {
  if (!event.value) return
  try {
    await confirmAction(
      '确定取消报名？取消后无法恢复，需在活动开始前 24 小时。',
      '确认取消',
      { type: 'warning' }
    )
    const res = await cancelRegistration(event.value.id)
    if (res.code === 200) {
      ElMessage.success('已取消报名')
      registered.value = false
      admission.value = undefined
    } else {
      ElMessage.error(res.message || '取消失败')
    }
  } catch {
    // cancelled
  }
}

onMounted(fetchEvent)
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
  max-width: 700px;
  margin: 0 auto;
}

.back-link {
  margin-bottom: 8px;
}

.organizer-badge {
  display: flex;
  align-items: center;
  gap: 4px;
  margin-bottom: 16px;
  color: #606266;
  font-size: 14px;
}

.organizer-badge a {
  color: #409eff;
  text-decoration: none;
}

.organizer-badge a:hover {
  text-decoration: underline;
}

.event-header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 20px;
}

.event-header h1 {
  margin: 0;
  font-size: 24px;
}

.info {
  margin-bottom: 20px;
}

.description {
  background: #fff;
  padding: 20px;
  border-radius: 8px;
  margin-bottom: 8px;
}

.description h3 {
  margin: 0 0 12px;
  font-size: 16px;
}

.description p {
  margin: 0;
  color: #606266;
  line-height: 1.8;
  white-space: pre-wrap;
}

.section {
  background: #fff;
  padding: 20px;
  border-radius: 8px;
}

.section h3 {
  margin: 0 0 16px;
  font-size: 16px;
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.section-header h3 {
  margin: 0;
}

.registered-actions {
  display: flex;
  gap: 12px;
  margin-top: 16px;
}

.event-credential {
  margin-top: 16px;
}

.loading {
  padding: 24px;
  background: #fff;
  border-radius: 8px;
}

.post-list {
  margin-top: 8px;
}

.post-item {
  padding: 12px 0;
  border-bottom: 1px solid #ebeef5;
  cursor: pointer;
}

.post-item:last-child {
  border-bottom: none;
}

.post-item h4 {
  margin: 0 0 6px;
  font-size: 14px;
  color: #303133;
}

.post-item:hover h4 {
  color: #409eff;
}

.post-meta {
  display: flex;
  gap: 16px;
  margin: 0;
  font-size: 12px;
  color: #909399;
}

.no-posts {
  color: #909399;
  font-size: 13px;
  text-align: center;
  padding: 16px 0 8px;
}
</style>
