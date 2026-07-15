<template>
  <el-card
    class="event-card"
    shadow="hover"
    role="link"
    tabindex="0"
    @click="openEvent"
    @keyup.enter="openEvent"
    @keyup.space.prevent="openEvent"
  >
    <div class="card-layout">
      <div v-if="event.cover_url && !coverFailed" class="cover-wrap">
        <img
          :src="event.cover_url"
          :alt="`${event.title}活动封面`"
          class="cover"
          loading="lazy"
          decoding="async"
          referrerpolicy="no-referrer"
          @error="coverFailed = true"
        />
      </div>
      <div class="card-content">
        <div class="card-body">
          <div class="card-main">
            <h3 class="title">{{ event.title }}</h3>
            <p class="desc" v-if="event.description">{{ event.description }}</p>
          </div>
          <div class="card-right">
            <el-tag :type="EventStatusColors[event.status] as any" size="small">
              {{ EventStatusMap[event.status] || event.status }}
            </el-tag>
          </div>
        </div>
        <div class="card-meta">
          <span class="meta-item organizer" v-if="event.organizer_name">
            <el-icon><Shop /></el-icon>
            {{ event.organizer_name }}
          </span>
          <span class="meta-item event-date">
            <el-icon><Calendar /></el-icon>
            {{ formatDateTime(event.event_time) }}
          </span>
          <span class="meta-item event-location">
            <el-icon><Location /></el-icon>
            {{ event.location }}
          </span>
          <span class="meta-item price">
            {{ formatPrice(event.price) }}
          </span>
        </div>
      </div>
    </div>
  </el-card>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useRouter } from 'vue-router'
import { Calendar, Location, Shop } from '@element-plus/icons-vue'
import type { Event } from '@/api/types'
import { EventStatusMap, EventStatusColors } from '@/api/types'
import { formatDateTime, formatPrice } from '@/utils/format'

const props = defineProps<{ event: Event }>()
const router = useRouter()
const coverFailed = ref(false)

function openEvent() {
  router.push(`/events/${props.event.id}`)
}
</script>

<style scoped>
.event-card {
  cursor: pointer;
  margin-bottom: 12px;
}

.event-card:focus-visible {
  outline: 3px solid #79bbff;
  outline-offset: 2px;
}

.card-layout {
  display: grid;
  grid-template-columns: 168px minmax(0, 1fr);
  gap: 18px;
}

.card-content {
  min-width: 0;
}

.cover-wrap {
  width: 168px;
  aspect-ratio: 16 / 10;
  overflow: hidden;
  border-radius: 6px;
  background: #e4e7ed;
}

.cover {
  width: 100%;
  height: 100%;
  object-fit: cover;
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

@media (max-width: 640px) {
  .card-layout {
    grid-template-columns: 1fr;
    gap: 12px;
  }

  .cover-wrap {
    width: 100%;
  }

  .card-meta {
    flex-wrap: wrap;
    gap: 8px 14px;
  }

  .meta-item.organizer,
  .meta-item.event-date {
    width: 100%;
  }

  .price {
    margin-left: 0;
  }
}
</style>
