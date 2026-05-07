<template>
  <el-card class="event-card" shadow="hover" @click="$router.push(`/events/${event.id}`)">
    <div class="card-body">
      <div class="card-main">
        <h3 class="title">{{ event.title }}</h3>
        <p class="desc" v-if="event.description">{{ event.description }}</p>
      </div>
      <div class="card-right">
        <el-tag
          :type="EventStatusColors[event.status] as any"
          size="small"
        >
          {{ EventStatusMap[event.status] || event.status }}
        </el-tag>
      </div>
    </div>
    <div class="card-meta">
      <span class="meta-item organizer" v-if="event.organizer_name">
        <el-icon><Shop /></el-icon>
        {{ event.organizer_name }}
      </span>
      <span class="meta-item">
        <el-icon><Calendar /></el-icon>
        {{ formatDateTime(event.event_time) }}
      </span>
      <span class="meta-item">
        <el-icon><Location /></el-icon>
        {{ event.location }}
      </span>
      <span class="meta-item price">
        {{ formatPrice(event.price) }}
      </span>
    </div>
  </el-card>
</template>

<script setup lang="ts">
import { Calendar, Location, Shop } from '@element-plus/icons-vue'
import type { Event } from '@/api/types'
import { EventStatusMap, EventStatusColors } from '@/api/types'
import { formatDateTime, formatPrice } from '@/utils/format'

defineProps<{ event: Event }>()
</script>

<style scoped>
.event-card {
  cursor: pointer;
  margin-bottom: 12px;
}

.card-body {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  gap: 16px;
}

.card-main {
  flex: 1;
  min-width: 0;
}

.title {
  margin: 0 0 8px;
  font-size: 16px;
  color: #303133;
}

.desc {
  margin: 0;
  color: #909399;
  font-size: 13px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.card-right {
  flex-shrink: 0;
}

.card-meta {
  display: flex;
  gap: 20px;
  margin-top: 12px;
  font-size: 13px;
  color: #909399;
}

.meta-item {
  display: flex;
  align-items: center;
  gap: 4px;
}

.price {
  margin-left: auto;
  font-weight: 600;
  color: #e6a23c;
}

.organizer {
  color: #409eff;
}
</style>
