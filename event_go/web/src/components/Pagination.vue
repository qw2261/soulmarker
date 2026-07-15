<template>
  <div class="pagination" v-if="total > 0">
    <el-pagination
      :current-page="currentPage"
      :page-size="pageSize"
      :total="total"
      :page-sizes="[10, 20, 50]"
      :pager-count="5"
      layout="total, sizes, prev, pager, next"
      @current-change="$emit('page-change', $event)"
      @size-change="$emit('size-change', $event)"
    />
  </div>
</template>

<script setup lang="ts">
defineProps<{
  total: number
  currentPage: number
  pageSize: number
}>()

defineEmits<{
  'page-change': [page: number]
  'size-change': [size: number]
}>()
</script>

<style scoped>
.pagination {
  display: flex;
  justify-content: center;
  margin-top: 24px;
}

@media (max-width: 600px) {
  .pagination {
    overflow-x: auto;
    justify-content: flex-start;
    padding-bottom: 4px;
  }

  .pagination :deep(.el-pagination__total),
  .pagination :deep(.el-pagination__sizes) {
    display: none;
  }
}
</style>
