<template>
  <div class="page">
    <NavBar />
    <el-main class="main">
      <div class="container">
        <div class="header">
          <div><h1>组织工作台</h1><p>选择组织，或创建新的运营空间。</p></div>
          <el-button type="primary" :icon="Plus" @click="$router.push('/workspace/new')">创建组织</el-button>
        </div>
        <el-table :data="workspace.organizations" v-loading="workspace.loading" stripe>
          <el-table-column prop="organization_name" label="组织" min-width="180" />
          <el-table-column prop="organization_slug" label="标识" min-width="140" />
          <el-table-column label="角色" width="100">
            <template #default="{ row }">{{ roleLabels[row.role] || row.role }}</template>
          </el-table-column>
          <el-table-column label="操作" width="120">
            <template #default="{ row }">
              <el-button size="small" @click="$router.push(`/workspace/${row.organization_id}`)">进入</el-button>
            </template>
          </el-table-column>
        </el-table>
        <el-empty v-if="!workspace.loading && !workspace.organizations.length" description="暂无组织" />
      </div>
    </el-main>
  </div>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import { Plus } from '@element-plus/icons-vue'
import NavBar from '@/components/NavBar.vue'
import { useWorkspaceStore } from '@/stores/workspace'

const workspace = useWorkspaceStore()
const roleLabels: Record<string, string> = { owner: '所有者', admin: '管理员', editor: '编辑', checker: '核销', finance: '财务' }
onMounted(() => workspace.loadOrganizations())
</script>

<style scoped>
.page { min-height: 100vh; background: #f5f7fa; }
.main { padding-top: 24px; }
.container { max-width: 960px; margin: 0 auto; }
.header { display: flex; align-items: center; justify-content: space-between; gap: 16px; margin-bottom: 18px; }
.header h1, .header p { margin: 0; }
.header h1 { font-size: 24px; }
.header p { margin-top: 6px; color: #606266; font-size: 14px; }
@media (max-width: 600px) { .main { padding: 16px 8px; } .header { align-items: stretch; flex-direction: column; } }
</style>
