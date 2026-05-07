<template>
  <div class="page">
    <NavBar />
    <el-main class="main">
      <el-card class="auth-card">
        <h2>用户登录</h2>
        <el-form ref="formRef" :model="form" :rules="rules" label-width="80px">
          <el-form-item label="联系方式" prop="contact">
            <el-input v-model="form.contact" placeholder="手机号或邮箱" />
          </el-form-item>
          <el-form-item label="密码" prop="password">
            <el-input v-model="form.password" type="password" show-password placeholder="输入密码" />
          </el-form-item>
          <el-form-item>
            <el-button type="primary" @click="submit" :loading="loading" style="width: 100%">
              登录
            </el-button>
          </el-form-item>
        </el-form>
        <p class="switch">
          还没有账号？<router-link to="/register">去注册</router-link>
        </p>
      </el-card>
    </el-main>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, type FormInstance } from 'element-plus'
import { loginUser } from '@/api/auth'
import { useUserStore } from '@/stores/user'
import NavBar from '@/components/NavBar.vue'

const router = useRouter()
const userStore = useUserStore()

const formRef = ref<FormInstance>()
const loading = ref(false)

const form = reactive({
  contact: '',
  password: '',
})

const rules = {
  contact: [{ required: true, message: '请输入联系方式', trigger: 'blur' }],
  password: [{ required: true, message: '请输入密码', trigger: 'blur' }],
}

async function submit() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  loading.value = true
  try {
    const res = await loginUser({ ...form })
    if (res.code === 200 && res.data) {
      userStore.setAuth(res.data.token, res.data.user)
      ElMessage.success('登录成功')
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
