<template>
  <div class="page">
    <NavBar />
    <el-main class="main">
      <div class="container">
        <el-button text @click="$router.push(`/events/${eventId}/discussion`)">
          ← 返回讨论区
        </el-button>

        <div v-if="loading" class="loading">
          <el-skeleton :rows="3" animated />
        </div>

        <PageLoadError v-else-if="error" :message="error" @retry="fetchPost" />

        <template v-else-if="post">
          <h2>{{ post.title }}</h2>
          <p class="post-meta">
            <span>{{ post.author_name }}</span>
            <span>{{ formatDateTime(post.created_at) }}</span>
          </p>
          <div class="post-content">{{ post.content }}</div>

          <el-divider />

          <h3>回复 ({{ replies.length }})</h3>

          <div v-if="replies.length === 0" class="no-reply">暂无回复</div>

          <div v-for="reply in replies" :key="reply.id" class="reply-item">
            <div class="reply-meta">
              <strong>{{ reply.author_name }}</strong>
              <span>{{ formatDateTime(reply.created_at) }}</span>
            </div>
            <p class="reply-content">{{ reply.content }}</p>
          </div>

          <el-card class="reply-form">
            <el-alert
              v-if="!userStore.isLoggedIn"
              title="登录并报名后可以参与回复"
              type="info"
              show-icon
              :closable="false"
              class="reply-login-hint"
            >
              <template #default>
                <el-button text type="primary" @click="goToLogin">去登录</el-button>
              </template>
            </el-alert>
            <el-form :model="form" label-position="top" @submit.prevent="submitReply">
              <el-form-item label="回复内容">
                <el-input
                  v-model="form.content"
                  type="textarea"
                  :rows="3"
                  placeholder="写下你的回复..."
                  maxlength="4000"
                  show-word-limit
                />
              </el-form-item>
              <el-form-item>
                <el-button native-type="submit" type="primary" :loading="replying" :disabled="!userStore.isLoggedIn">回复</el-button>
              </el-form-item>
            </el-form>
          </el-card>
        </template>

        <el-empty v-else description="帖子不存在或已被删除" />
      </div>
    </el-main>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { getPost, createReply } from '@/api/posts'
import type { Post, Reply } from '@/api/types'
import { useUserStore } from '@/stores/user'
import { formatDateTime } from '@/utils/format'
import NavBar from '@/components/NavBar.vue'
import PageLoadError from '@/components/PageLoadError.vue'
import { requestErrorMessage } from '@/utils/request-error'

const route = useRoute()
const router = useRouter()
const eventId = Number(route.params.id)
const postId = Number(route.params.postId)
const userStore = useUserStore()

const post = ref<Post | null>(null)
const replies = ref<Reply[]>([])
const loading = ref(true)
const replying = ref(false)
const error = ref('')

const form = reactive({
  content: '',
})

function goToLogin() {
  router.push({ path: '/login', query: { redirect: route.fullPath } })
}

async function fetchPost() {
  loading.value = true
  error.value = ''
  try {
    const res = await getPost(eventId, postId)
    if (res.data) {
      post.value = res.data.post
      replies.value = res.data.replies
    }
  } catch (cause) {
    post.value = null
    replies.value = []
    error.value = requestErrorMessage(cause, '帖子加载失败，请稍后重试')
  } finally {
    loading.value = false
  }
}

async function submitReply() {
  if (!userStore.isLoggedIn) {
    ElMessage.warning('请先登录并报名后再回复')
    return
  }
  if (!form.content) {
    ElMessage.warning('请填写回复内容')
    return
  }
  replying.value = true
  try {
    const res = await createReply(eventId, postId, { ...form })
    if (res.code === 201) {
      ElMessage.success('回复成功')
      form.content = ''
      await fetchPost()
    }
  } finally {
    replying.value = false
  }
}

onMounted(fetchPost)
</script>

<style scoped>
.page { min-height: 100vh; background: #f5f7fa; }
.main { padding-top: 24px; }
.container { max-width: 700px; margin: 0 auto; }

.post-meta {
  display: flex;
  gap: 16px;
  color: #909399;
  font-size: 13px;
  margin-bottom: 16px;
}

.post-content {
  background: #fff;
  padding: 20px;
  border-radius: 8px;
  line-height: 1.8;
  white-space: pre-wrap;
}

.reply-item {
  background: #fff;
  padding: 16px 20px;
  border-radius: 8px;
  margin-bottom: 8px;
}

.reply-meta {
  display: flex;
  gap: 12px;
  margin-bottom: 8px;
  font-size: 13px;
  color: #909399;
}

.reply-content {
  margin: 0;
  line-height: 1.6;
}

.no-reply {
  color: #909399;
  text-align: center;
  padding: 24px;
}

.reply-form { margin-top: 16px; }
.reply-login-hint { margin-bottom: 16px; }

.loading { padding: 24px; background: #fff; border-radius: 8px; }

@media (max-width: 640px) {
  .main { padding: 16px 8px; }
  h2 { overflow-wrap: anywhere; font-size: 20px; }
  .post-meta, .reply-meta { flex-wrap: wrap; gap: 6px 12px; }
  .post-content, .reply-item { padding: 16px; }
}
</style>
