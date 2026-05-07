<template>
  <div class="page">
    <NavBar />
    <el-main class="main">
      <el-card class="login-card">
        <h2>管理员登录</h2>
        <el-form @submit.prevent="login">
          <el-form-item>
            <el-input
              v-model="token"
              type="password"
              placeholder="请输入 ADMIN_TOKEN"
              show-password
            />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="login" :loading="loading" style="width: 100%">
              登录
            </el-button>
          </el-form-item>
        </el-form>
        <p class="hint">Token 由服务端 ADMIN_TOKEN 环境变量配置</p>
      </el-card>
    </el-main>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useAuthStore } from '@/stores/auth'
import NavBar from '@/components/NavBar.vue'

const router = useRouter()
const auth = useAuthStore()
const token = ref('')
const loading = ref(false)

function login() {
  if (!token.value) {
    ElMessage.warning('请输入 Token')
    return
  }
  loading.value = true
  auth.login(token.value)
  ElMessage.success('登录成功')
  router.push('/admin/events')
  loading.value = false
}
</script>

<style scoped>
.page { min-height: 100vh; background: #f5f7fa; }
.main { padding-top: 60px; display: flex; justify-content: center; }

.login-card {
  width: 400px;
}

.login-card h2 {
  text-align: center;
  margin-bottom: 24px;
}

.hint {
  text-align: center;
  color: #909399;
  font-size: 12px;
}
</style>
