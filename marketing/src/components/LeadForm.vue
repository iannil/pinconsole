<script setup lang="ts">
import { ref, reactive, onMounted } from 'vue';
import type { PageContent } from '../content/types';

interface Props {
  content: PageContent['finalCTA']['form'];
  locale: 'zh' | 'en';
}

const props = defineProps<Props>();

const form = reactive({
  name: '',
  company: '',
  contact: '',
  purpose: '',
  message: '',
  website: '', // honeypot — must stay empty
});

const status = ref<'idle' | 'submitting' | 'success' | 'error'>('idle');
const turnstileToken = ref<string>('');
const turnstileReady = ref(false);
const turnstileWidgetId = ref<string | null>(null);

// Turnstile site key — public, browser-visible, safe to commit.
// Replace with your own from Cloudflare dashboard → Turnstile.
const TURNSTILE_SITE_KEY = '0x4AAAAAADpMi0AZU7xEXtzr';

const isEmailOrPhone = (s: string): boolean => {
  const email = /^[^\s@]+@[^\s@]+\.[^\s@]+$/;
  const phone = /^[+\d][\d\s\-()]{6,}$/;
  return email.test(s) || phone.test(s);
};

const renderTurnstile = () => {
  if (!TURNSTILE_SITE_KEY) {
    // No site key configured (local dev or unset) — bypass widget
    turnstileReady.value = true;
    return;
  }
  const turnstile = (window as any).turnstile;
  if (!turnstile) return;
  if (turnstileWidgetId.value !== null) return;
  turnstileWidgetId.value = turnstile.render('#turnstile-container', {
    sitekey: TURNSTILE_SITE_KEY,
    theme: 'dark',
    size: 'normal',
    callback: (token: string) => { turnstileToken.value = token; },
    'expired-callback': () => { turnstileToken.value = ''; },
    'error-callback': () => { turnstileToken.value = ''; },
  });
};

onMounted(() => {
  // Wait for turnstile script to load (FinalCTA injects it via head slot)
  if ((window as any).turnstile) {
    renderTurnstile();
  } else {
    const interval = setInterval(() => {
      if ((window as any).turnstile) {
        clearInterval(interval);
        renderTurnstile();
      }
    }, 200);
    setTimeout(() => clearInterval(interval), 5000);
  }
});

const onSubmit = async () => {
  if (status.value === 'submitting') return;

  // Honeypot — silently accept bots
  if (form.website) {
    status.value = 'success';
    return;
  }

  if (!form.name.trim() || !form.company.trim() || !form.contact.trim() || !form.purpose) {
    status.value = 'error';
    return;
  }

  if (!isEmailOrPhone(form.contact)) {
    status.value = 'error';
    return;
  }

  if (TURNSTILE_SITE_KEY && !turnstileToken.value) {
    status.value = 'error';
    return;
  }

  status.value = 'submitting';

  try {
    const res = await fetch('/api/leads', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        name: form.name.trim(),
        company: form.company.trim(),
        contact: form.contact.trim(),
        purpose: form.purpose,
        message: form.message.trim() || null,
        locale: props.locale,
        turnstileToken: turnstileToken.value || null,
      }),
    });

    if (!res.ok) throw new Error(`HTTP ${res.status}`);

    status.value = 'success';
    form.name = '';
    form.company = '';
    form.contact = '';
    form.purpose = '';
    form.message = '';
    turnstileToken.value = '';
    if (turnstileWidgetId.value !== null && (window as any).turnstile) {
      (window as any).turnstile.reset(turnstileWidgetId.value);
    }
  } catch (err) {
    console.error('lead form submit failed', err);
    status.value = 'error';
  }
};
</script>

<template>
  <form class="lead-form" @submit.prevent="onSubmit" novalidate>
    <div v-if="status === 'success'" class="lead-form-message success">
      {{ props.content.successMessage }}
    </div>
    <div v-if="status === 'error'" class="lead-form-message error">
      {{ props.content.errorMessage }}
    </div>

    <div class="lead-form-field">
      <label class="lead-form-label" for="lead-name">{{ props.content.nameLabel }}</label>
      <input
        id="lead-name"
        v-model="form.name"
        type="text"
        class="lead-form-input"
        :placeholder="props.content.namePlaceholder"
        :disabled="status === 'submitting' || status === 'success'"
        autocomplete="name"
        required
      />
    </div>

    <div class="lead-form-field">
      <label class="lead-form-label" for="lead-company">{{ props.content.companyLabel }}</label>
      <input
        id="lead-company"
        v-model="form.company"
        type="text"
        class="lead-form-input"
        :placeholder="props.content.companyPlaceholder"
        :disabled="status === 'submitting' || status === 'success'"
        autocomplete="organization"
        required
      />
    </div>

    <div class="lead-form-field">
      <label class="lead-form-label" for="lead-contact">{{ props.content.contactLabel }}</label>
      <input
        id="lead-contact"
        v-model="form.contact"
        type="text"
        class="lead-form-input"
        :placeholder="props.content.contactPlaceholder"
        :disabled="status === 'submitting' || status === 'success'"
        autocomplete="email"
        required
      />
    </div>

    <div class="lead-form-field">
      <label class="lead-form-label" for="lead-purpose">{{ props.content.purposeLabel }}</label>
      <select
        id="lead-purpose"
        v-model="form.purpose"
        class="lead-form-select"
        :disabled="status === 'submitting' || status === 'success'"
        required
      >
        <option value="" disabled>—</option>
        <option v-for="p in props.content.purposes" :key="p.value" :value="p.value">
          {{ p.label }}
        </option>
      </select>
    </div>

    <div class="lead-form-field">
      <label class="lead-form-label" for="lead-message">{{ props.content.messageLabel }}</label>
      <textarea
        id="lead-message"
        v-model="form.message"
        class="lead-form-textarea"
        :placeholder="props.content.messagePlaceholder"
        :disabled="status === 'submitting' || status === 'success'"
      />
    </div>

    <!-- Honeypot — hidden from humans, attractive to bots -->
    <div class="lead-form-honeypot" aria-hidden="true">
      <label for="lead-website">Website (leave empty)</label>
      <input
        id="lead-website"
        v-model="form.website"
        type="text"
        autocomplete="off"
        tabindex="-1"
      />
    </div>

    <!-- Cloudflare Turnstile (only renders if TURNSTILE_SITE_KEY is set) -->
    <div id="turnstile-container" class="lead-form-turnstile" />

    <button
      type="submit"
      class="lead-form-submit"
      :disabled="status === 'submitting' || status === 'success'"
    >
      {{ status === 'submitting' ? '...' : props.content.submitLabel }}
    </button>

    <p class="lead-form-privacy">{{ props.content.privacyNote }}</p>
  </form>
</template>
