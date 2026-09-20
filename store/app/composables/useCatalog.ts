import { ref, computed } from 'vue'
import type { CatalogProduct, CatalogFilters, CatalogState } from '~~/shared/catalog'

const DEFAULT_FILTERS: CatalogFilters = {
  suppliers: [],
  leadTime: '',
  warehouses: [],
  moqMax: 100,
  compliance: [],
  query: '',
}

export function useCatalog() {
  const state = ref<CatalogState>({
    products: [],
    total: 0,
    page: 1,
    perPage: 24,
    filters: { ...DEFAULT_FILTERS },
    sortBy: 'contract_low',
    viewMode: 'grid',
  })

  const loading = ref(false)
  const error = ref<string | null>(null)

  const activeFilters = computed(() => {
    const f = state.value.filters
    const pills: { key: string; label: string }[] = []
    if (f.suppliers.length) {
      pills.push({ key: 'suppliers', label: f.suppliers.join(' & ') })
    }
    if (f.leadTime) {
      pills.push({ key: 'leadTime', label: f.leadTime })
    }
    if (f.moqMax < 100) {
      pills.push({ key: 'moqMax', label: `MOQ <= ${f.moqMax}` })
    }
    if (f.compliance.length) {
      pills.push({ key: 'compliance', label: f.compliance.join(', ') })
    }
    return pills
  })

  const totalPages = computed(() => Math.max(1, Math.ceil(state.value.total / state.value.perPage)))

  const paginatedRange = computed(() => {
    const { page, perPage, total } = state.value
    const start = (page - 1) * perPage + 1
    const end = Math.min(page * perPage, total)
    return { start, end, total }
  })

  async function fetchProducts() {
    loading.value = true
    error.value = null
    try {
      const params = new URLSearchParams()
      params.set('page', String(state.value.page))
      params.set('perPage', String(state.value.perPage))
      params.set('sortBy', state.value.sortBy)
      if (state.value.filters.query) params.set('q', state.value.filters.query)
      if (state.value.filters.suppliers.length) {
        params.set('suppliers', state.value.filters.suppliers.join(','))
      }
      if (state.value.filters.leadTime) {
        params.set('leadTime', state.value.filters.leadTime)
      }
      if (state.value.filters.moqMax < 100) {
        params.set('moqMax', String(state.value.filters.moqMax))
      }

      const data = await $fetch<{ products: CatalogProduct[]; total: number }>(
        `/api/catalog?${params.toString()}`,
      )
      state.value.products = data.products
      state.value.total = data.total
    } catch (e) {
      error.value = e instanceof Error ? e.message : 'Failed to load catalog'
    } finally {
      loading.value = false
    }
  }

  function setPage(page: number) {
    state.value.page = page
    fetchProducts()
  }

  function setSort(sortBy: string) {
    state.value.sortBy = sortBy
    state.value.page = 1
    fetchProducts()
  }

  function setViewMode(mode: 'grid' | 'table') {
    state.value.viewMode = mode
  }

  function removeFilter(key: string) {
    const f = state.value.filters
    if (key === 'suppliers') f.suppliers = []
    else if (key === 'leadTime') f.leadTime = ''
    else if (key === 'moqMax') f.moqMax = 100
    else if (key === 'compliance') f.compliance = []
    state.value.page = 1
    fetchProducts()
  }

  function applyFilters(filters: Partial<CatalogFilters>) {
    Object.assign(state.value.filters, filters)
    state.value.page = 1
    fetchProducts()
  }

  return {
    state,
    loading,
    error,
    activeFilters,
    totalPages,
    paginatedRange,
    fetchProducts,
    setPage,
    setSort,
    setViewMode,
    removeFilter,
    applyFilters,
  }
}
