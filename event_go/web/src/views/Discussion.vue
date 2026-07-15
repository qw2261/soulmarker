<template>
  <div class="page">
    <NavBar />
    <el-main class="main">
      <div class="container">
        <div class="header">
          <el-button text @click="$router.push(`/events/${eventId}`)">
            ← 返回活动
          </el-button>
          <h2>讨论区</h2>
          <el-button type="primary" @click="togglePostForm">
            {{ showForm ? '取消' : '发帖' }}
          </el-button>
        </div>

        <div v-if="loading" class="loading"><el-skeleton :rows="5" animated /></div>
        <PageLoadError v-else-if="error" :message="error" @retry="fetchPosts" />

        <template v-else>
          <el-card v-if="showForm" class="post-form">
            <el-form :model="form" label-position="top" @submit.prevent="createPost">
              <el-form-item label="标题">
                <el-input v-model="form.title" placeholder="帖子标题" maxlength="120" show-word-limit />
              </el-form-item>
              <el-form-item label="内容">
                <el-input
                  v-model="form.content"
                  type="textarea"
                  :rows="4"
                  placeholder="说点什么..."
                  maxlength="4000"
                  show-word-limit
                />
              </el-form-item>
              <el-form-item>
                <el-button native-type="submit" type="primary" :loading="posting">发布</el-button>
              </el-form-item>
            </el-form>
          </el-card>

          <el-empty v-if="posts.length === 0" description="暂无讨论，报名后可以发布第一条帖子" />

          <button
            v-for="post in posts"
            :key="post.id"
            type="button"
            class="post-card"
            @click="$router.push(`/events/${eventId}/posts/${post.id}`)"
          >
            <h4>{{ post.title }}</h4>
            <p class="post-meta">
              <span>{{ post.author_name }}</span>
              <span>{{ formatDateTime(post.created_at) }}</span>
              <span>{{ post.reply_count }} 回复</span>
            </p>
          </button>

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
import { ref, reactive, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { listPosts, createPost as apiCreatePost } from '@/api/posts'
import type { Post } from '@/api/types'
import { useUserStore } from '@/stores/user'
import { formatDateTime } from '@/utils/format'
import NavBar from '@/components/NavBar.vue'
import Pagination from '@/components/Pagination.vue'
import PageLoadError from '@/components/PageLoadError.vue'
import { requestErrorMessage } from '@/utils/request-error'

const route = useRoute()
const router = useRouter()
const eventId = Number(route.params.id)
const userStore = useUserStore()

const posts = ref<Post[]>([])
const loading = ref(false)
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)
const showForm = ref(false)
const posting = ref(false)
const error = ref('')

const form = reactive({
  title: '',
  content: '',
})

function togglePostForm() {
  if (!userStore.isLoggedIn) {
    ElMessage.warning('请先登录并报名后再发帖')
    router.push({ path: '/login', query: { redirect: route.fullPath } })
    return
  }
  showForm.value = !showForm.value
}

async function fetchPosts() {
  loading.value = true
  error.value = ''
  try {
    const res = await listPosts(eventId, { page: page.value, page_size: pageSize.value })
    posts.value = res.data || []
    total.value = res.total || 0
  } catch (cause) {
    posts.value = []
    total.value = 0
    error.value = requestErrorMessage(cause, '讨论区加载失败，请稍后重试')
  } finally {
    loading.value = false
  }
}

async function createPost() {
  if (!form.title || !form.content) {
    ElMessage.warning('请填写标题和内容')
    return
  }
  posting.value = true
  try {
    const res = await apiCreatePost(eventId, { ...form })
    if (res.code === 201) {
      ElMessage.success('发帖成功')
      showForm.value = false
      form.title = ''
      form.content = ''
      await fetchPosts()
    }
  } finally {
    posting.value = false
  }
}

function onPageChange(p: number) { page.value = p; fetchPosts() }
function onSizeChange(s: number) { pageSize.value = s; page.value = 1; fetchPosts() }

onMounted(fetchPosts)
</script>

<style scoped>
.page { min-height: 100vh; background: #f5f7fa; }
.main { padding-top: 24px; }
.container { max-width: 700px; margin: 0 auto; }

.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 20px;
}

.header h2 { margin: 0; font-size: 20px; }

.post-form { margin-bottom: 16px; }

.post-card {
  display: block;
  width: 100%;
  border: 0;
  background: #fff;
  border-radius: 8px;
  padding: 16px 20px;
  margin-bottom: 8px;
  cursor: pointer;
  color: inherit;
  text-align: left;
}

.post-card:hover { background: #f0f5ff; }
.post-card:focus-visible { outline: 3px solid #79bbff; outline-offset: 2px; }

.post-card h4 { margin: 0 0 8px; font-size: 15px; }

.post-meta {
  display: flex;
  gap: 16px;
  margin: 0;
  font-size: 12px;
  color: #909399;
}

.loading { padding: 24px; background: #fff; border-radius: 8px; }

@media (max-width: 640px) {
  .main { padding: 16px 8px; }
  .header { align-items: flex-start; flex-wrap: wrap; gap: 8px; }
  .header h2 { order: -1; width: 100%; }
  .post-card { padding: 16px; }
  .post-meta { flex-wrap: wrap; gap: 6px 12px; }
}
</style>
