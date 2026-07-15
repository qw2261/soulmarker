<template>
  <div class="page">
    <NavBar />
    <el-main class="main">
      <div class="container">
        <div class="page-header">
          <div>
            <h2>通知中心</h2>
            <span>{{ total }} 条通知</span>
          </div>
          <el-button :disabled="notificationStore.unreadCount === 0 || markingAll" @click="markAllRead">
            {{ markingAll ? '处理中...' : '全部标为已读' }}
          </el-button>
        </div>

        <el-radio-group v-model="filter" aria-label="通知筛选" class="filters" @change="changeFilter">
          <el-radio value="all">全部</el-radio>
          <el-radio value="unread">未读</el-radio>
        </el-radio-group>

        <el-skeleton v-if="loading" :rows="6" animated />
        <PageLoadError v-else-if="error" :message="error" @retry="fetchNotifications" />
        <el-empty v-else-if="notifications.length === 0" :description="emptyDescription" />
        <template v-else>
          <article
            v-for="notification in notifications"
            :key="notification.id"
            class="notification-item"
            :class="{ unread: !notification.read_at }"
          >
            <button class="notification-content" type="button" @click="openNotification(notification)">
              <span class="notification-title">
                <i v-if="!notification.read_at" aria-label="未读" />
                <strong>{{ notification.title }}</strong>
              </span>
              <span class="notification-body">{{ notification.body }}</span>
              <time :datetime="notification.created_at">{{ formatDateTime(notification.created_at) }}</time>
            </button>
            <el-button
              v-if="!notification.read_at"
              text
              :disabled="markingIds.has(notification.id)"
              :aria-label="`标记“${notification.title}”为已读`"
              @click="markRead(notification)"
            >
              标为已读
            </el-button>
          </article>
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
import { useRouter } from 'vue-router'
import { listNotifications } from '@/api/notifications'
import type { Notification } from '@/api/types'
import NavBar from '@/components/NavBar.vue'
import PageLoadError from '@/components/PageLoadError.vue'
import Pagination from '@/components/Pagination.vue'
import { useNotificationStore } from '@/stores/notifications'
import { formatDateTime } from '@/utils/format'
import { requestErrorMessage } from '@/utils/request-error'

const router = useRouter()
const notificationStore = useNotificationStore()
const notifications = ref<Notification[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)
const filter = ref<'all' | 'unread'>('all')
const loading = ref(true)
const error = ref('')
const markingAll = ref(false)
const markingIds = ref(new Set<number>())
const emptyDescription = computed(() => filter.value === 'unread' ? '暂无未读通知' : '暂无通知')

async function fetchNotifications() {
  loading.value = true
  error.value = ''
  try {
    const response = await listNotifications({
      unread_only: filter.value === 'unread',
      page: page.value,
      page_size: pageSize.value,
    })
    notifications.value = response.data || []
    total.value = response.total || 0
    await notificationStore.fetchUnreadCount()
  } catch (cause) {
    notifications.value = []
    total.value = 0
    error.value = requestErrorMessage(cause, '通知加载失败，请稍后重试')
  } finally {
    loading.value = false
  }
}

async function markRead(notification: Notification) {
  if (notification.read_at || markingIds.value.has(notification.id)) return
  markingIds.value.add(notification.id)
  try {
    await notificationStore.markRead(notification.id)
    notification.read_at = new Date().toISOString()
    if (filter.value === 'unread') await fetchNotifications()
  } catch {
    // The shared API client already presents the request failure to the user.
  } finally {
    markingIds.value.delete(notification.id)
  }
}

async function markAllRead() {
  markingAll.value = true
  try {
    await notificationStore.markAllRead()
    const readAt = new Date().toISOString()
    notifications.value.forEach((notification) => {
      notification.read_at ||= readAt
    })
    if (filter.value === 'unread') await fetchNotifications()
  } catch {
    // The shared API client already presents the request failure to the user.
  } finally {
    markingAll.value = false
  }
}

async function openNotification(notification: Notification) {
  await markRead(notification)
  if (notification.action_url) await router.push(notification.action_url)
}

function changeFilter() {
  page.value = 1
  fetchNotifications()
}

function onPageChange(value: number) {
  page.value = value
  fetchNotifications()
}

function onSizeChange(value: number) {
  pageSize.value = value
  page.value = 1
  fetchNotifications()
}

onMounted(fetchNotifications)
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
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 16px;
}

.page-header h2 {
  margin: 0 0 4px;
}

.page-header span,
.notification-content time {
  color: #909399;
  font-size: 13px;
}

.filters {
  margin-bottom: 12px;
}

.notification-item {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 16px;
  border-bottom: 1px solid #ebeef5;
  background: #fff;
}

.notification-item.unread {
  background: #ecf5ff;
}

.notification-content {
  display: grid;
  flex: 1;
  min-width: 0;
  gap: 7px;
  padding: 0;
  border: 0;
  background: transparent;
  color: inherit;
  cursor: pointer;
  text-align: left;
}

.notification-title {
  display: flex;
  align-items: center;
  gap: 8px;
}

.notification-title i {
  width: 8px;
  height: 8px;
  flex: 0 0 8px;
  border-radius: 50%;
  background: #409eff;
}

.notification-body {
  color: #606266;
  line-height: 1.6;
}

@media (max-width: 600px) {
  .main {
    padding: 16px 8px;
  }

  .page-header,
  .notification-item {
    align-items: flex-start;
  }

  .notification-item {
    flex-direction: column;
  }
}
</style>
