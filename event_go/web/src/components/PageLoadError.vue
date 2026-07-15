<template>
  <el-result icon="error" title="加载失败" :sub-title="message" class="page-load-error">
    <template #extra>
      <el-button type="primary" :icon="Refresh" @click="$emit('retry')">重新加载</el-button>
    </template>
  </el-result>
</template>

<script setup lang="ts">
import { Refresh } from '@element-plus/icons-vue'

defineProps<{ message: string }>()
defineEmits<{ retry: [] }>()
</script>

<style scoped>
.page-load-error {
  min-height: 320px;
  background: #fff;
}
</style>
