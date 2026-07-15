<template>
  <el-alert
    v-if="error"
    :title="error"
    type="error"
    show-icon
    :closable="false"
    class="ticket-error"
  >
    <template #default>
      <el-button text type="primary" @click="fetchTickets">重新加载票种</el-button>
    </template>
  </el-alert>
  <div class="ticket-selector" v-else-if="tickets.length > 0">
    <div class="ticket-label">选择门票（可选）</div>
    <el-radio-group v-model="selectedTicketId" @change="$emit('update:ticketId', selectedTicketId)">
      <el-radio
        v-for="ticket in tickets"
        :key="ticket.id"
        :value="ticket.id"
        class="ticket-item"
      >
        <span class="ticket-name">{{ ticket.name }}</span>
        <span class="ticket-price">{{ formatPrice(ticket.price) }}</span>
        <span class="ticket-stock" :class="{ 'sold-out': ticket.stock <= 0 }">
          剩余 {{ ticket.stock }} 张
        </span>
      </el-radio>
    </el-radio-group>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { listTickets } from '@/api/tickets'
import type { Ticket } from '@/api/types'
import { formatPrice } from '@/utils/format'
import { requestErrorMessage } from '@/utils/request-error'

const props = defineProps<{ eventId: number }>()
defineEmits<{ 'update:ticketId': [id: number | null] }>()

const tickets = ref<Ticket[]>([])
const selectedTicketId = ref<number | null>(null)
const error = ref('')

async function fetchTickets() {
  error.value = ''
  try {
    const res = await listTickets(props.eventId)
    tickets.value = res.data || []
  } catch (cause) {
    tickets.value = []
    error.value = requestErrorMessage(cause, '票种加载失败，请稍后重试')
  }
}

onMounted(fetchTickets)
</script>

<style scoped>
.ticket-selector {
  margin-bottom: 16px;
}

.ticket-error {
  margin-bottom: 16px;
}

.ticket-selector :deep(.el-radio-group) {
  display: flex;
  align-items: stretch;
  flex-direction: column;
}

.ticket-label {
  margin-bottom: 8px;
  font-size: 14px;
  color: #606266;
}

.ticket-item {
  display: flex !important;
  align-items: center;
  gap: 12px;
  margin-bottom: 8px;
  height: auto !important;
  padding: 8px 0;
}

.ticket-name {
  font-weight: 500;
}

.ticket-price {
  color: #e6a23c;
  font-weight: 600;
}

.ticket-stock {
  color: #909399;
  font-size: 12px;
}

.ticket-stock.sold-out {
  color: #f56c6c;
}
</style>
