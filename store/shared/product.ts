export interface ProductDetail {
  id: string
  sku: string
  name: string
  description: string
  mpn: string
  handling_class: string
  supplier: {
    name: string
    id: string
    tier: string
    audited: boolean
    onTimeSla: string
    qualityDefect: string
    terms: string
    compliance: string[]
  }
  pricing: {
    unitPrice: number
    unit: string
    tiers: PriceTier[]
  }
  stock: StockLocation[]
  totalStock: number
  specs: SpecRow[]
  images: ProductImage[]
  artifacts: Artifact[]
  category: string[]
  unspsc: string
  certifications: string[]
}

export interface PriceTier {
  id: string
  range: string
  price: number
  discount: string
}

export interface StockLocation {
  warehouse: string
  detail: string
  units: number
  transit: string
  priority: boolean
}

export interface SpecRow {
  label: string
  value: string
  highlight?: boolean
}

export interface ProductImage {
  label: string
  alt: string
  src?: string
  active?: boolean
}

export interface Artifact {
  icon: string
  iconColor: string
  title: string
  description: string
}
