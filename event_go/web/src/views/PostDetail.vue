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
            <el-form :model="form" label-position="top">
              <el-form-item label="昵称">
                <el-input v-model="form.author_name" placeholder="你的昵称" />
              </el-form-item>
              <el-form-item label="联系方式（报名时的手机/邮箱）">
                <el-input v-model="form.author_contact" placeholder="用于验证报名身份" />
              </el-form-item>
              <el-form-item label="回复内容">
                <el-input
                  v-model="form.content"
                  type="textarea"
                  :rows="3"
                  placeholder="写下你的回复..."
                />
              </el-form-item>
              <el-form-item>
                <el-button type="primary" @click="submitReply" :loading="replying">回复</el-button>
              </el-form-item>
            </el-form>
          </el-card>
        </template>
      </div>
    </el-main>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { getPost, createReply } from '@/api/posts'
import type { Post, Reply } from '@/api/types'
import { useUserStore } from '@/stores/user'
import { formatDateTime } from '@/utils/format'
import NavBar from '@/components/NavBar.vue'

const route = useRoute()
const eventId = Number(route.params.id)
const postId = Number(route.params.postId)
const userStore = useUserStore()

const post = ref<Post | null>(null)
const replies = ref<Reply[]>([])
const loading = ref(true)
const replying = ref(false)

const form = reactive({
  author_name: userStore.user?.name || '',
  author_contact: userStore.user?.contact || '',
  content: '',
})

async function fetchPost() {
  try {
    const res = await getPost(eventId, postId)
    if (res.data) {
      // The API returns post with replies embedded
      post.value = res.data
      replies.value = (res.data as any).replies || []
    }
  } finally {
    loading.value = false
  }
}

async function submitReply() {
  if (!form.content || !form.author_name || !form.author_contact) {
    ElMessage.warning('请填写完整信息')
    return
  }
  replying.value = true
  try {
    const res = await createReply(eventId, postId, { ...form })
    if (res.code === 201) {
      ElMessage.success('回复成功')
      form.content = ''
      fetchPost()
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

.loading { padding: 24px; background: #fff; border-radius: 8px; }
</style>
