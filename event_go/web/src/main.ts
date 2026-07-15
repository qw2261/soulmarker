import { createApp } from 'vue'
import { createPinia } from 'pinia'
import {
  ElAlert,
  ElButton,
  ElCard,
  ElDescriptions,
  ElDescriptionsItem,
  ElDivider,
  ElDrawer,
  ElEmpty,
  ElForm,
  ElFormItem,
  ElHeader,
  ElIcon,
  ElInput,
  ElInputNumber,
  ElMain,
  ElOption,
  ElPagination,
  ElRadio,
  ElRadioGroup,
  ElResult,
  ElSelect,
  ElSkeleton,
  ElSkeletonItem,
  ElTabPane,
  ElTable,
  ElTableColumn,
  ElTabs,
  ElTag,
} from 'element-plus'
import 'element-plus/dist/index.css'
import App from './App.vue'
import router from './router'
import { useUserStore } from './stores/user'
import { USER_SESSION_EXPIRED_EVENT } from './auth/session'

const app = createApp(App)

app.use(createPinia())
app.use(router)

const elementComponents = [
  ElAlert,
  ElButton,
  ElCard,
  ElDescriptions,
  ElDescriptionsItem,
  ElDivider,
  ElDrawer,
  ElEmpty,
  ElForm,
  ElFormItem,
  ElHeader,
  ElIcon,
  ElInput,
  ElInputNumber,
  ElMain,
  ElOption,
  ElPagination,
  ElRadio,
  ElRadioGroup,
  ElResult,
  ElSelect,
  ElSkeleton,
  ElSkeletonItem,
  ElTabPane,
  ElTable,
  ElTableColumn,
  ElTabs,
  ElTag,
]

for (const component of elementComponents) {
  app.use(component)
}

window.addEventListener(USER_SESSION_EXPIRED_EVENT, (event) => {
  const redirect = (event as CustomEvent<{ redirect?: string }>).detail?.redirect || '/'
  useUserStore().logout()
  if (router.currentRoute.value.path !== '/login') {
    router.push({ path: '/login', query: { reason: 'expired', redirect } })
  }
})

app.mount('#app')
