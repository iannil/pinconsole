<template>
  <div class="widgets-page">
    <h1>{{ t('widgets.title') }}</h1>

    <div v-if="loading" class="loading">{{ t('widgets.loading') }}</div>

    <template v-for="wt in widgetTypes" :key="wt">
      <section class="card">
        <h2>{{ t(`widgets.${wt}_title`) }}</h2>

        <!-- popup -->
        <template v-if="wt === 'popup'">
          <label>{{ t('widgets.title_field') }}</label>
          <input v-model="forms.popup.title" type="text" />
          <label>{{ t('widgets.body_field') }}</label>
          <textarea v-model="forms.popup.body" rows="3"></textarea>
          <label>{{ t('widgets.action_label') }}</label>
          <input v-model="forms.popup.action_label" type="text" />
          <label>{{ t('widgets.action_url') }}</label>
          <input v-model="forms.popup.action_url" type="text" />
          <label class="checkbox">
            <input v-model="forms.popup.dismissible" type="checkbox" />
            {{ t('widgets.dismissible') }}
          </label>
          <label>{{ t('widgets.primary_color') }}</label>
          <input v-model="forms.popup.primary_color" type="text" placeholder="#1a73e8" />
        </template>

        <!-- chat -->
        <template v-if="wt === 'chat'">
          <label>{{ t('widgets.header_field') }}</label>
          <input v-model="forms.chat.header" type="text" />
          <label>{{ t('widgets.placeholder_field') }}</label>
          <input v-model="forms.chat.placeholder" type="text" />
          <label>{{ t('widgets.send_label') }}</label>
          <input v-model="forms.chat.send_label" type="text" />
          <label>{{ t('widgets.bubble_color') }}</label>
          <input v-model="forms.chat.bubble_color" type="text" placeholder="#1a73e8" />
          <label>{{ t('widgets.header_color') }}</label>
          <input v-model="forms.chat.header_color" type="text" placeholder="#ffffff" />
        </template>

        <!-- cobrowse_banner -->
        <template v-if="wt === 'cobrowse_banner'">
          <label>{{ t('widgets.operator_label') }}</label>
          <input v-model="forms.cobrowse_banner.operator_label" type="text" />
          <label>{{ t('widgets.assist_hint') }}</label>
          <input v-model="forms.cobrowse_banner.assist_hint" type="text" />
          <label>{{ t('widgets.exit_label') }}</label>
          <input v-model="forms.cobrowse_banner.exit_label" type="text" />
        </template>

        <!-- consent_banner -->
        <template v-if="wt === 'consent_banner'">
          <label>{{ t('widgets.title_field') }}</label>
          <input v-model="forms.consent_banner.title" type="text" />
          <label>{{ t('widgets.body_field') }}</label>
          <textarea v-model="forms.consent_banner.body" rows="3"></textarea>
          <label>{{ t('widgets.accept_label') }}</label>
          <input v-model="forms.consent_banner.accept_label" type="text" />
          <label>{{ t('widgets.reject_label') }}</label>
          <input v-model="forms.consent_banner.reject_label" type="text" />
          <label>{{ t('widgets.privacy_link') }}</label>
          <input v-model="forms.consent_banner.privacy_link" type="text" />
        </template>

        <button class="save-btn" :disabled="saving[wt]" @click="onSave(wt)">
          {{ saving[wt] ? t('widgets.saving') : t('widgets.save') }}
        </button>
        <span v-if="saved[wt]" class="saved-msg">{{ t('widgets.saved_ok') }}</span>
      </section>
    </template>
  </div>
</template>

<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue';
import { useI18n } from 'vue-i18n';
import {
  fetchWidgetConfigs,
  upsertWidgetConfig,
  type PopupConfig,
  type ChatConfig,
  type CoBrowseBannerConfig,
  type ConsentBannerConfig,
} from '../api/widget-config';

const { t } = useI18n();

const widgetTypes = ['popup', 'chat', 'cobrowse_banner', 'consent_banner'] as const;

const loading = ref(true);

const forms = reactive({
  popup: { title: '', body: '', action_label: '', action_url: '', dismissible: true, primary_color: '' } as PopupConfig,
  chat: { header: '', placeholder: '', send_label: '', bubble_color: '', header_color: '' } as ChatConfig,
  cobrowse_banner: { operator_label: '', assist_hint: '', exit_label: '' } as CoBrowseBannerConfig,
  consent_banner: { title: '', body: '', accept_label: '', reject_label: '', privacy_link: '' } as ConsentBannerConfig,
});

const saving = reactive<Record<string, boolean>>({});
const saved = reactive<Record<string, boolean>>({});

onMounted(async () => {
  try {
    const res = await fetchWidgetConfigs();
    if (res.configs.popup) Object.assign(forms.popup, res.configs.popup);
    if (res.configs.chat) Object.assign(forms.chat, res.configs.chat);
    if (res.configs.cobrowse_banner) Object.assign(forms.cobrowse_banner, res.configs.cobrowse_banner);
    if (res.configs.consent_banner) Object.assign(forms.consent_banner, res.configs.consent_banner);
  } catch {
    // API unavailable — use defaults
  } finally {
    loading.value = false;
  }
});

async function onSave(wt: string) {
  saving[wt] = true;
  saved[wt] = false;
  try {
    await upsertWidgetConfig(wt, forms[wt as keyof typeof forms] as unknown as Record<string, unknown>);
    saved[wt] = true;
  } catch {
    // error handled silently
  } finally {
    saving[wt] = false;
  }
}
</script>

<style scoped>
.widgets-page {
  padding: 24px;
  max-width: 720px;
  margin: 0 auto;
}

.card {
  background: var(--pc-color-bg-surface);
  border: 1px solid var(--pc-color-border-subtle);
  border-radius: 8px;
  padding: 20px;
  margin-bottom: 24px;
}

.card h2 {
  margin: 0 0 16px;
  font-size: 18px;
}

label {
  display: block;
  margin-bottom: 4px;
  font-size: 13px;
  color: var(--pc-color-text-secondary);
}

input[type="text"],
textarea {
  width: 100%;
  padding: 8px 10px;
  border: 1px solid var(--pc-color-border-subtle);
  border-radius: 6px;
  font-size: 14px;
  margin-bottom: 12px;
  box-sizing: border-box;
  background: var(--pc-color-bg-input);
  color: var(--pc-color-text-primary);
}

textarea {
  resize: vertical;
}

.checkbox {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
}

.checkbox input {
  margin: 0;
}

.save-btn {
  padding: 8px 20px;
  background: var(--pc-color-teal-600, #0d9488);
  color: #fff;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  font-size: 14px;
}

.save-btn:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.saved-msg {
  margin-left: 12px;
  font-size: 13px;
  color: var(--pc-color-teal-600, #0d9488);
}

.loading {
  text-align: center;
  padding: 48px;
  color: var(--pc-color-text-secondary);
}
</style>
