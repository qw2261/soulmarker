<template>
  <div class="page">
    <NavBar />
    <el-main class="main">
      <div class="container">
        <AdminToolbar />
        <div class="header">
          <div>
            <h2>内容治理</h2>
            <p>处理参与者举报，保留软删除内容和不可丢失的操作记录。</p>
          </div>
        </div>

        <el-tabs v-model="activeTab" @tab-change="onTabChange">
          <el-tab-pane label="举报队列" name="reports">
            <div class="filters">
              <el-select v-model="statusFilter" aria-label="举报状态" @change="refreshReports">
                <el-option label="待处理" value="open" />
                <el-option label="已移除处理" value="resolved" />
                <el-option label="已驳回" value="dismissed" />
              </el-select>
              <el-select v-model="targetTypeFilter" aria-label="内容类型" clearable placeholder="全部内容" @change="refreshReports">
                <el-option label="帖子" value="post" />
                <el-option label="回复" value="reply" />
              </el-select>
            </div>

            <el-skeleton v-if="loading" :rows="6" animated />
            <PageLoadError v-else-if="error" :message="error" @retry="fetchReports" />
            <el-empty v-else-if="reports.length === 0" description="当前筛选条件下没有举报" />
            <div v-else class="report-list">
              <article v-for="report in reports" :key="report.id" class="report-card" :data-report-id="report.id">
                <div class="report-topline">
                  <div class="tags">
                    <el-tag :type="statusType(report.status)">{{ statusLabel(report.status) }}</el-tag>
                    <el-tag type="info">{{ targetTypeLabel(report.target_type) }}</el-tag>
                    <el-tag type="warning">{{ categoryLabel(report.category) }}</el-tag>
                    <el-tag v-if="report.target_moderation_status === 'removed'" type="danger">内容已移除</el-tag>
                  </div>
                  <span>#{{ report.id }} · {{ formatDateTime(report.created_at) }}</span>
                </div>
                <h3>{{ report.target_title }}</h3>
                <blockquote>{{ report.target_content }}</blockquote>
                <div class="report-meta">
                  <span>内容作者：{{ report.target_author_name }}</span>
                  <span>举报人：{{ report.reporter_name }}</span>
                </div>
                <p class="report-detail">举报说明：{{ report.detail || '未填写' }}</p>
                <p v-if="report.status !== 'open'" class="resolution">
                  处理：{{ report.resolved_by }} · {{ report.resolution_note }}
                </p>
                <div class="report-actions">
                  <template v-if="report.status === 'open'">
                    <el-button type="danger" @click="openAction(report, 'resolve-remove')">移除并处理</el-button>
                    <el-button @click="openAction(report, 'resolve-dismiss')">驳回举报</el-button>
                  </template>
                  <template v-else>
                    <el-button
                      v-if="report.target_moderation_status === 'removed'"
                      type="primary"
                      plain
                      @click="openAction(report, 'restore')"
                    >恢复内容</el-button>
                    <el-button v-else type="danger" plain @click="openAction(report, 'remove')">直接移除内容</el-button>
                  </template>
                  <router-link
                    v-if="report.target_moderation_status === 'visible'"
                    :to="`/events/${report.event_id}/posts/${report.post_id}`"
                  >查看公开内容</router-link>
                </div>
              </article>
            </div>
            <Pagination
              :total="reportsTotal"
              :current-page="reportsPage"
              :page-size="pageSize"
              @page-change="onReportsPageChange"
              @size-change="onReportsSizeChange"
            />
          </el-tab-pane>

          <el-tab-pane label="处理记录" name="actions">
            <el-skeleton v-if="loading" :rows="6" animated />
            <PageLoadError v-else-if="error" :message="error" @retry="fetchActions" />
            <el-empty v-else-if="actions.length === 0" description="暂无内容治理操作" />
            <div v-else class="action-list">
              <article v-for="action in actions" :key="action.id" class="action-card">
                <div>
                  <strong>{{ actionLabel(action.action) }}{{ targetTypeLabel(action.target_type) }}</strong>
                  <span>#{{ action.target_id }} · 活动 #{{ action.event_id }}</span>
                </div>
                <p>{{ action.reason }}</p>
                <small>{{ action.actor }} · {{ formatDateTime(action.created_at) }}</small>
              </article>
            </div>
            <Pagination
              :total="actionsTotal"
              :current-page="actionsPage"
              :page-size="pageSize"
              @page-change="onActionsPageChange"
              @size-change="onActionsSizeChange"
            />
          </el-tab-pane>
        </el-tabs>
      </div>
    </el-main>

    <el-dialog v-model="actionDialogVisible" :title="actionDialogTitle" width="min(92vw, 520px)" destroy-on-close>
      <el-form label-position="top" @submit.prevent="submitAction">
        <el-form-item label="处理说明">
          <el-input
            v-model="actionNote"
            type="textarea"
            :rows="4"
            maxlength="1000"
            show-word-limit
            placeholder="填写处理说明"
          />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="actionDialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="submitting" @click="submitAction">确认处理</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElDialog, ElMessage } from 'element-plus'
import type {
  ContentModerationAction,
  ContentReport,
  ContentReportCategory,
  ContentReportStatus,
  ContentTargetType,
} from '@/api/types'
import { listContentModerationActions, listContentReports, resolveContentReport } from '@/api/admin'
import { removePost, removeReply, restorePost, restoreReply } from '@/api/posts'
import { formatDateTime } from '@/utils/format'
import { requestErrorMessage } from '@/utils/request-error'
import NavBar from '@/components/NavBar.vue'
import AdminToolbar from '@/components/AdminToolbar.vue'
import PageLoadError from '@/components/PageLoadError.vue'
import Pagination from '@/components/Pagination.vue'

type ActionMode = 'resolve-remove' | 'resolve-dismiss' | 'remove' | 'restore'

const activeTab = ref('reports')
const statusFilter = ref<ContentReportStatus>('open')
const targetTypeFilter = ref<ContentTargetType | ''>('')
const reports = ref<ContentReport[]>([])
const actions = ref<ContentModerationAction[]>([])
const reportsTotal = ref(0)
const actionsTotal = ref(0)
const reportsPage = ref(1)
const actionsPage = ref(1)
const pageSize = ref(10)
const loading = ref(false)
const error = ref('')
const actionDialogVisible = ref(false)
const actionMode = ref<ActionMode>('resolve-remove')
const currentReport = ref<ContentReport | null>(null)
const actionNote = ref('')
const submitting = ref(false)

const actionDialogTitle = computed(() => ({
  'resolve-remove': '移除内容并处理举报',
  'resolve-dismiss': '驳回举报',
  remove: '直接移除内容',
  restore: '恢复内容',
}[actionMode.value]))

const categoryLabels: Record<ContentReportCategory, string> = {
  spam: '垃圾广告', abuse: '辱骂或骚扰', illegal: '违法违规', privacy: '隐私泄露', other: '其他',
}

function categoryLabel(value: ContentReportCategory) { return categoryLabels[value] || value }
function targetTypeLabel(value: ContentTargetType) { return value === 'post' ? '帖子' : '回复' }
function statusLabel(value: ContentReportStatus) {
  return value === 'open' ? '待处理' : value === 'resolved' ? '已移除处理' : '已驳回'
}
function statusType(value: ContentReportStatus) {
  return value === 'open' ? 'warning' : value === 'resolved' ? 'danger' : 'info'
}
function actionLabel(value: ContentModerationAction['action']) {
  return value === 'remove' ? '移除' : value === 'restore' ? '恢复' : '驳回'
}

async function fetchReports() {
  loading.value = true
  error.value = ''
  try {
    const response = await listContentReports({
      status: statusFilter.value,
      target_type: targetTypeFilter.value || undefined,
      page: reportsPage.value,
      page_size: pageSize.value,
    })
    reports.value = response.data || []
    reportsTotal.value = response.total || 0
  } catch (cause) {
    reports.value = []
    reportsTotal.value = 0
    error.value = requestErrorMessage(cause, '举报队列加载失败，请稍后重试')
  } finally {
    loading.value = false
  }
}

async function fetchActions() {
  loading.value = true
  error.value = ''
  try {
    const response = await listContentModerationActions({ page: actionsPage.value, page_size: pageSize.value })
    actions.value = response.data || []
    actionsTotal.value = response.total || 0
  } catch (cause) {
    actions.value = []
    actionsTotal.value = 0
    error.value = requestErrorMessage(cause, '治理记录加载失败，请稍后重试')
  } finally {
    loading.value = false
  }
}

function refreshReports() {
  reportsPage.value = 1
  fetchReports()
}

function onTabChange(name: string | number) {
  if (name === 'actions') fetchActions()
  else fetchReports()
}

function openAction(report: ContentReport, mode: ActionMode) {
  currentReport.value = report
  actionMode.value = mode
  actionNote.value = ''
  actionDialogVisible.value = true
}

async function moderateTarget(report: ContentReport, remove: boolean, reason: string) {
  if (report.target_type === 'post') {
    return remove
      ? removePost(report.event_id, report.post_id, reason)
      : restorePost(report.event_id, report.post_id, reason)
  }
  return remove
    ? removeReply(report.event_id, report.post_id, report.target_id, reason)
    : restoreReply(report.event_id, report.post_id, report.target_id, reason)
}

async function submitAction() {
  if (!currentReport.value || !actionNote.value.trim()) {
    ElMessage.warning('请填写处理说明')
    return
  }
  submitting.value = true
  try {
    if (actionMode.value === 'resolve-remove') {
      await resolveContentReport(currentReport.value.id, 'remove', actionNote.value)
    } else if (actionMode.value === 'resolve-dismiss') {
      await resolveContentReport(currentReport.value.id, 'dismiss', actionNote.value)
    } else {
      await moderateTarget(currentReport.value, actionMode.value === 'remove', actionNote.value)
    }
    ElMessage.success('处理成功')
    actionDialogVisible.value = false
    await fetchReports()
  } finally {
    submitting.value = false
  }
}

function onReportsPageChange(value: number) { reportsPage.value = value; fetchReports() }
function onReportsSizeChange(value: number) { pageSize.value = value; reportsPage.value = 1; fetchReports() }
function onActionsPageChange(value: number) { actionsPage.value = value; fetchActions() }
function onActionsSizeChange(value: number) { pageSize.value = value; actionsPage.value = 1; fetchActions() }

onMounted(fetchReports)
</script>

<style scoped>
.page { min-height: 100vh; background: #f5f7fa; }
.main { padding-top: 24px; }
.container { max-width: 960px; margin: 0 auto; }
.header { margin-bottom: 16px; }
.header h2, .header p { margin: 0; }
.header p { margin-top: 6px; color: #606266; font-size: 14px; }
.filters { display: flex; gap: 12px; margin-bottom: 18px; }
.filters .el-select { width: 180px; }
.report-list, .action-list { display: grid; gap: 12px; }
.report-card, .action-card { padding: 20px; background: #fff; border: 1px solid #ebeef5; }
.report-topline { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; color: #909399; font-size: 13px; }
.tags { display: flex; flex-wrap: wrap; gap: 6px; }
.report-card h3 { margin: 16px 0 10px; font-size: 17px; }
.report-card blockquote { margin: 0; padding: 14px 16px; background: #f5f7fa; border-left: 3px solid #a0cfff; white-space: pre-wrap; overflow-wrap: anywhere; }
.report-meta { display: flex; flex-wrap: wrap; gap: 10px 24px; margin-top: 14px; color: #606266; font-size: 13px; }
.report-detail, .resolution { margin: 10px 0 0; color: #606266; }
.resolution { color: #909399; }
.report-actions { display: flex; align-items: center; flex-wrap: wrap; gap: 10px; margin-top: 16px; }
.report-actions .el-button { margin-left: 0; }
.report-actions a { color: #409eff; text-decoration: none; }
.action-card > div { display: flex; align-items: baseline; justify-content: space-between; gap: 16px; }
.action-card span, .action-card small { color: #909399; }
.action-card p { margin: 10px 0; color: #606266; }

@media (max-width: 640px) {
  .main { padding: 16px 8px; }
  .filters { align-items: stretch; flex-direction: column; }
  .filters .el-select { width: 100%; }
  .report-card, .action-card { padding: 16px; }
  .report-topline, .action-card > div { align-items: flex-start; flex-direction: column; gap: 8px; }
  .report-actions { align-items: stretch; flex-direction: column; }
  .report-actions .el-button, .report-actions a { width: 100%; margin-left: 0; text-align: center; }
}
</style>
