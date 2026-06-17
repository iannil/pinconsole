<script setup lang="ts">
// 切片 1a hello world。
// 业务 UI（访客列表、实时回放、co-browsing 控件）从切片 1b 起加入。
import { useI18n } from 'vue-i18n';
import { ref, onMounted } from 'vue';

const { t, locale } = useI18n();
const healthStatus = ref<string>('checking...');

async function checkHealth() {
  try {
    const resp = await fetch('/healthz');
    const data = await resp.json();
    healthStatus.value = `HTTP ${resp.status}: ${data.status}`;
  } catch (e) {
    healthStatus.value = `error: ${(e as Error).message}`;
  }
}

onMounted(checkHealth);

function toggleLocale() {
  locale.value = locale.value === 'zh-CN' ? 'en-US' : 'zh-CN';
}
</script>

<template>
  <div class="hello">
    <h1>{{ t('app.title') }}</h1>
    <p>{{ t('app.hello') }}</p>
    <p>
      <el-button type="primary" @click="toggleLocale">
        {{ t('app.switch_lang') }}
      </el-button>
    </p>
    <p class="health">
      <strong>{{ t('app.health') }}:</strong> {{ healthStatus }}
    </p>
  </div>
</template>

<style scoped>
.hello {
  font-family: system-ui, -apple-system, sans-serif;
  max-width: 720px;
  margin: 4rem auto;
  padding: 0 1.5rem;
}
.hello h1 {
  font-size: 2rem;
  margin-bottom: 0.5rem;
}
.health {
  margin-top: 2rem;
  padding: 1rem;
  background: #f5f7fa;
  border-radius: 4px;
  font-family: ui-monospace, monospace;
}
</style>
