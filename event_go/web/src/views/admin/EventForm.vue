<template>
  <div class="page">
    <NavBar />
    <el-main class="main">
      <div class="container">
        <AdminToolbar />
        <div class="header">
          <el-button text @click="$router.push('/admin/events')">← 返回</el-button>
          <h2>{{ isEdit ? '编辑活动' : '创建活动' }}</h2>
        </div>

        <el-card>
          <el-form
            ref="formRef"
            :model="form"
            :rules="rules"
            label-width="100px"
          >
            <el-form-item label="门店" prop="organizer_id">
              <el-select
                v-model="form.organizer_id"
                placeholder="选择门店"
                filterable
              >
                <el-option
                  v-for="org in organizers"
                  :key="org.id"
                  :label="org.name"
                  :value="org.id"
                />
              </el-select>
            </el-form-item>
            <el-form-item label="标题" prop="title">
              <el-input v-model="form.title" placeholder="活动标题" />
            </el-form-item>
            <el-form-item label="描述">
              <el-input
                v-model="form.description"
                type="textarea"
                :rows="3"
                placeholder="活动描述"
              />
            </el-form-item>
            <el-form-item label="时间" prop="event_time">
              <el-input v-model="form.event_time" placeholder="2026-12-31T18:00:00+08:00" />
            </el-form-item>
            <el-form-item label="地点" prop="location">
              <el-input v-model="form.location" placeholder="活动地点" />
            </el-form-item>
            <el-form-item label="容量">
              <el-input-number v-model="form.capacity" :min="1" />
            </el-form-item>
            <el-form-item label="价格">
              <el-input-number v-model="form.price" :min="0" :precision="2" :step="0.01" />
            </el-form-item>
            <el-form-item label="状态" v-if="isEdit">
              <el-select v-model="form.status">
                <el-option label="草稿" value="draft" />
                <el-option label="已发布" value="published" />
                <el-option label="已取消" value="cancelled" />
                <el-option label="已结束" value="ended" />
              </el-select>
            </el-form-item>
            <el-form-item>
              <el-button type="primary" @click="submit" :loading="submitting">
                {{ isEdit ? '保存修改' : '创建活动' }}
              </el-button>
              <el-button @click="$router.push('/admin/events')">取消</el-button>
            </el-form-item>
          </el-form>
        </el-card>
      </div>
    </el-main>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, type FormInstance } from 'element-plus'
import { createEvent, getEvent, updateEvent } from '@/api/events'
import { listOrganizers } from '@/api/organizers'
import type { Organizer } from '@/api/types'
import AdminToolbar from '@/components/AdminToolbar.vue'

const route = useRoute()
const router = useRouter()
const isEdit = !!route.params.id

const formRef = ref<FormInstance>()
const submitting = ref(false)
const organizers = ref<Organizer[]>([])

const form = reactive({
  organizer_id: undefined as number | undefined,
  title: '',
  description: '',
  event_time: '',
  location: '',
  capacity: 50,
  price: 0,
  status: 'draft',
})

const rules = {
  title: [{ required: true, message: '请输入标题', trigger: 'blur' }],
  event_time: [{ required: true, message: '请输入时间', trigger: 'blur' }],
  location: [{ required: true, message: '请输入地点', trigger: 'blur' }],
  organizer_id: [{ required: true, message: '请选择门店', trigger: 'change', type: 'number', min: 1 }],
}

onMounted(async () => {
  const orgRes = await listOrganizers({ page_size: 200 })
  organizers.value = orgRes.data || []

  if (isEdit) {
    const id = Number(route.params.id)
    const res = await getEvent(id)
    if (res.data) {
      Object.assign(form, res.data)
    }
  }
})

async function submit() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return
  submitting.value = true
  try {
    if (isEdit) {
      const id = Number(route.params.id)
      await updateEvent(id, {
        organizer_id: form.organizer_id!,
        title: form.title,
        description: form.description,
        event_time: form.event_time,
        location: form.location,
        capacity: form.capacity,
        price: form.price,
        status: form.status,
      })
      ElMessage.success('活动已更新')
    } else {
      await createEvent({
        organizer_id: form.organizer_id!,
        title: form.title,
        description: form.description,
        event_time: form.event_time,
        location: form.location,
        capacity: form.capacity,
        price: form.price,
      })
      ElMessage.success('活动已创建')
    }
    router.push('/admin/events')
  } finally {
    submitting.value = false
  }
}
</script>

<style scoped>
.page { min-height: 100vh; background: #f5f7fa; }
.main { padding-top: 24px; }
.container { max-width: 700px; margin: 0 auto; }

.header {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 16px;
}

.header h2 { margin: 0; }

@media (max-width: 600px) {
  .main { padding: 16px 8px; }
}
</style>
