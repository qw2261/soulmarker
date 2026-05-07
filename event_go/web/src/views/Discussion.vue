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
          <el-button type="primary" @click="showForm = !showForm">
            {{ showForm ? '取消' : '发帖' }}
          </el-button>
        </div>

        <el-card v-if="showForm" class="post-form">
          <el-form :model="form" label-position="top">
            <el-form-item label="昵称">
              <el-input v-model="form.author_name" placeholder="你的昵称" />
            </el-form-item>
            <el-form-item label="联系方式（报名时的手机/邮箱）">
              <el-input v-model="form.author_contact" placeholder="用于验证报名身份" />
            </el-form-item>
            <el-form-item label="标题">
              <el-input v-model="form.title" placeholder="帖子标题" />
            </el-form-item>
            <el-form-item label="内容">
              <el-input
                v-model="form.content"
                type="textarea"
                :rows="4"
                placeholder="说点什么..."
              />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="createPost" :loading="posting">发布</el-button>
            </el-form-item>
          </el-form>
        </el-card>

        <el-empty v-if="!loading && posts.length === 0" description="暂无讨论" />

        <div v-for="post in posts" :key="post.id" class="post-card" @click="$router.push(`/events/${eventId}/posts/${post.id}`)">
          <h4>{{ post.title }}</h4>
          <p class="post-meta">
            <span>{{ post.author_name }}</span>
            <span>{{ formatDateTime(post.created_at) }}</span>
            <span>{{ post.reply_count }} 回复</span>
          </p>
        </div>

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
import { ref, reactive, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { listPosts, createPost as apiCreatePost } from '@/api/posts'
import type { Post } from '@/api/types'
import { useUserStore } from '@/stores/user'
import { formatDateTime } from '@/utils/format'
import NavBar from '@/components/NavBar.vue'
import Pagination from '@/components/Pagination.vue'

const route = useRoute()
const eventId = Number(route.params.id)
const userStore = useUserStore()

const posts = ref<Post[]>([])
const loading = ref(false)
const total = ref(0)
const page = ref(1)
const pageSize = ref(10)
const showForm = ref(false)
const posting = ref(false)

const form = reactive({
  author_name: userStore.user?.name || '',
  author_contact: userStore.user?.contact || '',
  title: '',
  content: '',
})

async function fetchPosts() {
  loading.value = true
  try {
    const res = await listPosts(eventId, { page: page.value, page_size: pageSize.value })
    posts.value = res.data || []
    total.value = res.total || 0
  } finally {
    loading.value = false
  }
}

async function createPost() {
  if (!form.title || !form.content || !form.author_name || !form.author_contact) {
    ElMessage.warning('请填写完整信息')
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
      fetchPosts()
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
  background: #fff;
  border-radius: 8px;
  padding: 16px 20px;
  margin-bottom: 8px;
  cursor: pointer;
}

.post-card:hover { background: #f0f5ff; }

.post-card h4 { margin: 0 0 8px; font-size: 15px; }

.post-meta {
  display: flex;
  gap: 16px;
  margin: 0;
  font-size: 12px;
  color: #909399;
}
</style>
