<template>
  <el-form
    :model="form"
    ref="formRef"
    label-width="80px"
    @submit.prevent="submit"
  >
    <TicketSelector
      :event-id="eventId"
      @update:ticket-id="form.ticket_id = $event ?? undefined"
    />

    <el-form-item>
      <el-button type="primary" @click="submit" :loading="submitting">
        立即报名
      </el-button>
    </el-form-item>

    <el-dialog v-model="dialogVisible" title="报名成功" width="400px">
      <p>你已经成功报名该活动！</p>
      <p>现在可以参与活动讨论区。</p>
      <template #footer>
        <el-button @click="dialogVisible = false">关闭</el-button>
        <el-button type="primary" @click="goDiscussion">去讨论区</el-button>
      </template>
    </el-dialog>
  </el-form>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { useRouter } from 'vue-router'
import { ElMessage, type FormInstance } from 'element-plus'
import { registerEvent } from '@/api/events'
import TicketSelector from '@/components/TicketSelector.vue'

const props = defineProps<{ eventId: number }>()
const emit = defineEmits<{ registered: [] }>()
const router = useRouter()

const formRef = ref<FormInstance>()
const submitting = ref(false)
const dialogVisible = ref(false)

const form = reactive({
  ticket_id: undefined as number | undefined,
})

async function submit() {
  const valid = await formRef.value?.validate().catch(() => false)
  if (!valid) return

  submitting.value = true
  try {
    const res = await registerEvent(props.eventId, {
      ticket_id: form.ticket_id,
    })
    if (res.code === 201) {
      dialogVisible.value = true
      emit('registered')
    } else {
      ElMessage.error(res.message || '报名失败')
    }
  } catch {
    // error handled by interceptor
  } finally {
    submitting.value = false
  }
}

function goDiscussion() {
  dialogVisible.value = false
  router.push(`/events/${props.eventId}/discussion`)
}
</script>
