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

  </el-form>
</template>

<script setup lang="ts">
import { ref, reactive } from 'vue'
import { ElMessage, type FormInstance } from 'element-plus'
import { registerEvent } from '@/api/events'
import type { Admission } from '@/api/types'
import TicketSelector from '@/components/TicketSelector.vue'

const props = defineProps<{ eventId: number }>()
const emit = defineEmits<{ registered: [admission?: Admission] }>()
const formRef = ref<FormInstance>()
const submitting = ref(false)

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
      ElMessage.success('报名成功，入场凭证已生成')
      emit('registered', res.data?.admission)
    } else {
      ElMessage.error(res.message || '报名失败')
    }
  } catch {
    // error handled by interceptor
  } finally {
    submitting.value = false
  }
}
</script>
