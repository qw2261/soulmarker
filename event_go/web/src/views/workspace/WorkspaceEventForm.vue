<template>
  <div class="page"><NavBar /><el-main class="main"><div class="container"><WorkspaceToolbar />
    <div class="header"><el-button text :icon="ArrowLeft" @click="$router.push(base)">返回</el-button><h1>{{ isEdit ? '编辑活动' : '创建活动' }}</h1></div>
    <el-card><el-form ref="formRef" :model="form" :rules="rules" label-width="100px">
      <el-form-item label="标题" prop="title"><el-input v-model="form.title" placeholder="活动标题" /></el-form-item>
      <el-form-item label="描述"><el-input v-model="form.description" type="textarea" :rows="3" placeholder="活动描述" /></el-form-item>
      <el-form-item label="活动封面"><el-input v-model="form.cover_url" placeholder="https://example.com/event-cover.jpg" /></el-form-item>
      <el-form-item label="时间" prop="event_time"><el-input v-model="form.event_time" placeholder="2026-12-31T18:00:00+08:00" /></el-form-item>
      <el-form-item label="地点" prop="location"><el-input v-model="form.location" placeholder="活动地点" /></el-form-item>
      <el-form-item label="容量"><el-input-number v-model="form.capacity" :min="1" /></el-form-item>
      <el-form-item label="价格"><el-input-number v-model="form.price" :min="0" :precision="2" :step="0.01" /></el-form-item>
      <el-form-item v-if="isEdit" label="状态"><el-select v-model="form.status"><el-option label="草稿" value="draft" /><el-option label="已发布" value="published" /><el-option label="已取消" value="cancelled" /><el-option label="已结束" value="ended" /></el-select></el-form-item>
      <el-form-item><el-button type="primary" :loading="submitting" @click="submit">{{ isEdit ? '保存修改' : '创建活动' }}</el-button><el-button @click="$router.push(base)">取消</el-button></el-form-item>
    </el-form></el-card>
  </div></el-main></div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ArrowLeft } from '@element-plus/icons-vue'
import { ElMessage, type FormInstance } from 'element-plus'
import { useRoute, useRouter } from 'vue-router'
import NavBar from '@/components/NavBar.vue'
import WorkspaceToolbar from '@/components/WorkspaceToolbar.vue'
import { createOrganizationEvent, getOrganizationEvent, getOrganizationWorkspace, updateOrganizationEvent } from '@/api/organizations'

const route = useRoute()
const router = useRouter()
const organizationId = Number(route.params.organizationId)
const eventId = Number(route.params.id)
const isEdit = Number.isInteger(eventId) && eventId > 0
const base = `/workspace/${organizationId}`
const organizerId = ref<number>()
const formRef = ref<FormInstance>()
const submitting = ref(false)
const form = reactive({ title: '', description: '', cover_url: '', event_time: '', location: '', capacity: 50, price: 0, status: 'draft' })
const rules = { title: [{ required: true, message: '请输入标题', trigger: 'blur' }], event_time: [{ required: true, message: '请输入时间', trigger: 'blur' }], location: [{ required: true, message: '请输入地点', trigger: 'blur' }] }
onMounted(async () => {
  const workspaceResponse = await getOrganizationWorkspace(organizationId)
  organizerId.value = workspaceResponse.data?.profile.id
  if (isEdit) { const response = await getOrganizationEvent(organizationId, eventId); if (response.data) Object.assign(form, response.data) }
})
async function submit() {
  if (!await formRef.value?.validate().catch(() => false) || !organizerId.value) return
  submitting.value = true
  try {
    const data = { organizer_id: organizerId.value, title: form.title, description: form.description, cover_url: form.cover_url, event_time: form.event_time, location: form.location, capacity: form.capacity, price: form.price }
    if (isEdit) { await updateOrganizationEvent(organizationId, eventId, { ...data, status: form.status }); ElMessage.success('活动已更新') }
    else { await createOrganizationEvent(organizationId, data); ElMessage.success('活动已创建') }
    await router.push(base)
  } finally { submitting.value = false }
}
</script>

<style scoped>.page { min-height: 100vh; background: #f5f7fa; }.main { padding-top: 12px; }.container { max-width: 760px; margin: 0 auto; }.header { display: flex; align-items: center; gap: 16px; margin-bottom: 18px; }.header h1 { margin: 0; font-size: 24px; }@media (max-width: 600px) { .main { padding: 8px; } }</style>
