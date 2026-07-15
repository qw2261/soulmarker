<template>
  <div class="credential" :class="{ inactive: admission.status !== 'active' }">
    <div class="credential-main">
      <div class="qr-frame">
        <img v-if="qrDataURL" :src="qrDataURL" alt="入场凭证二维码" width="184" height="184" />
        <el-skeleton-item v-else variant="image" class="qr-placeholder" />
      </div>
      <div class="credential-info">
        <div class="status-line">
          <el-tag :type="statusType" effect="dark">{{ statusLabel }}</el-tag>
          <span v-if="admission.ticket_name">{{ admission.ticket_name }}</span>
        </div>
        <p v-if="admission.checked_in_at" class="checked-time">
          核销时间：{{ formatDateTime(admission.checked_in_at) }}
        </p>
        <p class="code">{{ admission.credential_code }}</p>
        <el-button :icon="DocumentCopy" @click.stop="copyCode">复制凭证码</el-button>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import QRCode from 'qrcode'
import { DocumentCopy } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import type { Admission } from '@/api/types'
import { formatDateTime } from '@/utils/format'

const props = defineProps<{ admission: Admission }>()
const qrDataURL = ref('')

const statusLabel = computed(() => {
  if (props.admission.status === 'revoked') return '已取消'
  if (props.admission.checked_in_at) return '已入场'
  return '待入场'
})

const statusType = computed(() => {
  if (props.admission.status === 'revoked') return 'danger'
  if (props.admission.checked_in_at) return 'success'
  return 'primary'
})

watch(
  () => props.admission.credential,
  async (credential) => {
    qrDataURL.value = await QRCode.toDataURL(credential, {
      width: 184,
      margin: 2,
      errorCorrectionLevel: 'M',
      color: { dark: '#111827', light: '#ffffff' },
    })
  },
  { immediate: true }
)

async function copyCode() {
  await navigator.clipboard.writeText(props.admission.credential)
  ElMessage.success('凭证码已复制')
}
</script>

<style scoped>
.credential {
  border: 1px solid #dcdfe6;
  background: #fff;
  padding: 16px;
}

.credential.inactive {
  background: #f5f7fa;
}

.credential-main {
  display: flex;
  align-items: center;
  gap: 20px;
}

.qr-frame {
  width: 184px;
  height: 184px;
  flex: 0 0 184px;
  background: #fff;
}

.qr-frame img,
.qr-placeholder {
  display: block;
  width: 184px;
  height: 184px;
}

.inactive .qr-frame {
  opacity: 0.35;
}

.credential-info {
  min-width: 0;
}

.status-line {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  color: #606266;
}

.checked-time {
  margin: 14px 0 0;
  color: #606266;
}

.code {
  max-width: 320px;
  margin: 14px 0;
  overflow-wrap: anywhere;
  color: #303133;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 13px;
}

@media (max-width: 560px) {
  .credential-main {
    flex-direction: column;
    align-items: stretch;
  }

  .qr-frame {
    margin: 0 auto;
  }
}
</style>
