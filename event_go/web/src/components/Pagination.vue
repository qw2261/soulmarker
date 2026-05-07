<template>
  <div class="pagination" v-if="total > 0">
    <el-pagination
      :current-page="currentPage"
      :page-size="pageSize"
      :total="total"
      :page-sizes="[10, 20, 50]"
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
</style>
