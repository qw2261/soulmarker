import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { getOrganizationSession, listMyOrganizations } from '@/api/organizations'
import type { OrganizationContext } from '@/api/types'

export const useWorkspaceStore = defineStore('workspace', () => {
  const organizations = ref<OrganizationContext[]>([])
  const current = ref<OrganizationContext>()
  const loading = ref(false)

  const capabilities = computed(() => new Set(current.value?.capabilities || []))

  function can(capability: string) {
    return capabilities.value.has(capability)
  }

  async function loadOrganizations() {
    loading.value = true
    try {
      const response = await listMyOrganizations()
      organizations.value = response.data || []
      return organizations.value
    } finally {
      loading.value = false
    }
  }

  async function loadSession(organizationId: number) {
    const response = await getOrganizationSession(organizationId)
    current.value = response.data
    return current.value
  }

  function clear() {
    organizations.value = []
    current.value = undefined
  }

  return { organizations, current, loading, can, loadOrganizations, loadSession, clear }
})
