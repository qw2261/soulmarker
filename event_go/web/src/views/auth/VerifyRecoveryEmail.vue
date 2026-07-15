<template>
  <div class="page">
    <NavBar />
    <el-main class="main">
      <el-card class="verify-card">
        <el-result v-if="loading" icon="info" title="正在验证恢复邮箱" />
        <el-result v-else-if="completed" icon="success" title="恢复邮箱已绑定" sub-title="为保护账户安全，旧登录会话已失效。">
          <template #extra><el-button type="primary" @click="$router.push('/login')">重新登录</el-button></template>
        </el-result>
        <el-result v-else icon="error" title="恢复邮箱验证失败" :sub-title="errorMessage">
          <template #extra>
            <el-button type="primary" @click="$router.push('/me/security')">重新申请</el-button>
          </template>
        </el-result>
      </el-card>
    </el-main>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { confirmRecoveryEmail } from '@/api/auth'
import { requestErrorMessage } from '@/utils/request-error'
import { useUserStore } from '@/stores/user'
import NavBar from '@/components/NavBar.vue'

const route = useRoute()
const user = useUserStore()
const token = computed(() => typeof route.query.token === 'string' ? route.query.token : '')
const loading = ref(true)
const completed = ref(false)
const errorMessage = ref('验证链接无效或已过期，请重新申请。')

onMounted(async () => {
  if (!token.value) {
    loading.value = false
    return
  }
  try {
    await confirmRecoveryEmail({ token: token.value })
    user.logout()
    completed.value = true
  } catch (cause) {
    errorMessage.value = requestErrorMessage(cause, errorMessage.value)
  } finally {
    loading.value = false
  }
})
</script>

<style scoped>
.page { min-height: 100vh; background: #f5f7fa; }
.main { display: flex; justify-content: center; padding-top: 40px; }
.verify-card { width: min(520px, calc(100vw - 24px)); }
</style>
