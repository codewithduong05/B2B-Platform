<script setup lang="ts">
definePageMeta({ title: 'Verification' })

const verification = ref<any>(null)
const loading = ref(true)
const error = ref('')
const submitting = ref(false)
const submitted = ref(false)

const form = ref({
  licence_number: '',
  licence_expiry: '',
  trading_name: '',
  business_address_line1: '',
  business_address_line2: '',
  business_city: '',
  business_state_province: '',
  business_postal_code: '',
  business_country: 'VN',
})

async function loadVerification() {
  loading.value = true
  error.value = ''
  try {
    verification.value = await $fetch('/api/account/verification')
  } catch (e: any) {
    if (e?.response?.status === 401) {
      error.value = 'Please sign in to view verification status.'
    } else {
      error.value = 'Failed to load verification status.'
    }
  } finally {
    loading.value = false
  }
}

async function submitVerification() {
  submitting.value = true
  error.value = ''
  try {
    await $fetch('/api/account/verification', {
      method: 'POST',
      body: form.value,
    })
    submitted.value = true
    await loadVerification()
  } catch (e: any) {
    error.value = e?.data?.message || 'Failed to submit verification.'
  } finally {
    submitting.value = false
  }
}

onMounted(loadVerification)
</script>

<template>
  <div class="verify-page">
    <nav class="verify-breadcrumb">
      <a href="/" class="verify-breadcrumb-link">Home</a>
      <span class="verify-breadcrumb-sep">/</span>
      <a href="/account" class="verify-breadcrumb-link">Account</a>
      <span class="verify-breadcrumb-sep">/</span>
      <span class="verify-breadcrumb-current">Verification</span>
    </nav>

    <h1 class="verify-title">Business Verification</h1>

    <div v-if="loading" class="verify-loading">Loading verification status...</div>
    <div v-else-if="error" class="verify-error">{{ error }}</div>
    <template v-else>
      <div v-if="verification?.status && verification.status !== 'not_submitted'" class="verify-status-card">
        <span class="material-symbols-outlined verify-status-icon">
          {{ verification.status === 'approved' ? 'check_circle' : verification.status === 'pending' ? 'pending' : 'info' }}
        </span>
        <div>
          <h3 class="verify-status-title">Verification Status: {{ verification.status }}</h3>
          <p class="verify-status-desc">
            {{ verification.status === 'approved' ? 'Your business has been verified.' :
               verification.status === 'pending' ? 'Your verification is under review.' :
               'Please update your verification details.' }}
          </p>
        </div>
      </div>

      <form v-if="!verification?.status || verification.status === 'not_submitted' || verification.status === 'rejected' || verification.status === 'resubmitted'" class="verify-form" @submit.prevent="submitVerification">
        <div class="verify-field">
          <label class="verify-label">Licence Number</label>
          <input v-model="form.licence_number" type="text" class="verify-input" required />
        </div>
        <div class="verify-field">
          <label class="verify-label">Licence Expiry</label>
          <input v-model="form.licence_expiry" type="date" class="verify-input" required />
        </div>
        <div class="verify-field">
          <label class="verify-label">Trading Name</label>
          <input v-model="form.trading_name" type="text" class="verify-input" required />
        </div>
        <div class="verify-field">
          <label class="verify-label">Business Address Line 1</label>
          <input v-model="form.business_address_line1" type="text" class="verify-input" required />
        </div>
        <div class="verify-field">
          <label class="verify-label">Business Address Line 2</label>
          <input v-model="form.business_address_line2" type="text" class="verify-input" />
        </div>
        <div class="verify-form-row">
          <div class="verify-field">
            <label class="verify-label">City</label>
            <input v-model="form.business_city" type="text" class="verify-input" required />
          </div>
          <div class="verify-field">
            <label class="verify-label">State / Province</label>
            <input v-model="form.business_state_province" type="text" class="verify-input" required />
          </div>
        </div>
        <div class="verify-form-row">
          <div class="verify-field">
            <label class="verify-label">Postal Code</label>
            <input v-model="form.business_postal_code" type="text" class="verify-input" required />
          </div>
          <div class="verify-field">
            <label class="verify-label">Country</label>
            <input v-model="form.business_country" type="text" class="verify-input" required maxlength="2" />
          </div>
        </div>
        <button type="submit" class="verify-submit" :disabled="submitting">
          {{ submitting ? 'Submitting...' : 'Submit for Verification' }}
        </button>
      </form>
    </template>
  </div>
</template>

<style scoped>
.verify-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-lg);
  max-width: 640px;
}

.verify-breadcrumb {
  display: flex;
  align-items: center;
  gap: var(--space-xs);
  font-size: var(--text-label-sm);
  color: var(--muted);
}

.verify-breadcrumb-link { color: var(--muted); text-decoration: none; }
.verify-breadcrumb-link:hover { color: var(--secondary); }
.verify-breadcrumb-sep { color: var(--outline-variant); }
.verify-breadcrumb-current { font-weight: 600; color: var(--on-surface); }

.verify-title { margin: 0; font-size: var(--text-headline-lg); font-weight: 700; }

.verify-loading, .verify-error {
  padding: var(--space-2xl);
  text-align: center;
  color: var(--muted);
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
}

.verify-error { color: var(--error); }

.verify-status-card {
  display: flex;
  align-items: flex-start;
  gap: var(--space-md);
  padding: var(--space-lg);
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
}

.verify-status-icon { font-size: 28px; color: var(--secondary); flex-shrink: 0; }
.verify-status-title { margin: 0; font-size: var(--text-headline-sm); }
.verify-status-desc { margin: 4px 0 0; color: var(--muted); font-size: var(--text-body-sm); }

.verify-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-md);
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: var(--space-lg);
}

.verify-form-row {
  display: flex;
  gap: var(--space-md);
}

.verify-form-row > .verify-field { flex: 1; }

.verify-field { display: flex; flex-direction: column; gap: var(--space-xs); }
.verify-label { font-size: var(--text-label-md); font-weight: 600; }

.verify-input {
  width: 100%;
  padding: var(--space-sm) var(--space-md);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--surface);
  font-family: var(--font-family);
  font-size: var(--text-body-md);
  outline: none;
}

.verify-input:focus { border-color: var(--secondary); box-shadow: 0 0 0 2px var(--secondary-fixed); }

.verify-submit {
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

.verify-submit:disabled { opacity: 0.5; cursor: not-allowed; }
</style>
