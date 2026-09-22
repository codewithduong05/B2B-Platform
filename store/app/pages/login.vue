<script setup lang="ts">
definePageMeta({ layout: false })

const email = ref('')
const password = ref('')
const showPassword = ref(false)
const loading = ref(false)
const errorMessage = ref('')
const route = useRoute()
const router = useRouter()

const returnTo = computed(() => {
  const r = route.query.return as string
  return r && r.startsWith('/') ? r : '/'
})

async function handleLogin() {
  if (!email.value || !password.value) return
  loading.value = true
  errorMessage.value = ''
  try {
    await $fetch('/api/auth/login', {
      method: 'POST',
      body: { email: email.value, password: password.value },
    })
    await navigateTo(returnTo.value)
  } catch (e: any) {
    const status = e?.response?.status || e?.statusCode
    if (status === 401) {
      errorMessage.value = 'Invalid email or password. Please check your credentials.'
    } else if (status === 429) {
      errorMessage.value = 'Too many login attempts. Please try again later.'
    } else if (status === 403) {
      errorMessage.value = 'Account is restricted. Please contact support.'
    } else {
      errorMessage.value = e?.data?.message || e?.message || 'Login failed. Please try again.'
    }
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="login-page">
    <div class="login-container">
      <div class="login-card">
        <div class="login-header">
          <span class="login-badge">B2B Enterprise Procurement</span>
          <h1 class="login-title">Sign In</h1>
          <p class="login-subtitle">Access your Atlas Wholesale procurement portal</p>
        </div>

        <div v-if="errorMessage" class="login-error">
          <span class="material-symbols-outlined login-error-icon">error</span>
          <span>{{ errorMessage }}</span>
        </div>

        <form class="login-form" @submit.prevent="handleLogin">
          <div class="login-field">
            <label class="login-label" for="email">
              Work Email <span class="login-required">*</span>
            </label>
            <div class="login-input-wrapper">
              <span class="material-symbols-outlined login-input-icon">business_center</span>
              <input
                id="email"
                v-model="email"
                type="email"
                class="login-input"
                placeholder="you@company.com"
                required
                autocomplete="email"
              />
            </div>
          </div>

          <div class="login-field">
            <div class="login-label-row">
              <label class="login-label" for="password">
                Password <span class="login-required">*</span>
              </label>
            </div>
            <div class="login-input-wrapper">
              <span class="material-symbols-outlined login-input-icon">key</span>
              <input
                id="password"
                v-model="password"
                :type="showPassword ? 'text' : 'password'"
                class="login-input login-input-password"
                placeholder="Enter your password"
                required
                autocomplete="current-password"
              />
              <button
                type="button"
                class="login-eye-btn"
                @click="showPassword = !showPassword"
              >
                <span class="material-symbols-outlined">
                  {{ showPassword ? 'visibility_off' : 'visibility' }}
                </span>
              </button>
            </div>
          </div>

          <button
            type="submit"
            class="login-submit"
            :disabled="loading || !email || !password"
          >
            <span v-if="loading" class="material-symbols-outlined login-spinner">progress_activity</span>
            <span v-else class="material-symbols-outlined">lock</span>
            {{ loading ? 'Signing in...' : 'Sign In' }}
          </button>
        </form>
      </div>

      <div class="login-info">
        <div class="login-info-card">
          <span class="material-symbols-outlined login-info-icon">verified_user</span>
          <div>
            <h3 class="login-info-title">Enterprise Security</h3>
            <p class="login-info-desc">TLS 1.3 encrypted session with 256-bit SSL</p>
          </div>
        </div>
        <div class="login-info-card">
          <span class="material-symbols-outlined login-info-icon">shield_lock</span>
          <div>
            <h3 class="login-info-title">B2B Authentication</h3>
            <p class="login-info-desc">Enterprise identity verification required for procurement access</p>
          </div>
        </div>
        <div class="login-info-card">
          <span class="material-symbols-outlined login-info-icon">support_agent</span>
          <div>
            <h3 class="login-info-title">Support</h3>
            <p class="login-info-desc">Need help? Contact your procurement administrator</p>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.login-page {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: var(--surface-container-low);
  padding: var(--space-lg);
}

.login-container {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-xl);
  max-width: 960px;
  width: 100%;
}

.login-card {
  background-color: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-xl);
  padding: var(--space-xl);
  display: flex;
  flex-direction: column;
  gap: var(--space-lg);
}

.login-header {
  display: flex;
  flex-direction: column;
  gap: var(--space-xs);
}

.login-badge {
  font-size: var(--text-label-sm);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.05em;
  color: var(--secondary);
  background-color: var(--secondary-fixed);
  padding: 2px 8px;
  border-radius: var(--radius);
  width: fit-content;
}

.login-title {
  font-size: var(--text-headline-lg);
  font-weight: 700;
  color: var(--on-surface);
  letter-spacing: var(--tracking-headline-lg);
  margin: 0;
}

.login-subtitle {
  margin: 0;
  font-size: var(--text-body-md);
  color: var(--muted);
}

.login-error {
  display: flex;
  align-items: flex-start;
  gap: var(--space-sm);
  padding: var(--space-md);
  border-radius: var(--radius-lg);
  background-color: var(--error-bg, #fef2f2);
  color: var(--error);
  font-size: var(--text-body-sm);
}

.login-error-icon {
  font-size: 20px;
  flex-shrink: 0;
  margin-top: 1px;
}

.login-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-md);
}

.login-field {
  display: flex;
  flex-direction: column;
  gap: var(--space-xs);
}

.login-label-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.login-label {
  font-size: var(--text-label-md);
  font-weight: 600;
  color: var(--on-surface);
}

.login-required {
  color: var(--error);
  font-weight: 700;
}

.login-forgot {
  border: none;
  background: transparent;
  font-family: var(--font-family);
  font-size: var(--text-label-sm);
  color: var(--secondary);
  cursor: pointer;
}

.login-forgot:hover {
  text-decoration: underline;
}

.login-input-wrapper {
  position: relative;
  display: flex;
  align-items: center;
}

.login-input-icon {
  position: absolute;
  left: 12px;
  font-size: 18px;
  color: var(--outline-variant);
  pointer-events: none;
}

.login-input {
  width: 100%;
  height: 44px;
  padding: 0 12px 0 40px;
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background-color: var(--surface);
  font-family: var(--font-family);
  font-size: var(--text-body-md);
  color: var(--on-surface);
  outline: none;
  transition: border-color 0.15s ease;
}

.login-input::placeholder {
  color: var(--outline-variant);
}

.login-input:focus {
  border-color: var(--secondary);
  box-shadow: 0 0 0 2px var(--secondary-fixed);
}

.login-input-password {
  padding-right: 44px;
}

.login-eye-btn {
  position: absolute;
  right: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: none;
  background: transparent;
  color: var(--outline-variant);
  cursor: pointer;
  border-radius: var(--radius);
}

.login-eye-btn:hover {
  color: var(--on-surface);
  background-color: var(--surface-container-low);
}

.login-submit {
  width: 100%;
  height: 44px;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-sm);
  border: none;
  border-radius: var(--radius-lg);
  background-color: var(--primary);
  color: var(--on-primary);
  font-family: var(--font-family);
  font-size: var(--text-label-md);
  font-weight: 600;
  cursor: pointer;
  transition: opacity 0.15s ease, transform 0.1s ease;
}

.login-submit:hover:not(:disabled) {
  opacity: 0.9;
}

.login-submit:active:not(:disabled) {
  transform: scale(0.99);
}

.login-submit:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.login-spinner {
  animation: spin 1s linear infinite;
}

@keyframes spin {
  from { transform: rotate(0deg); }
  to { transform: rotate(360deg); }
}

.login-footer {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-sm);
  padding-top: var(--space-md);
  border-top: 1px solid var(--border);
  font-size: var(--text-body-sm);
  color: var(--muted);
}

.login-register-link {
  color: var(--secondary);
  font-weight: 600;
  text-decoration: none;
}

.login-register-link:hover {
  text-decoration: underline;
}

.login-info {
  display: flex;
  flex-direction: column;
  gap: var(--space-md);
  justify-content: center;
}

.login-info-card {
  display: flex;
  align-items: flex-start;
  gap: var(--space-md);
  padding: var(--space-lg);
  background-color: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
}

.login-info-icon {
  font-size: 24px;
  color: var(--secondary);
  flex-shrink: 0;
}

.login-info-title {
  font-size: var(--text-headline-sm);
  font-weight: 700;
  color: var(--on-surface);
  margin: 0 0 4px;
}

.login-info-desc {
  margin: 0;
  font-size: var(--text-body-sm);
  color: var(--muted);
  line-height: 1.5;
}

@media (max-width: 768px) {
  .login-container {
    grid-template-columns: 1fr;
  }

  .login-info {
    display: none;
  }
}
</style>
