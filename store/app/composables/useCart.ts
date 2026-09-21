import { ref, computed } from 'vue'

export interface CartItem {
  code: string
  product_code: string
  product_name: string
  quantity: number
  unit_price_minor: number
  total_price_minor: number
  currency: string
  supplier_code: string
  supplier_name: string
}

export interface Cart {
  code: string
  items: CartItem[]
  total_minor: number
  currency: string
  item_count: number
}

export function useCart() {
  const cart = ref<Cart | null>(null)
  const loading = ref(false)
  const error = ref<string | null>(null)

  const itemCount = computed(() => cart.value?.item_count || 0)
  const totalMinor = computed(() => cart.value?.total_minor || 0)
  const totalFormatted = computed(() => (totalMinor.value / 100).toFixed(2))

  async function fetchCart() {
    loading.value = true
    error.value = null
    try {
      const data = await $fetch<Cart>('/api/cart')
      cart.value = data
    } catch (e) {
      if (e && typeof e === 'object' && 'statusCode' in e && (e as { statusCode: number }).statusCode === 401) {
        cart.value = null
      } else {
        error.value = e instanceof Error ? e.message : 'Failed to load cart'
      }
    } finally {
      loading.value = false
    }
  }

  async function addToCart(productCode: string, quantity: number = 1) {
    loading.value = true
    error.value = null
    try {
      await $fetch('/api/cart/items', {
        method: 'POST',
        body: { product_code: productCode, quantity },
      })
      await fetchCart()
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to add item'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function updateQty(itemCode: string, quantity: number) {
    loading.value = true
    error.value = null
    try {
      await $fetch(`/api/cart/items/${itemCode}`, {
        method: 'PATCH',
        body: { quantity },
      })
      await fetchCart()
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to update item'
      throw e
    } finally {
      loading.value = false
    }
  }

  async function removeItem(itemCode: string) {
    loading.value = true
    error.value = null
    try {
      await $fetch(`/api/cart/items/${itemCode}`, {
        method: 'DELETE',
      })
      await fetchCart()
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to remove item'
      throw e
    } finally {
      loading.value = false
    }
  }

  return {
    cart,
    loading,
    error,
    itemCount,
    totalMinor,
    totalFormatted,
    fetchCart,
    addToCart,
    updateQty,
    removeItem,
  }
}
