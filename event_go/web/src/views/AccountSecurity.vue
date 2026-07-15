<template>
  <div class="page">
    <NavBar />
    <el-main class="main">
      <el-card class="security-card">
        <h2>账户安全</h2>
        <el-descriptions :column="1" border>
          <el-descriptions-item label="登录联系方式">{{ user.user?.contact || '—' }}</el-descriptions-item>
          <el-descriptions-item label="恢复邮箱">
            <template v-if="hasVerifiedRecoveryEmail">
              {{ user.user?.recovery_email }}
              <el-tag type="success" size="small">已验证</el-tag>
            </template>
            <template v-else-if="user.user?.recovery_email">
              {{ user.user.recovery_email }}
              <el-tag type="warning" size="small">待验证</el-tag>
            </template>
            <span v-else>尚未绑定</span>
          </el-descriptions-item>
        </el-descriptions>

        <el-alert
          v-if="hasVerifiedRecoveryEmail"
          type="success"
          :closable="false"
          title="密码重置邮件将发送到已验证的恢复邮箱。"
        />
        <template v-else>
          <p class="hint">历史手机号账户绑定恢复邮箱后，仍使用原手机号登录；邮箱完成验证后用于安全验证和密码重置。</p>
          <el-alert v-if="sentTo" type="success" :closable="false" :title="`验证邮件已发送至 ${sentTo}`" />
          <el-form ref="formRef" :model="form" :rules="rules" label-position="top">
            <el-form-item label="恢复邮箱" prop="email">
              <el-input v-model="form.email" type="email" placeholder="name@example.com" />
            </el-form-item>
            <el-form-item label="当前密码" prop="password">
              <el-input v-model="form.password" type="password" show-password placeholder="用于确认是你本人" />
            </el-form-item>
            <el-button type="primary" :loading="loading" class="submit" @click="submit">发送验证邮件</el-button>
          </el-form>
        </template>
      </el-card>
    </el-main>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { ElMessage, type FormInstance } from 'element-plus'
import { requestRecoveryEmail } from '@/api/auth'
import { useUserStore } from '@/stores/user'
import NavBar from '@/components/NavBar.vue'

const user = useUserStore()
const formRef = ref<FormInstance>()
const loading = ref(false)
const sentTo = ref('')
const form = reactive({ email: user.user?.recovery_email || '', password: '' })
const hasVerifiedRecoveryEmail = computed(() => Boolean(
  user.user?.recovery_email && user.user?.recovery_email_verified_at,
))
const rules = {
  email: [
    { required: true, message: '请输入恢复邮箱', trigger: 'blur' },
    { type: 'email', message: '请输入有效邮箱', trigger: 'blur' },
  ],
  password: [{ required: true, message: '请输入当前密码', trigger: 'blur' }],
}

async function submit() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  loading.value = true
  try {
    await requestRecoveryEmail({ email: form.email, password: form.password })
    sentTo.value = form.email.trim().toLowerCase()
    form.password = ''
    ElMessage.success('验证邮件已发送')
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.page { min-height: 100vh; background: #f5f7fa; }
.main { display: flex; justify-content: center; padding-top: 40px; }
.security-card { width: min(560px, calc(100vw - 24px)); }
.security-card h2 { margin: 0 0 24px; text-align: center; }
.security-card :deep(.el-alert), .security-card form { margin-top: 20px; }
.hint { margin: 20px 0 0; color: #606266; line-height: 1.7; }
.submit { width: 100%; }
</style>
