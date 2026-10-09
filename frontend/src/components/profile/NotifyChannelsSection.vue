<script setup>
import { ref, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import request from '../../utils/request'
import SettingsSection from './SettingsSection.vue'
import ActionItem from './ActionItem.vue'

// 通知方式：可同时开启多个；Webhook 在单独的分组里配置
const botTypes = [
  { value: 'wecom', label: '企业微信', placeholder: 'https://qyapi.weixin.qq.com/cgi-bin/webhook/send?key=...' },
  { value: 'dingtalk', label: '钉钉', placeholder: 'https://oapi.dingtalk.com/robot/send?access_token=...' },
  { value: 'feishu', label: '飞书', placeholder: 'https://open.feishu.cn/open-apis/bot/v2/hook/...' }
]
const channelNames = { webhook: 'Webhook', pushplus: '微信', email: '邮箱', bot: '群机器人' }

const emailAvailable = ref(false)
const form = ref({
  pushplusOn: false, pushplusToken: '',
  emailOn: false, emailTo: '',
  botOn: false, botType: 'wecom', botUrl: '', botSecret: ''
})
const saving = ref(false)
const testing = ref(false)
const botPlaceholder = computed(() => botTypes.find((b) => b.value === form.value.botType).placeholder)

onMounted(async () => {
  try {
    const res = await request.get('/notify/channels')
    const c = res.data?.channels || {}
    emailAvailable.value = !!res.data?.emailAvailable
    form.value = {
      pushplusOn: !!c.pushplus, pushplusToken: c.pushplus?.token || '',
      emailOn: !!c.email, emailTo: c.email?.to || '',
      botOn: !!c.bot, botType: c.bot?.type || 'wecom', botUrl: c.bot?.url || '', botSecret: c.bot?.secret || ''
    }
  } catch {}
})

const save = async () => {
  const f = form.value
  saving.value = true
  try {
    await request.put('/notify/channels', {
      pushplus: f.pushplusOn ? { token: f.pushplusToken } : null,
      email: f.emailOn ? { to: f.emailTo } : null,
      bot: f.botOn ? { type: f.botType, url: f.botUrl, secret: f.botType === 'wecom' ? '' : f.botSecret } : null
    })
    ElMessage.success('通知设置已保存')
  } catch {
  } finally {
    saving.value = false
  }
}

// 向所有已开启的渠道发送测试通知，逐个显示结果
const test = async () => {
  testing.value = true
  try {
    const results = (await request.post('/notify/test')).data?.results || []
    const failed = results.filter((r) => r.error)
    const name = (r) => channelNames[r.channel] || r.channel
    if (failed.length === 0) {
      ElMessage.success(`测试通知已发送（${results.map(name).join('、')}）`)
    } else {
      ElMessage({ type: 'warning', duration: 8000, message: failed.map((r) => `${name(r)}发送失败：${r.error}`).join('；') })
    }
  } catch {
  } finally {
    testing.value = false
  }
}
</script>

<template>
  <SettingsSection title="通知方式">
    <div class="apple-grouped-item">
      <div class="item-content channel-row">
        <div class="channel-head">
          <span class="item-label">微信（PushPlus）</span>
          <el-switch v-model="form.pushplusOn" size="small" />
        </div>
        <template v-if="form.pushplusOn">
          <el-input v-model="form.pushplusToken" placeholder="PushPlus token" size="small" />
          <span class="channel-hint">在 <a href="https://www.pushplus.plus" target="_blank" rel="noopener">pushplus.plus</a> 微信扫码登录后获取 token</span>
        </template>
      </div>
    </div>
    <div class="apple-grouped-item">
      <div class="item-content channel-row">
        <div class="channel-head">
          <span class="item-label">邮箱</span>
          <el-switch v-model="form.emailOn" size="small" :disabled="!emailAvailable && !form.emailOn" />
        </div>
        <el-input v-if="form.emailOn" v-model="form.emailTo" placeholder="you@example.com" size="small" />
        <span v-if="!emailAvailable" class="channel-hint">服务器未配置发信邮箱，暂不可用</span>
      </div>
    </div>
    <div class="apple-grouped-item">
      <div class="item-content channel-row">
        <div class="channel-head">
          <span class="item-label">群机器人</span>
          <el-switch v-model="form.botOn" size="small" />
        </div>
        <template v-if="form.botOn">
          <el-radio-group v-model="form.botType" size="small">
            <el-radio-button v-for="b in botTypes" :key="b.value" :value="b.value">{{ b.label }}</el-radio-button>
          </el-radio-group>
          <el-input v-model="form.botUrl" :placeholder="botPlaceholder" size="small" />
          <el-input v-if="form.botType !== 'wecom'" v-model="form.botSecret" placeholder="加签密钥（可选）" size="small" />
          <span v-if="form.botType === 'dingtalk'" class="channel-hint">钉钉机器人如设置了关键词，请包含「课表」</span>
        </template>
      </div>
    </div>
    <ActionItem text="保存通知设置" busy-text="保存中…" :busy="saving" @click="save" />
    <ActionItem text="发送测试通知" busy-text="发送中…" :busy="testing" @click="test" />
  </SettingsSection>
</template>
