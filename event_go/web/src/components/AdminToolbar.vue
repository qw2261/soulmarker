<template>
  <nav class="admin-toolbar" aria-label="运营后台导航">
    <router-link to="/admin/events">活动</router-link>
    <router-link to="/admin/organizers">门店</router-link>
  </nav>
</template>

<style scoped>
.admin-toolbar {
  display: flex;
  gap: 24px;
  margin-bottom: 22px;
  border-bottom: 1px solid #dcdfe6;
}

.admin-toolbar a {
  position: relative;
  padding: 0 2px 12px;
  color: #606266;
  text-decoration: none;
}

.admin-toolbar a.router-link-active {
  color: #303133;
  font-weight: 600;
}

.admin-toolbar a.router-link-active::after {
  position: absolute;
  right: 0;
  bottom: -1px;
  left: 0;
  height: 2px;
  background: #409eff;
  content: '';
}
</style>
