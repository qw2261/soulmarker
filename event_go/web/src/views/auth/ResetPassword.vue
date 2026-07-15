<template>
  <div class="page">
    <NavBar />
    <el-main class="main">
      <el-card class="auth-card">
        <el-result v-if="!token" icon="error" title="重置链接无效">
          <template #extra><el-button type="primary" @click="$router.push('/forgot-password')">重新申请</el-button></template>
        </el-result>
        <el-result v-else-if="completed" icon="success" title="密码已重置">
          <template #extra><el-button type="primary" @click="$router.push('/login')">重新登录</el-button></template>
        </el-result>
        <template v-else>
          <h2>设置新密码</h2>
          <el-form ref="formRef" :model="form" :rules="rules" label-width="90px">
            <el-form-item label="新密码" prop="password">
              <el-input v-model="form.password" type="password" show-password placeholder="8 到 72 位" />
            </el-form-item>
            <el-form-item label="确认密码" prop="confirmation">
              <el-input v-model="form.confirmation" type="password" show-password placeholder="再次输入" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" :loading="loading" class="submit" @click="submit">确认重置</el-button>
            </el-form-item>
          </el-form>
        </template>
      </el-card>
    </el-main>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useRoute } from 'vue-router'
import type { FormInstance, FormItemRule } from 'element-plus'
import { confirmPasswordReset } from '@/api/auth'
import NavBar from '@/components/NavBar.vue'

const route = useRoute()
const token = computed(() => typeof route.query.token === 'string' ? route.query.token : '')
const formRef = ref<FormInstance>()
const form = reactive({ password: '', confirmation: '' })
const loading = ref(false)
const completed = ref(false)

const confirmPassword: FormItemRule['validator'] = (_rule, value, callback) => {
  if (value !== form.password) callback(new Error('两次输入的密码不一致'))
  else callback()
}
const rules = {
  password: [
    { required: true, message: '请输入新密码', trigger: 'blur' },
    { min: 8, max: 72, message: '密码必须为 8 到 72 位', trigger: 'blur' },
  ],
  confirmation: [
    { required: true, message: '请再次输入密码', trigger: 'blur' },
    { validator: confirmPassword, trigger: 'blur' },
  ],
}

async function submit() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid || !token.value) return
  loading.value = true
  try {
    await confirmPasswordReset({ token: token.value, password: form.password })
    completed.value = true
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.page { min-height: 100vh; background: #f5f7fa; }
.main { display: flex; justify-content: center; padding-top: 40px; }
.auth-card { width: min(440px, calc(100vw - 24px)); }
.auth-card h2 { margin-bottom: 24px; text-align: center; }
.submit { width: 100%; }
</style>
