<template>
  <el-header class="navbar">
    <div class="navbar-left">
      <router-link to="/" class="logo">亦闻</router-link>
      <router-link to="/organizers" class="nav-link">门店</router-link>
    </div>
    <div class="navbar-right">
      <template v-if="user.isLoggedIn">
        <router-link to="/me/registrations">我的活动</router-link>
        <span class="user-name">{{ user.user?.name }}</span>
        <el-button text @click="logoutUser">退出</el-button>
      </template>
      <template v-else>
        <router-link to="/login">登录</router-link>
        <router-link to="/register">注册</router-link>
      </template>
      <span class="divider">|</span>
      <template v-if="auth.isAdmin">
        <router-link to="/admin/events">运营</router-link>
        <el-button text @click="logoutAdmin">退出管理</el-button>
      </template>
      <router-link to="/admin" v-else>管理登录</router-link>
    </div>
    <el-button
      class="mobile-menu-button"
      text
      :icon="Menu"
      aria-label="打开导航菜单"
      title="导航菜单"
      @click="mobileMenuOpen = true"
    />
    <el-drawer v-model="mobileMenuOpen" title="导航" direction="rtl" size="min(82vw, 320px)">
      <nav class="mobile-nav">
        <router-link to="/" @click="mobileMenuOpen = false">活动</router-link>
        <router-link to="/organizers" @click="mobileMenuOpen = false">门店</router-link>
        <template v-if="user.isLoggedIn">
          <router-link to="/me/registrations" @click="mobileMenuOpen = false">我的活动</router-link>
          <span class="mobile-user">{{ user.user?.name }}</span>
          <el-button text @click="logoutUser">退出登录</el-button>
        </template>
        <template v-else>
          <router-link to="/login" @click="mobileMenuOpen = false">登录</router-link>
          <router-link to="/register" @click="mobileMenuOpen = false">注册</router-link>
        </template>
        <template v-if="auth.isAdmin">
          <router-link to="/admin/events" @click="mobileMenuOpen = false">运营后台</router-link>
          <el-button text @click="logoutAdmin">退出管理</el-button>
        </template>
        <router-link v-else to="/admin" @click="mobileMenuOpen = false">管理登录</router-link>
      </nav>
    </el-drawer>
  </el-header>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { Menu } from '@element-plus/icons-vue'
import { useAuthStore } from '@/stores/auth'
import { useUserStore } from '@/stores/user'
import { logoutUser as revokeUserSession } from '@/api/auth'
import { useRouter } from 'vue-router'
const auth = useAuthStore()
const user = useUserStore()
const router = useRouter()
const mobileMenuOpen = ref(false)

async function logoutUser() {
  try {
    await revokeUserSession()
  } finally {
    user.logout()
    mobileMenuOpen.value = false
    router.push('/')
  }
}

function logoutAdmin() {
  auth.logout()
  mobileMenuOpen.value = false
  router.push('/')
}
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

.mobile-menu-button {
  display: none;
  width: 40px;
  height: 40px;
}

.mobile-nav {
  display: grid;
  gap: 4px;
}

.mobile-nav a,
.mobile-nav button,
.mobile-user {
  display: flex;
  min-height: 44px;
  align-items: center;
  padding: 0 8px;
  color: #303133;
  text-decoration: none;
}

.mobile-user {
  color: #909399;
  font-size: 14px;
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

@media (max-width: 720px) {
  .navbar {
    height: 56px;
    padding: 0 16px;
  }

  .navbar-right,
  .nav-link {
    display: none;
  }

  .mobile-menu-button {
    display: inline-flex;
  }
}
</style>
