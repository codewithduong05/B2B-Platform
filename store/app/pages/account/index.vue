<script setup lang="ts">
definePageMeta({ title: 'My Account' })

interface Address {
  id: string
  label: string
  address_line1: string
  address_line2: string
  city: string
  state: string
  postal_code: string
  country: string
  is_default: boolean
}

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

const addresses = ref<Address[]>([])
const loadingAddresses = ref(false)
const showAddressForm = ref(false)
const addressForm = ref({
  label: '',
  address_line1: '',
  address_line2: '',
  city: '',
  state: '',
  postal_code: '',
  country: '',
  is_default: false,
})
const savingAddress = ref(false)

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

async function loadAddresses() {
  loadingAddresses.value = true
  try {
    const data = await $fetch<{ items: Address[] }>('/api/account/addresses')
    addresses.value = data.items || []
  } catch {
  } finally {
    loadingAddresses.value = false
  }
}

async function saveAddress() {
  savingAddress.value = true
  try {
    await $fetch('/api/account/addresses', {
      method: 'POST',
      body: addressForm.value,
    })
    showAddressForm.value = false
    addressForm.value = { label: '', address_line1: '', address_line2: '', city: '', state: '', postal_code: '', country: '', is_default: false }
    await loadAddresses()
  } catch {
  } finally {
    savingAddress.value = false
  }
}

onMounted(() => {
  loadProfile()
  loadAddresses()
})
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
      <!-- Profile Section -->
      <div class="account-section">
        <h2 class="account-section-title">
          <span class="material-symbols-outlined">person</span>
          Profile
        </h2>
        <form class="account-form" @submit.prevent="saveProfile">
          <div class="account-fields-grid">
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
          </div>
          <div v-if="saveMessage" class="account-message">{{ saveMessage }}</div>
          <button type="submit" class="account-submit" :disabled="saving">
            {{ saving ? 'Saving...' : 'Save Changes' }}
          </button>
        </form>
      </div>

      <!-- Addresses Section -->
      <div class="account-section">
        <div class="account-section-header">
          <h2 class="account-section-title">
            <span class="material-symbols-outlined">location_on</span>
            Delivery Addresses
          </h2>
          <button class="account-add-btn" @click="showAddressForm = !showAddressForm">
            <span class="material-symbols-outlined">{{ showAddressForm ? 'close' : 'add' }}</span>
            {{ showAddressForm ? 'Cancel' : 'Add Address' }}
          </button>
        </div>

        <!-- Address Form -->
        <form v-if="showAddressForm" class="account-address-form" @submit.prevent="saveAddress">
          <div class="account-fields-grid">
            <div class="account-field">
              <label class="account-label">Label</label>
              <input v-model="addressForm.label" type="text" class="account-input" placeholder="e.g. Main Warehouse" />
            </div>
            <div class="account-field">
              <label class="account-label">Country</label>
              <input v-model="addressForm.country" type="text" class="account-input" placeholder="e.g. Vietnam" />
            </div>
            <div class="account-field account-field--full">
              <label class="account-label">Address Line 1</label>
              <input v-model="addressForm.address_line1" type="text" class="account-input" placeholder="Street address" />
            </div>
            <div class="account-field account-field--full">
              <label class="account-label">Address Line 2</label>
              <input v-model="addressForm.address_line2" type="text" class="account-input" placeholder="Suite, unit, building (optional)" />
            </div>
            <div class="account-field">
              <label class="account-label">City</label>
              <input v-model="addressForm.city" type="text" class="account-input" />
            </div>
            <div class="account-field">
              <label class="account-label">State / Province</label>
              <input v-model="addressForm.state" type="text" class="account-input" />
            </div>
            <div class="account-field">
              <label class="account-label">Postal Code</label>
              <input v-model="addressForm.postal_code" type="text" class="account-input" />
            </div>
            <div class="account-field">
              <label class="account-label-check">
                <input v-model="addressForm.is_default" type="checkbox" />
                Set as default
              </label>
            </div>
          </div>
          <button type="submit" class="account-submit" :disabled="savingAddress">
            {{ savingAddress ? 'Saving...' : 'Save Address' }}
          </button>
        </form>

        <!-- Address List -->
        <div v-if="loadingAddresses" class="account-loading-sm">Loading addresses...</div>
        <div v-else-if="addresses.length === 0 && !showAddressForm" class="account-empty">
          No addresses saved yet.
        </div>
        <div v-else class="account-address-list">
          <div v-for="addr in addresses" :key="addr.id" class="account-address-card">
            <div class="account-address-header">
              <span class="account-address-label">{{ addr.label || 'Address' }}</span>
              <span v-if="addr.is_default" class="account-address-default">Default</span>
            </div>
            <p class="account-address-body">
              {{ addr.address_line1 }}<br v-if="addr.address_line2" />
              <template v-if="addr.address_line2">{{ addr.address_line2 }}<br /></template>
              {{ addr.city }}<template v-if="addr.state">, {{ addr.state }}</template>
              {{ addr.postal_code }}<br />
              {{ addr.country }}
            </p>
          </div>
        </div>
      </div>

      <!-- Links -->
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
  max-width: 800px;
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

/* ── Sections ── */
.account-section {
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  padding: var(--space-lg);
  display: flex;
  flex-direction: column;
  gap: var(--space-md);
}

.account-section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.account-section-title {
  margin: 0;
  font-size: var(--text-headline-sm);
  font-weight: 700;
  display: flex;
  align-items: center;
  gap: var(--space-sm);
}

.account-section-title .material-symbols-outlined { font-size: 20px; color: var(--secondary); }

.account-add-btn {
  display: flex;
  align-items: center;
  gap: var(--space-xs);
  padding: var(--space-sm) var(--space-md);
  border: 1px solid var(--secondary);
  border-radius: var(--radius-lg);
  background: transparent;
  color: var(--secondary);
  font-family: var(--font-family);
  font-size: var(--text-label-sm);
  font-weight: 600;
  cursor: pointer;
}

.account-add-btn .material-symbols-outlined { font-size: 16px; }

/* ── Form ── */
.account-form, .account-address-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-md);
}

.account-fields-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-md);
}

.account-field { display: flex; flex-direction: column; gap: var(--space-xs); }
.account-field--full { grid-column: 1 / -1; }
.account-label { font-size: var(--text-label-md); font-weight: 600; }

.account-label-check {
  display: flex;
  align-items: center;
  gap: var(--space-xs);
  font-size: var(--text-label-md);
  font-weight: 600;
  cursor: pointer;
}

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

/* ── Addresses ── */
.account-loading-sm {
  padding: var(--space-md);
  text-align: center;
  color: var(--muted);
  font-size: var(--text-body-sm);
}

.account-empty {
  padding: var(--space-md);
  text-align: center;
  color: var(--muted);
  font-size: var(--text-body-sm);
  background: var(--surface-container-low);
  border-radius: var(--radius-lg);
}

.account-address-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-sm);
}

.account-address-card {
  padding: var(--space-md);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
}

.account-address-header {
  display: flex;
  align-items: center;
  gap: var(--space-sm);
  margin-bottom: var(--space-xs);
}

.account-address-label { font-weight: 600; color: var(--on-surface); }

.account-address-default {
  padding: 2px 8px;
  border-radius: var(--radius);
  background: var(--secondary-fixed);
  font-size: var(--text-label-sm);
  font-weight: 600;
  color: var(--on-secondary-fixed);
}

.account-address-body {
  margin: 0;
  font-size: var(--text-body-sm);
  color: var(--muted);
  line-height: 1.5;
}

/* ── Links ── */
.account-links {
  display: flex;
  flex-direction: column;
  gap: var(--space-sm);
}

.account-link { color: var(--secondary); font-weight: 600; text-decoration: none; font-size: var(--text-body-md); }
.account-link:hover { text-decoration: underline; }

@media (max-width: 640px) {
  .account-fields-grid { grid-template-columns: 1fr; }
}
</style>
