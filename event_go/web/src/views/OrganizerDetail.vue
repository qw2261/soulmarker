<template>
  <div class="page">
    <NavBar />
    <el-main class="main">
      <div class="container" v-loading="loading">
        <el-button text class="back-link" @click="$router.push('/organizers')">
          ← 返回门店列表
        </el-button>

        <template v-if="organizer">
          <div class="org-header">
            <div class="org-logo">
              <img v-if="organizer.logo_url" :src="organizer.logo_url" :alt="organizer.name" />
              <el-icon v-else size="48"><Shop /></el-icon>
            </div>
            <div class="org-info">
              <h1>{{ organizer.name }}</h1>
              <p class="desc" v-if="organizer.description">{{ organizer.description }}</p>
              <div class="org-meta">
                <span v-if="organizer.address">
                  <el-icon><Location /></el-icon> {{ organizer.address }}
                </span>
                <span v-if="organizer.contact">
                  <el-icon><Phone /></el-icon> {{ organizer.contact }}
                </span>
                <span v-if="organizer.website">
                  <el-icon><Link /></el-icon>
                  <a :href="organizer.website" target="_blank">{{ organizer.website }}</a>
                </span>
              </div>
              <div class="tags" v-if="organizer.tags">
                <el-tag v-for="tag in organizer.tags.split(',')" :key="tag" size="small">
                  {{ tag.trim() }}
                </el-tag>
              </div>
            </div>
          </div>

          <el-divider />

          <h3>旗下活动 ({{ total }})</h3>

          <div v-if="events.length === 0" class="empty">
            暂无活动
          </div>

          <div class="event-grid">
            <EventCard v-for="e in events" :key="e.id" :event="e" />
          </div>

          <Pagination
            :total="total"
            :current-page="page"
            :page-size="pageSize"
            @page-change="onPageChange"
            @size-change="onSizeChange"
          />
        </template>
      </div>
    </el-main>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { Shop, Location, Phone, Link } from '@element-plus/icons-vue'
import { getOrganizer } from '@/api/organizers'
import { listEvents } from '@/api/events'
import type { Organizer, Event } from '@/api/types'
import NavBar from '@/components/NavBar.vue'
import EventCard from '@/components/EventCard.vue'
import Pagination from '@/components/Pagination.vue'

const route = useRoute()
const organizer = ref<Organizer | null>(null)
const events = ref<Event[]>([])
const loading = ref(true)
const total = ref(0)
const page = ref(1)
const pageSize = ref(12)

async function fetchEvents() {
  const id = Number(route.params.id)
  try {
    const res = await listEvents({ organizer_id: id, page: page.value, page_size: pageSize.value })
    events.value = res.data || []
    total.value = res.total || 0
  } finally {
    loading.value = false
  }
}

function onPageChange(p: number) { page.value = p; fetchEvents() }
function onSizeChange(s: number) { pageSize.value = s; page.value = 1; fetchEvents() }

onMounted(async () => {
  const id = Number(route.params.id)
  try {
    const [orgRes] = await Promise.all([
      getOrganizer(id),
      fetchEvents(),
    ])
    organizer.value = orgRes.data || null
  } finally {
    if (loading.value) loading.value = false
  }
})
</script>

<style scoped>
.page { min-height: 100vh; background: #f5f7fa; }
.main { padding-top: 24px; }
.container { max-width: 800px; margin: 0 auto; }
.back-link { margin-bottom: 16px; }

.org-header { display: flex; gap: 24px; align-items: flex-start; background: #fff; padding: 24px; border-radius: 8px; }

.org-logo {
  width: 80px; height: 80px; border-radius: 12px;
  background: #ecf5ff; display: flex; align-items: center; justify-content: center;
  flex-shrink: 0; overflow: hidden; color: #409eff;
}
.org-logo img { width: 100%; height: 100%; object-fit: cover; }

.org-info h1 { margin: 0 0 8px; font-size: 22px; }
.desc { color: #606266; margin: 0 0 12px; line-height: 1.6; }
.org-meta { display: flex; flex-wrap: wrap; gap: 16px; color: #909399; font-size: 13px; margin-bottom: 8px; }
.org-meta span { display: flex; align-items: center; gap: 4px; }
.org-meta a { color: #409eff; text-decoration: none; }
.tags { display: flex; gap: 6px; flex-wrap: wrap; margin-top: 8px; }

h3 { margin: 20px 0 16px; }

.event-grid { display: flex; flex-direction: column; gap: 12px; }
.empty { text-align: center; color: #909399; padding: 32px 0; }
</style>
