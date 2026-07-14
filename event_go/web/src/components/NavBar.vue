<template>
  <el-header class="navbar">
    <div class="navbar-left">
      <router-link to="/" class="logo">亦闻</router-link>
      <router-link to="/organizers" class="nav-link">门店</router-link>
    </div>
    <div class="navbar-right">
      <template v-if="user.isLoggedIn">
        <router-link to="/me/registrations">我的报名</router-link>
        <span class="user-name">{{ user.user?.name }}</span>
        <el-button text @click="user.logout()">退出</el-button>
      </template>
      <template v-else>
        <router-link to="/login">登录</router-link>
        <router-link to="/register">注册</router-link>
      </template>
      <span class="divider">|</span>
      <router-link to="/admin/events" v-if="auth.isAdmin">管理</router-link>
      <router-link to="/admin" v-else>管理登录</router-link>
    </div>
  </el-header>
</template>

<script setup lang="ts">
import { useAuthStore } from '@/stores/auth'
import { useUserStore } from '@/stores/user'
const auth = useAuthStore()
const user = useUserStore()
</script>

<style scoped>
.navbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  background: #fff;
  border-bottom: 1px solid #ebeef5;
  padding: 0 24px;
  height: 56px;
}

.logo {
  font-size: 20px;
  font-weight: 700;
  color: #409eff;
  text-decoration: none;
}

.nav-link {
  color: #606266;
  text-decoration: none;
  font-size: 14px;
  margin-left: 20px;
}

.nav-link:hover {
  color: #409eff;
}

.navbar-right {
  display: flex;
  align-items: center;
  gap: 12px;
}

.navbar-right a {
  color: #606266;
  text-decoration: none;
  font-size: 14px;
}

.navbar-right a:hover {
  color: #409eff;
}

.user-name {
  color: #303133;
  font-weight: 500;
  font-size: 14px;
}

.divider {
  color: #dcdfe6;
}
</style>
