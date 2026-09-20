import type { CatalogProduct } from '~~/shared/catalog'

const MOCK_PRODUCTS: CatalogProduct[] = [
  {
    id: '1',
    sku: 'VLV-IND-9021',
    name: 'High-Pressure Solenoid Valve 24V DC',
    description: 'Proportional pneumatic regulation, IP65 rated, DIN EN ISO 4414 compliant.',
    supplier: 'Festo Authorized',
    price: 142.5,
    unit: 'unit',
    bulkPrice: 118.0,
    bulkLabel: 'Tier 1 Bulk: $118.00 at MOQ 10+',
    stock: 450,
    stockLabel: '450 units available',
    warehouse: '2 Regional Warehouses',
    leadTime: 'next-day',
    specs: '24V DC - Proportional - IP65',
    lowStock: false,
    moq: 1,
  },
  {
    id: '2',
    sku: 'MTR-EL-4402',
    name: 'Industrial Stepper Motor NEMA 34',
    description: '8.5 Nm torque, 1.8-degree step, Class H insulation.',
    supplier: 'Apex Dynamics',
    price: 285.0,
    unit: 'unit',
    stock: 82,
    stockLabel: '82 units in stock, MOQ: 2 Units',
    warehouse: 'FOB Chicago East',
    leadTime: 'next-day',
    specs: '8.5 Nm - 1.8 deg - Class H',
    lowStock: false,
    moq: 2,
  },
  {
    id: '3',
    sku: 'CBL-NET-8831',
    name: 'Cat6A Industrial Shielded Cable (500m)',
    description: 'S/FTP PUR jacket, 10M flex cycle rated, RoHS & UL AWM.',
    supplier: 'Belden Sourced',
    price: 620.0,
    unit: 'spool',
    bulkPrice: 1.24,
    bulkLabel: '$1.24/meter, Immediate Dispatch',
    stock: 34,
    stockLabel: '34 spools in stock',
    warehouse: 'RoHS & UL AWM',
    leadTime: '3-5-days',
    specs: 'S/FTP - PUR - 10M Flex',
    lowStock: false,
    moq: 1,
  },
  {
    id: '4',
    sku: 'HYD-CYL-102',
    name: 'Hydraulic Cylinder Double Acting 50mm Bore',
    description: '210 bar rated, ISO 6020/2 compliant, FKM seals.',
    supplier: 'Parker Hannifin',
    price: 410.0,
    unit: 'unit',
    stock: 14,
    stockLabel: '14 units remaining',
    warehouse: 'Factory replen: 8d',
    leadTime: 'factory-direct',
    specs: '210 Bar - ISO 6020/2',
    lowStock: true,
    moq: 1,
  },
  {
    id: '5',
    sku: 'BRG-FL-772',
    name: 'Heavy Duty Flange Bearing Unit 40mm',
    description: '4-bolt square, cast iron housing, 30.7kN dynamic load.',
    supplier: 'SKF Authorized',
    price: 54.2,
    unit: 'unit',
    bulkPrice: 48.5,
    bulkLabel: 'Case of 10: $48.50, In Stock Hub 1',
    stock: 1200,
    stockLabel: '1,200 available',
    warehouse: 'Multi-hub inventory',
    leadTime: 'next-day',
    specs: '4-Bolt Square - Cast Iron - 30.7kN',
    lowStock: false,
    moq: 1,
  },
  {
    id: '6',
    sku: 'SW-EST-019',
    name: 'Emergency Stop Pushbutton IP67',
    description: '22mm turn-to-release, SIL 3 rated, IP67 sealed.',
    supplier: 'Schneider Electric',
    price: 38.9,
    unit: 'unit',
    bulkPrice: 35.0,
    bulkLabel: 'Pack of 5: $175.00',
    stock: 310,
    stockLabel: '310 available',
    warehouse: 'Next-day ready',
    leadTime: 'next-day',
    specs: '22mm Turn-release - SIL 3',
    lowStock: false,
    moq: 1,
  },
]

export default defineEventHandler((event) => {
  const query = getQuery(event)
  const page = Number(query.page) || 1
  const perPage = Number(query.perPage) || 24
  const sortBy = String(query.sortBy || 'contract_low')
  const q = String(query.q || '').toLowerCase()
  const suppliers = query.suppliers ? String(query.suppliers).split(',') : []

  let filtered = [...MOCK_PRODUCTS]

  if (q) {
    filtered = filtered.filter(
      (p) =>
        p.name.toLowerCase().includes(q) ||
        p.sku.toLowerCase().includes(q) ||
        p.description.toLowerCase().includes(q),
    )
  }

  if (suppliers.length) {
    filtered = filtered.filter((p) => suppliers.includes(p.supplier))
  }

  // Sort
  if (sortBy === 'contract_low') {
    filtered.sort((a, b) => a.price - b.price)
  } else if (sortBy === 'stock_high') {
    filtered.sort((a, b) => b.stock - a.stock)
  }

  const start = (page - 1) * perPage
  const products = filtered.slice(start, start + perPage)

  return {
    products,
    total: filtered.length,
    page,
    perPage,
    sortBy,
  }
})
