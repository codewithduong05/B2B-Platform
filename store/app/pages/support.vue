<script setup lang="ts">
definePageMeta({ title: 'Support' })

const form = ref({
  name: '',
  email: '',
  subject: '',
  message: '',
})
const submitting = ref(false)
const submitted = ref(false)
const error = ref('')

async function handleSubmit() {
  submitting.value = true
  error.value = ''
  try {
    await $fetch('/api/support', {
      method: 'POST',
      body: form.value,
    })
    submitted.value = true
  } catch (e: any) {
    error.value = e?.data?.message || 'Failed to submit enquiry.'
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <div class="support-page">
    <nav class="support-breadcrumb">
      <a href="/" class="support-breadcrumb-link">Home</a>
      <span class="support-breadcrumb-sep">/</span>
      <span class="support-breadcrumb-current">Support</span>
    </nav>

    <h1 class="support-title">Enterprise Support</h1>

    <div v-if="submitted" class="support-success">
      <span class="material-symbols-outlined support-success-icon">check_circle</span>
      <h2>Enquiry Submitted</h2>
      <p>Our team will respond within 1 business day.</p>
      <a href="/" class="support-success-cta">Return Home →</a>
    </div>

    <form v-else class="support-form" @submit.prevent="handleSubmit">
      <div v-if="error" class="support-error">{{ error }}</div>
      <div class="support-row">
        <div class="support-field">
          <label class="support-label">Name</label>
          <input v-model="form.name" type="text" class="support-input" required />
        </div>
        <div class="support-field">
          <label class="support-label">Email</label>
          <input v-model="form.email" type="email" class="support-input" required />
        </div>
      </div>
      <div class="support-field">
        <label class="support-label">Subject</label>
        <input v-model="form.subject" type="text" class="support-input" required />
      </div>
      <div class="support-field">
        <label class="support-label">Message</label>
        <textarea v-model="form.message" class="support-textarea" rows="5" required />
      </div>
      <button type="submit" class="support-submit" :disabled="submitting">
        {{ submitting ? 'Sending...' : 'Send Enquiry' }}
      </button>
    </form>
  </div>
</template>

<style scoped>
.support-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-lg);
  max-width: 720px;
}

.support-breadcrumb {
  display: flex;
  align-items: center;
  gap: var(--space-xs);
  font-size: var(--text-label-sm);
  color: var(--muted);
}

.support-breadcrumb-link { color: var(--muted); text-decoration: none; }
.support-breadcrumb-link:hover { color: var(--secondary); }
.support-breadcrumb-sep { color: var(--outline-variant); }
.support-breadcrumb-current { font-weight: 600; color: var(--on-surface); }

.support-title { margin: 0; font-size: var(--text-headline-lg); font-weight: 700; }

.support-success {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-md);
  padding: var(--space-2xl);
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  text-align: center;
}

.support-success-icon { font-size: 48px; color: var(--secondary); }
.support-success h2 { margin: 0; }
.support-success p { margin: 0; color: var(--muted); }
.support-success-cta {
  padding: var(--space-sm) var(--space-lg);
  background: var(--primary);
  color: var(--on-primary);
  border-radius: var(--radius-lg);
  text-decoration: none;
  font-weight: 600;
}

.support-error {
  padding: var(--space-md);
  border-radius: var(--radius-lg);
  background: var(--error-bg, #fef2f2);
  color: var(--error);
  font-size: var(--text-body-sm);
}

.support-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-md);
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: var(--space-lg);
}

.support-row { display: grid; grid-template-columns: 1fr 1fr; gap: var(--space-md); }
.support-field { display: flex; flex-direction: column; gap: var(--space-xs); }
.support-label { font-size: var(--text-label-md); font-weight: 600; }

.support-input, .support-textarea {
  width: 100%;
  padding: var(--space-sm) var(--space-md);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--surface);
  font-family: var(--font-family);
  font-size: var(--text-body-md);
  outline: none;
}

.support-input:focus, .support-textarea:focus { border-color: var(--secondary); box-shadow: 0 0 0 2px var(--secondary-fixed); }
.support-textarea { resize: vertical; }

.support-submit {
  height: 44px;
  border: none;
  border-radius: var(--radius-lg);
  background: var(--primary);
  color: var(--on-primary);
  font-family: var(--font-family);
  font-size: var(--text-label-md);
  font-weight: 600;
  cursor: pointer;
}

.support-submit:disabled { opacity: 0.5; cursor: not-allowed; }
</style>
