<template>
  <div class="page">
    <NavBar />
    <el-main class="main">
      <div class="container">
        <h2>门店列表</h2>

        <el-empty v-if="!loading && organizers.length === 0" description="暂无门店" />

        <div v-loading="loading" class="org-grid">
          <el-card
            v-for="org in organizers"
            :key="org.id"
            class="org-card"
            shadow="hover"
            @click="$router.push(`/organizers/${org.id}`)"
          >
            <div class="org-logo" v-if="org.logo_url">
              <img :src="org.logo_url" :alt="org.name" />
            </div>
            <div class="org-logo placeholder" v-else>
              <el-icon size="32"><Shop /></el-icon>
            </div>
            <h3>{{ org.name }}</h3>
            <p class="desc" v-if="org.description">{{ org.description }}</p>
            <div class="tags" v-if="org.tags">
              <el-tag v-for="tag in org.tags.split(',')" :key="tag" size="small" class="tag">
                {{ tag.trim() }}
              </el-tag>
            </div>
            <div class="org-meta">
              <span>{{ org.event_count || 0 }} 个活动</span>
              <span v-if="org.address">{{ org.address }}</span>
            </div>
          </el-card>
        </div>

        <Pagination
          :total="total"
          :current-page="page"
          :page-size="pageSize"
          @page-change="onPageChange"
          @size-change="onSizeChange"
        />
      </div>
    </el-main>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { Shop } from '@element-plus/icons-vue'
import { listOrganizers } from '@/api/organizers'
import type { Organizer } from '@/api/types'
import NavBar from '@/components/NavBar.vue'
import Pagination from '@/components/Pagination.vue'

const organizers = ref<Organizer[]>([])
const loading = ref(false)
const total = ref(0)
const page = ref(1)
const pageSize = ref(12)

async function fetchData() {
  loading.value = true
  try {
    const res = await listOrganizers({ page: page.value, page_size: pageSize.value })
    organizers.value = res.data || []
    total.value = res.total || 0
  } finally {
    loading.value = false
  }
}

function onPageChange(p: number) { page.value = p; fetchData() }
function onSizeChange(s: number) { pageSize.value = s; page.value = 1; fetchData() }

onMounted(fetchData)
</script>

<style scoped>
.page { min-height: 100vh; background: #f5f7fa; }
.main { padding-top: 24px; }
.container { max-width: 960px; margin: 0 auto; }
.container h2 { margin: 0 0 20px; }

.org-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 16px;
}

.org-card {
  cursor: pointer;
  text-align: center;
  padding: 12px 0;
}

.org-logo {
  width: 64px;
  height: 64px;
  border-radius: 8px;
  background: #ecf5ff;
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 12px;
  overflow: hidden;
}

.org-logo img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.org-logo.placeholder {
  color: #409eff;
}

.org-card h3 { margin: 0 0 8px; font-size: 16px; }

.desc { color: #909399; font-size: 13px; margin: 0 0 8px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

.tags { margin-bottom: 8px; }
.tag { margin: 2px; }

.org-meta {
  display: flex;
  justify-content: center;
  gap: 12px;
  font-size: 12px;
  color: #909399;
}
</style>
