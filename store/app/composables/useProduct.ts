import { ref, computed } from 'vue'
import type { ProductDetail } from '~~/shared/product'

export function useProduct(sku: string) {
  const product = ref<ProductDetail | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)
  const selectedQty = ref(1)

  const activeTierIndex = computed(() => {
    if (!product.value) return 0
    const qty = selectedQty.value
    const tiers = product.value.pricing.tiers
    for (let i = tiers.length - 1; i >= 0; i--) {
      const tier = tiers[i]
      if (!tier) continue
      const match = tier.range.match(/(\d+)\+?/)
      if (match?.[1] && qty >= Number(match[1])) return i
    }
    return 0
  })

  const currentUnitPrice = computed(() => {
    if (!product.value) return 0
    return (
      product.value.pricing.tiers[activeTierIndex.value]?.price ?? product.value.pricing.unitPrice
    )
  })

  const subtotal = computed(() => currentUnitPrice.value * selectedQty.value)

  async function fetchProduct() {
    loading.value = true
    error.value = null
    try {
      const data = await $fetch<ProductDetail>(`/api/product/${sku}`)
      product.value = data
      selectedQty.value = 1
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load product'
    } finally {
      loading.value = false
    }
  }

  function adjustQty(delta: number) {
    const next = selectedQty.value + delta
    const maxStock = product.value?.totalStock
    if (next >= 1 && (maxStock == null || maxStock === 0 || next <= maxStock)) {
      selectedQty.value = next
    }
  }

  function setQty(val: number) {
    selectedQty.value = Math.max(1, val)
  }

  return {
    product,
    loading,
    error,
    selectedQty,
    activeTierIndex,
    currentUnitPrice,
    subtotal,
    fetchProduct,
    adjustQty,
    setQty,
  }
}
