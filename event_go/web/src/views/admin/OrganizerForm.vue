<template>
  <div class="page">
    <NavBar />
    <el-main class="main">
      <div class="container">
        <AdminToolbar />
        <div class="header">
          <el-button text :icon="ArrowLeft" @click="$router.push('/admin/organizers')">返回</el-button>
          <h2>{{ isEdit ? '编辑门店' : '创建门店' }}</h2>
        </div>

        <el-card class="form-card">
          <el-form ref="formRef" :model="form" :rules="rules" label-width="92px">
            <el-form-item label="名称" prop="name">
              <el-input v-model="form.name" placeholder="门店或主办方名称" />
            </el-form-item>
            <el-form-item label="简介">
              <el-input v-model="form.description" type="textarea" :rows="3" placeholder="对外展示的简介" />
            </el-form-item>
            <el-form-item label="联系方式">
              <el-input v-model="form.contact" placeholder="邮箱或电话" />
            </el-form-item>
            <el-form-item label="地址">
              <el-input v-model="form.address" placeholder="线下地址" />
            </el-form-item>
            <el-form-item label="官网">
              <el-input v-model="form.website" placeholder="https://example.com" />
            </el-form-item>
            <el-form-item label="Logo URL">
              <el-input v-model="form.logo_url" placeholder="https://example.com/logo.png" />
            </el-form-item>
            <el-form-item label="标签">
              <el-input v-model="form.tags" placeholder="教育,讲座" />
            </el-form-item>
            <el-form-item>
              <el-button type="primary" :loading="submitting" @click="submit">
                {{ isEdit ? '保存修改' : '创建门店' }}
              </el-button>
              <el-button @click="$router.push('/admin/organizers')">取消</el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </div>
    </el-main>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ArrowLeft } from '@element-plus/icons-vue'
import { ElMessage, type FormInstance } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'
import {
  createOrganizer,
  getOrganizer,
  updateOrganizer,
  type CreateOrganizerReq,
} from '@/api/organizers'
import AdminToolbar from '@/components/AdminToolbar.vue'
import NavBar from '@/components/NavBar.vue'

const route = useRoute()
const router = useRouter()
const isEdit = Boolean(route.params.id)
const formRef = ref<FormInstance>()
const submitting = ref(false)
const form = reactive<CreateOrganizerReq>({
  name: '', description: '', contact: '', logo_url: '', address: '', website: '', tags: '',
})
const rules = { name: [{ required: true, message: '请输入门店名称', trigger: 'blur' }] }

onMounted(async () => {
  if (!isEdit) return
  const response = await getOrganizer(Number(route.params.id))
  if (response.data) {
    const organizer = response.data
    Object.assign(form, {
      name: organizer.name,
      description: organizer.description,
      contact: organizer.contact,
      logo_url: organizer.logo_url,
      address: organizer.address,
      website: organizer.website,
      tags: organizer.tags,
    })
  }
})

async function submit() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  submitting.value = true
  try {
    if (isEdit) {
      await updateOrganizer(Number(route.params.id), form)
      ElMessage.success('门店已更新')
    } else {
      await createOrganizer(form)
      ElMessage.success('门店已创建')
    }
    await router.push('/admin/organizers')
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.page { min-height: 100vh; background: #f5f7fa; }
.main { padding-top: 24px; }
.container { max-width: 760px; margin: 0 auto; }
.header { display: flex; align-items: center; gap: 12px; margin-bottom: 16px; }
.header h2 { margin: 0; font-size: 22px; }
.form-card { width: 100%; }

@media (max-width: 600px) {
  .main { padding: 16px 8px; }
  .form-card :deep(.el-card__body) { padding: 18px 12px; }
}
</style>
