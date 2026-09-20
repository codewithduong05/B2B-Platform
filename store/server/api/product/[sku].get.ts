import type { ProductDetail } from '~~/shared/product'

const MOCK_PRODUCT: ProductDetail = {
  id: '1',
  sku: 'VLV-IND-9021',
  name: 'High-Pressure Solenoid Valve 24V DC - IP65 Rated',
  description:
    'Precision-engineered proportional pneumatic solenoid valve for industrial fluid power applications. 316L stainless steel body, Class H insulation, IP65 sealed.',
  mpn: 'MPN-FES-4491',
  supplier: {
    name: 'Apex Fluidic Solutions Ltd',
    id: 'SUP-0042',
    tier: 'Tier 1 MRO',
    audited: true,
    onTimeSla: '99.4%',
    qualityDefect: '< 0.02%',
    terms: 'Net 30',
    compliance: [
      'Accredited ISO 9001:2015 & ISO 14001 Production Facility',
      'Hassle-free 30-day RMA return window under Master Agreement #MA-77109',
      '24-Month OEM Festo Replacement Warranty & Seal Assurance',
    ],
  },
  pricing: {
    unitPrice: 142.5,
    unit: 'unit',
    tiers: [
      { id: 'tier-1', range: '1 - 9', price: 142.5, discount: 'Standard' },
      { id: 'tier-2', range: '10 - 49', price: 128.0, discount: '-10%' },
      { id: 'tier-3', range: '50 - 99', price: 118.0, discount: '-17% VIP' },
      { id: 'tier-4', range: '100+', price: 104.5, discount: '-26% MSA' },
    ],
  },
  stock: [
    {
      warehouse: 'Chicago Central Logistics Hub',
      detail: 'Priority Dispatch • Cutoff 4:30 PM CST',
      units: 320,
      transit: 'Same Day',
      priority: true,
    },
    {
      warehouse: 'Dallas Regional Depot',
      detail: 'Ground Freight Zone 3',
      units: 130,
      transit: '2-Day Transit',
      priority: false,
    },
    {
      warehouse: 'European Central (Stuttgart)',
      detail: 'Air Intermodal Re-allocation',
      units: 600,
      transit: '7-Day Transit',
      priority: false,
    },
  ],
  totalStock: 1050,
  specs: [
    { label: 'Operating Pressure', value: '0.5 to 16.0 bar (7.25 to 232 psi)', highlight: true },
    {
      label: 'Compatible Fluid Media',
      value: 'Filtered Compressed Air (40μm), Inert Neutral Gases',
    },
    { label: 'Valve Body Metallurgy', value: '316L Austenitic Stainless Steel (1.4404)' },
    {
      label: 'Coil Insulation Thermal Class',
      value: 'Class H 180°C (VDE 0580 Continuous Duty 100% ED)',
    },
    { label: 'Dynamic Response Time', value: '18 ms opening / 24 ms closing', highlight: true },
    { label: 'Ingress Enclosure Rating', value: 'IP65 (NEMA 4 equivalent with cable plug fitted)' },
    { label: 'Actuation Nominal Voltage', value: '24V DC (±10% tolerance) • 6.5W power draw' },
    { label: 'Internal Porting / Flow Factor', value: 'G 1/2" ISO 228 Female • Kv 3.8 m³/h' },
  ],
  images: [
    { label: 'ISO Front', alt: 'Isometric front elevation', active: true },
    { label: 'Cutaway', alt: 'Armature & Sealing Section' },
    { label: 'Terminal', alt: 'Electrical Terminal IP65' },
    { label: 'Manifold', alt: 'Manifold Integration Context' },
  ],
  artifacts: [
    {
      icon: 'picture_as_pdf',
      iconColor: 'error',
      title: 'Datasheet & Curves',
      description: 'PDF (4.2 MB) • EN/DE/FR',
    },
    {
      icon: 'view_in_ar',
      iconColor: 'secondary',
      title: '3D STEP Solid Model',
      description: 'ISO 10303 AP214 (12.8 MB)',
    },
    {
      icon: 'verified_user',
      iconColor: 'secondary',
      title: 'RoHS & REACH CoC',
      description: 'Signed Certificate (840 KB)',
    },
  ],
  category: ['Industrial Supplies', 'Valves', 'Solenoid Valves'],
  unspsc: '40141604',
  certifications: ['Pre-Qualified MRO Contract'],
}

export default defineEventHandler((event) => {
  const _sku = getRouterParam(event, 'sku')
  // In production, this would look up by SKU
  return MOCK_PRODUCT
})
