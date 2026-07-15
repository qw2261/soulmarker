<template>
  <div class="page"><NavBar /><el-main class="main"><div class="container"><WorkspaceToolbar />
    <div class="header"><div><h1>成员与邀请</h1><p>角色权限由服务端实时计算。</p></div></div>
    <section v-if="workspace.can('members.invite')" class="invite-section">
      <h2>邀请成员</h2>
      <el-form :model="invite" inline><el-form-item label="邮箱"><el-input v-model="invite.email" placeholder="member@example.com" /></el-form-item><el-form-item label="角色"><el-select v-model="invite.role"><el-option v-for="option in availableRoles" :key="option.value" :label="option.label" :value="option.value" /></el-select></el-form-item><el-form-item><el-button type="primary" :loading="inviting" @click="sendInvite">发送邀请</el-button></el-form-item></el-form>
    </section>
    <el-tabs v-model="tab">
      <el-tab-pane label="成员" name="members"><div class="table-scroll"><el-table :data="members" v-loading="loading" stripe class="data-table">
        <el-table-column prop="name" label="姓名" min-width="130" /><el-table-column prop="contact" label="联系方式" min-width="210" />
        <el-table-column label="角色" width="150"><template #default="{ row }"><el-select v-if="canManage(row)" :model-value="row.role" size="small" @change="changeRole(row, $event)"><el-option v-for="option in availableRoles" :key="option.value" :label="option.label" :value="option.value" /></el-select><span v-else>{{ roleLabel(row.role) }}</span></template></el-table-column>
        <el-table-column label="状态" width="90"><template #default="{ row }"><el-tag :type="row.status === 'active' ? 'success' : 'info'" size="small">{{ row.status === 'active' ? '有效' : '已撤销' }}</el-tag></template></el-table-column>
        <el-table-column v-if="workspace.can('members.manage')" label="操作" width="100"><template #default="{ row }"><el-button v-if="canManage(row) && row.status === 'active'" size="small" type="danger" @click="revoke(row)">撤销</el-button></template></el-table-column>
      </el-table></div></el-tab-pane>
      <el-tab-pane label="邀请" name="invitations"><div class="table-scroll"><el-table :data="invitations" v-loading="loading" stripe class="data-table"><el-table-column prop="email" label="邮箱" min-width="220" /><el-table-column label="角色" width="100"><template #default="{ row }">{{ roleLabel(row.role) }}</template></el-table-column><el-table-column label="状态" width="100"><template #default="{ row }">{{ invitationStatus[row.status] || row.status }}</template></el-table-column><el-table-column label="有效期" min-width="180"><template #default="{ row }">{{ formatDateTime(row.expires_at) }}</template></el-table-column><el-table-column v-if="workspace.can('members.invite')" label="操作" width="100"><template #default="{ row }"><el-button v-if="row.status === 'pending'" size="small" type="danger" @click="revokeInvite(row)">撤销</el-button></template></el-table-column></el-table></div></el-tab-pane>
    </el-tabs>
  </div></el-main></div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useRoute } from 'vue-router'
import NavBar from '@/components/NavBar.vue'
import WorkspaceToolbar from '@/components/WorkspaceToolbar.vue'
import { createOrganizationInvitation, listOrganizationInvitations, listOrganizationMembers, revokeOrganizationInvitation, revokeOrganizationMember, updateOrganizationMember } from '@/api/organizations'
import type { OrganizationInvitation, OrganizationMember, OrganizationRole } from '@/api/types'
import { useWorkspaceStore } from '@/stores/workspace'
import { confirmAction } from '@/utils/confirm'
import { formatDateTime } from '@/utils/format'

type AssignableRole = Exclude<OrganizationRole, 'owner'>
const organizationId = Number(useRoute().params.organizationId)
const workspace = useWorkspaceStore()
const members = ref<OrganizationMember[]>([])
const invitations = ref<OrganizationInvitation[]>([])
const loading = ref(false)
const inviting = ref(false)
const tab = ref('members')
const invite = reactive<{ email: string; role: AssignableRole }>({ email: '', role: 'editor' })
const allRoles: { value: AssignableRole; label: string }[] = [{ value: 'admin', label: '管理员' }, { value: 'editor', label: '编辑' }, { value: 'checker', label: '核销' }, { value: 'finance', label: '财务' }]
const availableRoles = allRoles.filter((role) => workspace.current?.role === 'owner' || role.value !== 'admin')
const labels: Record<string, string> = { owner: '所有者', admin: '管理员', editor: '编辑', checker: '核销', finance: '财务' }
const invitationStatus: Record<string, string> = { pending: '待接受', accepted: '已接受', revoked: '已撤销', expired: '已过期' }
function roleLabel(role: string) { return labels[role] || role }
function canManage(member: OrganizationMember) { return workspace.can('members.manage') && member.role !== 'owner' && (workspace.current?.role === 'owner' || member.role !== 'admin') }
async function load() {
  loading.value = true
  try {
    const [memberResponse, invitationResponse] = await Promise.all([listOrganizationMembers(organizationId), listOrganizationInvitations(organizationId)])
    members.value = memberResponse.data || []
    invitations.value = invitationResponse.data || []
  } finally { loading.value = false }
}
async function sendInvite() {
  if (!invite.email.trim()) { ElMessage.warning('请输入邀请邮箱'); return }
  inviting.value = true
  try { await createOrganizationInvitation(organizationId, invite.email.trim(), invite.role); invite.email = ''; ElMessage.success('邀请已发送'); await load() } finally { inviting.value = false }
}
async function changeRole(member: OrganizationMember, role: AssignableRole) { await updateOrganizationMember(organizationId, member.id, role); ElMessage.success('角色已更新'); await workspace.loadSession(organizationId); await load() }
async function revoke(member: OrganizationMember) { try { await confirmAction(`确定撤销成员「${member.name}」？`, '撤销成员', { type: 'warning' }); await revokeOrganizationMember(organizationId, member.id); ElMessage.success('成员已撤销'); await load() } catch { /* cancelled */ } }
async function revokeInvite(invitation: OrganizationInvitation) { try { await confirmAction(`确定撤销发给 ${invitation.email} 的邀请？`, '撤销邀请', { type: 'warning' }); await revokeOrganizationInvitation(organizationId, invitation.id); ElMessage.success('邀请已撤销'); await load() } catch { /* cancelled */ } }
onMounted(load)
</script>

<style scoped>
.page { min-height: 100vh; background: #f5f7fa; }.main { padding-top: 12px; }.container { max-width: 1060px; margin: 0 auto; }.header h1,.header p { margin: 0; }.header h1 { font-size: 24px; }.header p { margin-top: 5px; color: #606266; }.invite-section { margin: 22px 0; padding: 18px 0; border-top: 1px solid #dcdfe6; border-bottom: 1px solid #dcdfe6; }.invite-section h2 { margin: 0 0 14px; font-size: 18px; }.table-scroll { max-width: 100%; overflow-x: auto; }.data-table { min-width: 760px; }
@media (max-width: 600px) { .main { padding: 8px; }.invite-section :deep(.el-form--inline .el-form-item) { display: flex; margin-right: 0; } }
</style>
