<template>
  <div class="page">
    <NavBar />
    <el-main class="main">
      <el-card class="auth-card">
        <el-result v-if="submitted" icon="success" title="请检查邮箱">
          <template #sub-title>如果该邮箱已注册，重置链接会发送到对应邮箱。</template>
          <template #extra><el-button type="primary" @click="$router.push('/login')">返回登录</el-button></template>
        </el-result>
        <template v-else>
          <h2>找回密码</h2>
          <el-form ref="formRef" :model="form" :rules="rules" label-width="70px">
            <el-form-item label="邮箱" prop="contact">
              <el-input v-model="form.contact" type="email" placeholder="name@example.com" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" :loading="loading" class="submit" @click="submit">发送重置链接</el-button>
            </el-form-item>
          </el-form>
          <p class="switch"><router-link to="/login">返回登录</router-link></p>
        </template>
      </el-card>
    </el-main>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import type { FormInstance } from 'element-plus'
import { requestPasswordReset } from '@/api/auth'
import NavBar from '@/components/NavBar.vue'

const formRef = ref<FormInstance>()
const form = reactive({ contact: '' })
const loading = ref(false)
const submitted = ref(false)
const rules = {
  contact: [
    { required: true, message: '请输入邮箱', trigger: 'blur' },
    { type: 'email', message: '请输入有效邮箱', trigger: 'blur' },
  ],
}

async function submit() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  loading.value = true
  try {
    await requestPasswordReset({ contact: form.contact.trim().toLowerCase() })
    submitted.value = true
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.page { min-height: 100vh; background: #f5f7fa; }
.main { display: flex; justify-content: center; padding-top: 40px; }
.auth-card { width: min(400px, calc(100vw - 24px)); }
.auth-card h2 { margin-bottom: 24px; text-align: center; }
.submit { width: 100%; }
.switch { margin-top: 8px; color: #909399; font-size: 13px; text-align: center; }
</style>
