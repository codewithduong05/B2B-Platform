export interface CatalogProduct {
  id: string
  sku: string
  name: string
  description: string
  supplier: string
  price: number
  unit: string
  bulkPrice?: number
  bulkLabel?: string
  stock: number
  stockLabel: string
  warehouse: string
  leadTime: string
  specs: string
  lowStock: boolean
  moq: number
  imageUrl?: string
}

export interface CatalogFilters {
  suppliers: string[]
  leadTime: string
  warehouses: string[]
  moqMax: number
  compliance: string[]
  query: string
}

export interface CatalogState {
  products: CatalogProduct[]
  total: number
  page: number
  perPage: number
  filters: CatalogFilters
  sortBy: string
  viewMode: 'grid' | 'table'
}
