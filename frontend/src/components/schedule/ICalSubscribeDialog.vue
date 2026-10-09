<script setup>
// 日历订阅：展示订阅地址与二维码（手机扫码即可在日历中订阅）
import { ref, watch, nextTick } from 'vue'
import { ElMessage } from 'element-plus'
import QRCode from 'qrcode'

const props = defineProps({
  url: { type: String, default: '' },
  webcal: { type: String, default: '' },
  expireAt: { type: String, default: '' }
})
const visible = defineModel({ type: Boolean, default: false })
const qrCanvas = ref(null)

watch([visible, () => props.url], async ([open, url]) => {
  if (!open || !url) return
  await nextTick()
  if (qrCanvas.value) {
    QRCode.toCanvas(qrCanvas.value, url, { width: 200, margin: 1 }).catch(() => {})
  }
})

const copy = async (text, msg) => {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success(msg)
  } catch {
    ElMessage.warning('复制失败，请手动复制')
  }
}
</script>

<template>
  <el-dialog v-model="visible" title="日历订阅" width="420px">
    <div class="ical-dialog-content">
      <p class="ical-hint">扫描下方二维码或在日历中添加以下订阅地址：</p>
      <div class="ical-qr-wrap">
        <canvas ref="qrCanvas" class="ical-qr-canvas"></canvas>
      </div>
      <div class="ical-links">
        <div class="ical-link-item">
          <span class="ical-link-label">HTTPS 订阅</span>
          <div class="ical-link-row">
            <el-input :model-value="url" readonly size="small" />
            <el-button size="small" @click="copy(url, 'HTTPS链接已复制')">复制</el-button>
          </div>
        </div>
        <div class="ical-link-item">
          <span class="ical-link-label">WebCal 订阅</span>
          <div class="ical-link-row">
            <el-input :model-value="webcal" readonly size="small" />
            <el-button size="small" @click="copy(webcal, 'WebCal链接已复制')">复制</el-button>
          </div>
        </div>
      </div>
      <p class="ical-expire">有效期至：{{ expireAt }}</p>
    </div>
  </el-dialog>
</template>

<style scoped>
.ical-dialog-content {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.ical-hint {
  font-size: 14px;
  color: var(--color-text-muted);
  text-align: center;
}

.ical-qr-wrap {
  display: flex;
  justify-content: center;
  padding: 16px;
  background: white;
  border-radius: var(--radius-lg);
}

.ical-qr-canvas {
  width: 200px;
  height: 200px;
}

.ical-links {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.ical-link-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.ical-link-label {
  font-size: 12px;
  font-weight: 500;
  color: var(--color-text-muted);
}

.ical-link-row {
  display: flex;
  gap: 8px;
}

.ical-expire {
  font-size: 12px;
  color: var(--color-text-muted);
  text-align: center;
}
</style>
