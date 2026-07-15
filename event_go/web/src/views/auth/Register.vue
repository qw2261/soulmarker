<template>
  <div class="page">
    <NavBar />
    <el-main class="main">
      <el-card class="auth-card">
        <h2>用户注册</h2>
        <el-form ref="formRef" :model="form" :rules="rules" label-width="80px">
          <el-form-item label="昵称" prop="name">
            <el-input v-model="form.name" placeholder="你的名字" />
          </el-form-item>
          <el-form-item label="邮箱" prop="contact">
            <el-input v-model="form.contact" type="email" placeholder="name@example.com" />
          </el-form-item>
          <el-form-item label="密码" prop="password">
            <el-input v-model="form.password" type="password" show-password placeholder="8 到 72 位" />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="submit" :loading="loading" style="width: 100%">
              注册
            </el-button>
          </el-form-item>
        </el-form>
        <p class="switch">
          已有账号？<router-link to="/login">去登录</router-link>
        </p>
      </el-card>
    </el-main>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, type FormInstance } from 'element-plus'
import { registerUser } from '@/api/auth'
import { useUserStore } from '@/stores/user'
import NavBar from '@/components/NavBar.vue'

const router = useRouter()
const userStore = useUserStore()

const formRef = ref<FormInstance>()
const loading = ref(false)

const form = reactive({
  name: '',
  contact: '',
  password: '',
})

const rules = {
  name: [{ required: true, message: '请输入昵称', trigger: 'blur' }],
  contact: [
    { required: true, message: '请输入邮箱', trigger: 'blur' },
    { type: 'email', message: '请输入有效邮箱', trigger: 'blur' },
  ],
  password: [
    { required: true, message: '请输入密码', trigger: 'blur' },
    { min: 8, max: 72, message: '密码必须为 8 到 72 位', trigger: 'blur' },
  ],
}

async function submit() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  loading.value = true
  try {
    const res = await registerUser({ ...form })
    if (res.code === 201 && res.data) {
      userStore.setAuth(res.data.token, res.data.user)
      ElMessage.success('注册成功')
      router.push('/')
    }
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.page { min-height: 100vh; background: #f5f7fa; }
.main { padding-top: 40px; display: flex; justify-content: center; }
.auth-card { width: 400px; }
.auth-card h2 { text-align: center; margin-bottom: 24px; }
.switch { text-align: center; color: #909399; font-size: 13px; margin-top: 8px; }
</style>
