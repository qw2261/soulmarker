<template>
  <div class="page">
    <NavBar />
    <el-main class="main"><div class="container">
      <div class="header"><el-button text :icon="ArrowLeft" @click="$router.push('/workspace')">返回</el-button><h1>创建组织</h1></div>
      <el-card>
        <el-form ref="formRef" :model="form" :rules="rules" label-width="110px">
          <el-form-item label="组织名称" prop="name"><el-input v-model="form.name" placeholder="组织或团队名称" /></el-form-item>
          <el-form-item label="组织标识" prop="slug"><el-input v-model="form.slug" placeholder="例如: soulmark-team" /></el-form-item>
          <el-form-item label="主办方名称" prop="profile_name"><el-input v-model="form.profile_name" placeholder="公开展示名称" /></el-form-item>
          <el-form-item label="简介"><el-input v-model="form.profile_description" type="textarea" :rows="3" /></el-form-item>
          <el-form-item label="联系方式"><el-input v-model="form.profile_contact" placeholder="公开联系方式" /></el-form-item>
          <el-form-item label="地址"><el-input v-model="form.profile_address" /></el-form-item>
          <el-form-item label="网站"><el-input v-model="form.profile_website" placeholder="https://" /></el-form-item>
          <el-form-item label="Logo"><el-input v-model="form.profile_logo_url" placeholder="https://" /></el-form-item>
          <el-form-item label="标签"><el-input v-model="form.profile_tags" /></el-form-item>
          <el-form-item><el-button type="primary" :loading="submitting" @click="submit">创建并进入</el-button></el-form-item>
        </el-form>
      </el-card>
    </div></el-main>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref } from 'vue'
import { ArrowLeft } from '@element-plus/icons-vue'
import { ElMessage, type FormInstance } from 'element-plus'
import { useRouter } from 'vue-router'
import NavBar from '@/components/NavBar.vue'
import { createOrganization } from '@/api/organizations'

const router = useRouter()
const formRef = ref<FormInstance>()
const submitting = ref(false)
const form = reactive({ name: '', slug: '', profile_name: '', profile_description: '', profile_contact: '', profile_logo_url: '', profile_address: '', profile_website: '', profile_tags: '' })
const rules = {
  name: [{ required: true, message: '请输入组织名称', trigger: 'blur' }],
  slug: [{ required: true, pattern: /^[a-z0-9][a-z0-9-]*[a-z0-9]$/, message: '使用小写字母、数字和连字符，首尾需为字母或数字', trigger: 'blur' }],
  profile_name: [{ required: true, message: '请输入公开展示名称', trigger: 'blur' }],
}
async function submit() {
  if (!await formRef.value?.validate().catch(() => false)) return
  submitting.value = true
  try {
    const response = await createOrganization({ ...form })
    if (!response.data) return
    ElMessage.success('组织已创建')
    await router.push(`/workspace/${response.data.id}`)
  } finally { submitting.value = false }
}
</script>

<style scoped>
.page { min-height: 100vh; background: #f5f7fa; }.main { padding-top: 24px; }.container { max-width: 760px; margin: 0 auto; }.header { display: flex; align-items: center; gap: 16px; margin-bottom: 18px; }.header h1 { margin: 0; font-size: 24px; }
@media (max-width: 600px) { .main { padding: 16px 8px; } :deep(.el-form-item) { display: block; } }
</style>
