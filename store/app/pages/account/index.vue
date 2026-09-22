<script setup lang="ts">
definePageMeta({ title: 'My Account' })

const profile = ref<any>(null)
const loading = ref(true)
const error = ref('')
const saving = ref(false)
const saveMessage = ref('')

const form = ref({
  business_name: '',
  trading_name: '',
  phone: '',
  website: '',
})

async function loadProfile() {
  loading.value = true
  error.value = ''
  try {
    profile.value = await $fetch('/api/account/profile')
    form.value.business_name = profile.value.business_name || ''
    form.value.trading_name = profile.value.trading_name || ''
    form.value.phone = profile.value.phone || ''
    form.value.website = profile.value.website || ''
  } catch (e: any) {
    if (e?.response?.status === 401) {
      error.value = 'Please sign in to view your account.'
    } else {
      error.value = 'Failed to load profile.'
    }
  } finally {
    loading.value = false
  }
}

async function saveProfile() {
  saving.value = true
  saveMessage.value = ''
  try {
    await $fetch('/api/account/profile', {
      method: 'PATCH',
      body: form.value,
    })
    saveMessage.value = 'Profile updated successfully.'
  } catch {
    saveMessage.value = 'Failed to update profile.'
  } finally {
    saving.value = false
  }
}

onMounted(loadProfile)
</script>

<template>
  <div class="account-page">
    <nav class="account-breadcrumb">
      <a href="/" class="account-breadcrumb-link">Home</a>
      <span class="account-breadcrumb-sep">/</span>
      <span class="account-breadcrumb-current">Account</span>
    </nav>

    <h1 class="account-title">My Account</h1>

    <div v-if="loading" class="account-loading">Loading profile...</div>
    <div v-else-if="error" class="account-error">{{ error }}</div>
    <template v-else>
      <form class="account-form" @submit.prevent="saveProfile">
        <div class="account-field">
          <label class="account-label">Business Name</label>
          <input v-model="form.business_name" type="text" class="account-input" />
        </div>
        <div class="account-field">
          <label class="account-label">Trading Name</label>
          <input v-model="form.trading_name" type="text" class="account-input" />
        </div>
        <div class="account-field">
          <label class="account-label">Phone</label>
          <input v-model="form.phone" type="tel" class="account-input" />
        </div>
        <div class="account-field">
          <label class="account-label">Website</label>
          <input v-model="form.website" type="url" class="account-input" />
        </div>
        <div v-if="saveMessage" class="account-message">{{ saveMessage }}</div>
        <button type="submit" class="account-submit" :disabled="saving">
          {{ saving ? 'Saving...' : 'Save Changes' }}
        </button>
      </form>

      <div class="account-links">
        <a href="/account/verification" class="account-link">Verification Status →</a>
      </div>
    </template>
  </div>
</template>

<style scoped>
.account-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-lg);
  max-width: 640px;
}

.account-breadcrumb {
  display: flex;
  align-items: center;
  gap: var(--space-xs);
  font-size: var(--text-label-sm);
  color: var(--muted);
}

.account-breadcrumb-link { color: var(--muted); text-decoration: none; }
.account-breadcrumb-link:hover { color: var(--secondary); }
.account-breadcrumb-sep { color: var(--outline-variant); }
.account-breadcrumb-current { font-weight: 600; color: var(--on-surface); }

.account-title { margin: 0; font-size: var(--text-headline-lg); font-weight: 700; }

.account-loading, .account-error {
  padding: var(--space-2xl);
  text-align: center;
  color: var(--muted);
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
}

.account-error { color: var(--error); }

.account-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-md);
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: var(--space-lg);
}

.account-field { display: flex; flex-direction: column; gap: var(--space-xs); }
.account-label { font-size: var(--text-label-md); font-weight: 600; }

.account-input {
  height: 44px;
  padding: 0 var(--space-md);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--surface);
  font-family: var(--font-family);
  font-size: var(--text-body-md);
  outline: none;
}

.account-input:focus { border-color: var(--secondary); box-shadow: 0 0 0 2px var(--secondary-fixed); }

.account-message {
  padding: var(--space-md);
  border-radius: var(--radius-lg);
  font-size: var(--text-body-sm);
  background: var(--surface-container-low);
}

.account-submit {
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

.account-submit:disabled { opacity: 0.5; cursor: not-allowed; }

.account-links {
  display: flex;
  flex-direction: column;
  gap: var(--space-sm);
}

.account-link { color: var(--secondary); font-weight: 600; text-decoration: none; font-size: var(--text-body-md); }
.account-link:hover { text-decoration: underline; }
</style>
