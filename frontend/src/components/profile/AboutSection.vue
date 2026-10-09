<script setup>
import SettingsSection from './SettingsSection.vue'
import SettingsItem from './SettingsItem.vue'
</script>

<template>
  <SettingsSection title="安全设置">
    <SettingsItem label="修改密码" arrow>
      <template #icon>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75"><circle cx="12" cy="12" r="3" /><path d="M12 1v2M12 21v2M4.22 4.22l1.42 1.42M18.36 18.36l1.42 1.42M1 12h2M21 12h2M4.22 19.78l1.42-1.42M18.36 5.64l1.42-1.42" /></svg>
      </template>
    </SettingsItem>
    <SettingsItem label="隐私政策" arrow>
      <template #icon>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" /></svg>
      </template>
    </SettingsItem>
    <SettingsItem label="关于我们" arrow>
      <template #icon>
        <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.75"><circle cx="12" cy="12" r="10" /><line x1="12" y1="16" x2="12" y2="12" /><line x1="12" y1="8" x2="12.01" y2="8" /></svg>
      </template>
    </SettingsItem>
  </SettingsSection>
</template>
