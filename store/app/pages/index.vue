<script setup lang="ts">
definePageMeta({ title: 'Store' })
</script>

<template>
  <div class="home-page">
    <HomeHeroSection />
    <HomeMultiVendorSection />
    <HomeContractBanner />
    <HomeCategoryBento />
    <HomeReorderTray />
    <HomeComplianceStrip />
  </div>
</template>

<style scoped>
.home-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-xl);
}
</style>
