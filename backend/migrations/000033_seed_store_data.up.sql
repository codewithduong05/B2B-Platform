-- Seed data for Store frontend development
-- Represents a realistic B2B industrial supplies catalog

-- ============================================================
-- 1. ROLES
-- ============================================================
INSERT INTO identity.role (code, name, description, is_system) VALUES
  ('admin', 'Administrator', 'Full platform access', true),
  ('buyer', 'Buyer', 'Purchasing agent with catalog and order access', true),
  ('supplier_admin', 'Supplier Admin', 'Supplier portal administrator', true);

-- ============================================================
-- 2. USERS
-- ============================================================
-- Buyer user: password = "password" (bcrypt hash)
-- $2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy
INSERT INTO identity."user" (code, email, password_hash, user_type, is_active, is_verified) VALUES
  ('USER001', 'marcus.vance@apexindustrial.com', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'buyer', true, true),
  ('USER002', 'supplier@festo.com', '$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy', 'supplier', true, true);

-- ============================================================
-- 3. BUYER PROFILE
-- ============================================================
INSERT INTO identity.buyer_profile (code, user_id, business_name, trading_name, tax_id, phone, industry, employee_count, annual_revenue_minor, credit_limit_minor, credit_terms_days, currency) VALUES
  ('BUYER001', 1, 'Apex Industrial Corp', 'Apex Industrial', 'US-12-3456789', '+1-312-555-0100', 'Industrial Manufacturing', 2450, 185000000, 50000000, 30, 'USD');

-- ============================================================
-- 4. ADDRESSES
-- ============================================================
INSERT INTO identity.address (code, owner_user_id, owner_type, label, recipient_name, company_name, line1, line2, city, state_province, postal_code, country, phone, is_default) VALUES
  ('ADDR001', 1, 'buyer', 'Main Warehouse', 'Marcus Vance', 'Apex Industrial Corp', '4200 W Industrial Blvd', 'Building C', 'Chicago', 'IL', '60632', 'US', '+1-312-555-0101', true);

-- ============================================================
-- 5. USER ROLES
-- ============================================================
INSERT INTO identity.user_role (user_id, role_id) VALUES (1, 2); -- buyer role

-- ============================================================
-- 6. HANDLING CLASSES
-- ============================================================
INSERT INTO catalog.handling_class (code, name, description, sort_order, temperature_min_c, temperature_max_c) VALUES
  ('HC-AMBIENT', 'ambient', 'Standard ambient temperature storage', 1, 15, 30),
  ('HC-CHILLED', 'chilled', 'Cold chain storage required', 2, 2, 8),
  ('HC-FROZEN', 'frozen', 'Frozen storage required', 3, -25, -18);

-- ============================================================
-- 7. CATEGORIES (hierarchical)
-- ============================================================
-- Top-level categories
INSERT INTO catalog.category (code, name, slug, description, sort_order) VALUES
  ('CAT-IND', 'Industrial Supplies', 'industrial-supplies', 'Pneumatics, hydraulics, drives, and actuation', 1),
  ('CAT-ELEC', 'Electrical & Automation', 'electrical-automation', 'PLCs, VFDs, sensors, and control systems', 2),
  ('CAT-SAFE', 'Safety & PPE', 'safety-ppe', 'Personal protective equipment and safety systems', 3),
  ('CAT-PACK', 'Packaging & Logistics', 'packaging-logistics', 'Industrial packaging and logistics supplies', 4),
  ('CAT-FAC', 'Facility & Maintenance', 'facility-maintenance', 'HVAC, plumbing, and facility upkeep', 5),
  ('CAT-RAW', 'Raw Materials', 'raw-materials', 'Metals, polymers, and raw materials', 6);

-- Subcategories
INSERT INTO catalog.category (code, name, slug, description, parent_id, sort_order) VALUES
  ('CAT-PNEU', 'Pneumatics & Actuators', 'pneumatics-actuators', 'Cylinders, valves, and pneumatic actuators', 1, 1),
  ('CAT-HYDR', 'Hydraulics', 'hydraulics', 'Hydraulic pumps, cylinders, and systems', 1, 2),
  ('CAT-DRIVE', 'Drives & Motors', 'drives-motors', 'Servo drives, stepper motors, and gearboxes', 1, 3),
  ('CAT-PLC', 'PLCs & Controllers', 'plcs-controllers', 'Programmable logic controllers and I/O modules', 2, 1),
  ('CAT-SENS', 'Sensors', 'sensors', 'Proximity, photoelectric, and process sensors', 2, 2),
  ('CAT-VFD', 'VFDs & Drives', 'vfds-drives', 'Variable frequency drives and soft starters', 2, 3),
  ('CAT-PPE', 'Personal Protective Equipment', 'ppe', 'Safety helmets, gloves, and protective clothing', 3, 1);

-- ============================================================
-- 8. BRANDS
-- ============================================================
INSERT INTO catalog.brand (code, name, slug, description) VALUES
  ('BRD-FESTO', 'Festo', 'festo', 'Pneumatic and electrical automation technology'),
  ('BRD-PARKER', 'Parker Hannifin', 'parker-hannifin', 'Motion and control technologies'),
  ('BRD-SIEMENS', 'Siemens', 'siemens', 'Industrial automation and digitalization'),
  ('BRD-ABB', 'ABB', 'abb', 'Electrification and automation'),
  ('BRD-SCHNEIDER', 'Schneider Electric', 'schneider-electric', 'Energy management and automation'),
  ('BRD-SKF', 'SKF', 'skf', 'Bearings, seals, and lubrication systems'),
  ('BRD-BOSCH', 'Bosch Rexroth', 'bosch-rexroth', 'Drive and control technology'),
  ('BRD-APEX', 'Apex Dynamics', 'apex-dynamics', 'Precision gearboxes and drives');

-- ============================================================
-- 9. UNITS
-- ============================================================
INSERT INTO catalog.unit (code, name, symbol, unit_type) VALUES
  ('U-EA', 'Each', 'ea', 'base'),
  ('U-BOX', 'Box', 'box', 'base'),
  ('U-CASE', 'Case', 'case', 'base'),
  ('U-PALLET', 'Pallet', 'pallet', 'base'),
  ('U-M', 'Meter', 'm', 'base'),
  ('U-KG', 'Kilogram', 'kg', 'base'),
  ('U-L', 'Liter', 'L', 'base'),
  ('U-SET', 'Set', 'set', 'base');

-- ============================================================
-- 10. SUPPLIERS
-- ============================================================
INSERT INTO catalog.supplier (code, supplier_id, name, slug, description) VALUES
  ('SUP-FESTO', 2, 'Festo Authorized', 'festo-authorized', 'Festo authorized distributor - pneumatics and automation'),
  ('SUP-PARKER', 2, 'Parker Hannifin Direct', 'parker-hannifin-direct', 'Parker Hannifin direct supply - hydraulics and motion'),
  ('SUP-SIEMENS', 2, 'Siemens Industrial', 'siemens-industrial', 'Siemens industrial automation and controls'),
  ('SUP-APEX', 2, 'Apex Dynamics Partner', 'apex-dynamics-partner', 'Apex Dynamics precision gearboxes'),
  ('SUP-SCHNEIDER', 2, 'Schneider Electric', 'schneider-electric', 'Schneider Electric energy management'),
  ('SUP-SKF', 2, 'SKF Authorized', 'skf-authorized', 'SKF bearings and rotating equipment');

-- ============================================================
-- 11. ATTRIBUTES
-- ============================================================
INSERT INTO catalog.attribute (code, name, attribute_type, is_filterable, is_required, sort_order) VALUES
  ('ATT-BAR', 'Operating Pressure (bar)', 'number', true, false, 1),
  ('ATT-VDC', 'Voltage (VDC)', 'number', true, false, 2),
  ('ATT-STROKE', 'Stroke Length (mm)', 'number', false, false, 3),
  ('ATT-IP', 'IP Rating', 'text', true, false, 4),
  ('ATT-TEMP', 'Operating Temperature (°C)', 'text', false, false, 5),
  ('ATT-MAT', 'Body Material', 'text', true, false, 6),
  ('ATT-CONN', 'Connection Type', 'text', true, false, 7),
  ('ATT-ROHS', 'RoHS Compliant', 'boolean', true, false, 8),
  ('ATT-CE', 'CE Marked', 'boolean', true, false, 9);

-- ============================================================
-- 12. PRODUCTS
-- ============================================================
-- Products from Festo brand, Pneumatics category
INSERT INTO catalog.product (code, slug, name, description, short_description, category_id, brand_id, handling_class, base_unit_id, supplier_id, status, is_active, is_featured, sort_order, base_price_minor, currency, track_inventory, sku) VALUES
  ('PRD-VLV-9021', 'vlv-ind-9021', 'High-Pressure Solenoid Valve 24V DC', 'Industrial solenoid valve for high-pressure pneumatic systems. 24V DC coil, 5/2 way, port size G1/4. IP67 rated for harsh environments.', '5/2 way solenoid valve, 24V DC, G1/4 ports', 7, 1, 'ambient', 1, 1, 'published', true, true, 1, 18950, 'USD', true, 'VLV-IND-9021'),
  ('PRD-MTR-4402', 'mtr-el-4402', 'Industrial Stepper Motor NEMA 34', 'High-torque stepper motor, NEMA 34 frame. 3.0 Nm holding torque, 6.0A rated current. Bipolar series winding for precision positioning.', 'NEMA 34 stepper, 3.0 Nm, 6.0A', 9, 8, 'ambient', 1, 4, 'published', true, true, 2, 42500, 'USD', true, 'MTR-EL-4402'),
  ('PRD-CBL-8831', 'cbl-net-8831', 'Cat6A Industrial Shielded Cable (500m)', 'Industrial-grade Cat6A S/FTP cable. Shielded for EMI resistance. 500m reel, LSZH jacket, UV resistant for outdoor conduit runs.', 'Cat6A S/FTP, 500m, LSZH, UV resistant', 2, 5, 'ambient', 4, 5, 'published', true, true, 3, 89900, 'USD', true, 'CBL-NET-8831'),
  ('PRD-SNS-2010', 'sns-ind-2010', 'Inductive Proximity Sensor M18', 'M18 inductive proximity sensor, 8mm sensing range, PNP NO output. IP69K rated, IO-Link v1.1 compatible. Extended temperature range -25°C to +85°C.', 'M18 inductive, 8mm, PNP NO, IP69K', 11, 1, 'ambient', 1, 1, 'published', true, false, 4, 6750, 'USD', true, 'SNS-IND-2010'),
  ('PRD-CYL-5510', 'cyl-pne-5510', 'Pneumatic Linear Cylinder ISO 15552', 'ISO 15552 compliant pneumatic cylinder, 40mm bore, 200mm stroke. Double-acting, magnetic piston for position sensing. Anodized aluminum body.', 'ISO 15552 cylinder, 40mm bore, 200mm stroke', 7, 1, 'ambient', 1, 1, 'published', true, true, 5, 12400, 'USD', true, 'CYL-PNE-5510'),
  ('PRD-FIL-3300', 'fil-hyd-3300', 'Hydraulic Return Filter Element 10μm', '10 micron hydraulic return line filter element. Compatible with HLP 32-68 fluids. Glass fiber media, bypass valve integrated. Collapse pressure 3 bar.', '10μm return filter, glass fiber, 3 bar collapse', 8, 2, 'ambient', 1, 2, 'published', true, false, 6, 4200, 'USD', true, 'FIL-HYD-3300'),
  ('PRD-PLC-7701', 'plc-sim-7701', 'Siemens S7-1200 CPU 1214C DC/DC/DC', 'Compact CPU with 14 DI/10 DO/2 AI. PROFINET integrated, 100KB work memory. Supports TIA Portal V17+. Built-in web server for diagnostics.', 'S7-1200 CPU, 14DI/10DO/2AI, PROFINET', 10, 3, 'ambient', 1, 3, 'published', true, true, 7, 285000, 'USD', true, 'PLC-SIM-7701'),
  ('PRD-VFD-6600', 'vfd-abb-6600', 'ABB ACS580 Variable Frequency Drive 2.2kW', 'General purpose VFD, 2.2kW, 3-phase 380-480V. IP21 panel mount. Built-in C3 EMC filter, STO function. Compatible with ABB Ability.', 'ACS580 VFD, 2.2kW, 380-480V, IP21', 12, 4, 'ambient', 1, 3, 'published', true, false, 8, 345000, 'USD', true, 'VFD-ABB-6600'),
  ('PRD-GBX-8810', 'gbx-apex-8810', 'Apex Dynamics Planetary Gearbox AB Series', 'AB series inline planetary gearbox, ratio 5:1, 14mm output shaft. Backlash < 1 arcmin, rated torque 120Nm. IP65, food-grade lubricant option.', 'Planetary gearbox, 5:1, <1 arcmin, 120Nm', 9, 8, 'ambient', 1, 4, 'published', true, false, 9, 52800, 'USD', true, 'GBX-APEX-8810'),
  ('PRD-SEN-1100', 'sen-abb-1100', 'ABB Flow Meter Prometheus', 'Electromagnetic flow meter, DN50, 0.5-10 m/s range. 4-20mA + HART output. Stainless steel body, PTFE liner. ±0.2% accuracy.', 'EM flow meter, DN50, 4-20mA+HART', 11, 4, 'ambient', 1, 3, 'published', true, false, 10, 78000, 'USD', true, 'SEN-ABB-1100'),
  ('PRD-PSU-2200', 'psu-2200', 'Switching Power Supply 24VDC 10A', 'Industrial switching power supply, 24VDC 10A (240W). Universal input 90-264VAC. Short circuit, overload, and over-voltage protection. DIN rail mount.', '24VDC 10A PSU, DIN rail, 90-264VAC input', 2, 5, 'ambient', 1, 5, 'published', true, false, 11, 15800, 'USD', true, 'PSU-2200'),
  ('PRD-BRG-4400', 'brg-skf-4400', 'SKF Deep Groove Ball Bearing 6205-2RS', 'Deep groove ball bearing, 25x52x15mm. 2RS sealed, CN clearance. Max speed 12000 RPM. Chrome steel, pre-greased.', '6205-2RS, 25x52x15mm, sealed, CN', 1, 6, 'ambient', 1, 6, 'published', true, false, 12, 3200, 'USD', true, 'BRG-SKF-4400');

-- ============================================================
-- 13. PRODUCT MEDIA
-- ============================================================
INSERT INTO catalog.product_media (code, product_id, url, alt_text, media_type, sort_order, is_primary) VALUES
  ('MED-VLV-01', 1, '/images/products/vlv-ind-9021-main.jpg', 'High-Pressure Solenoid Valve 24V DC - Front View', 'image', 0, true),
  ('MED-VLV-02', 1, '/images/products/vlv-ind-9021-detail.jpg', 'Solenoid Valve - Port Detail', 'image', 1, false),
  ('MED-MTR-01', 2, '/images/products/mtr-el-4402-main.jpg', 'NEMA 34 Stepper Motor - Front View', 'image', 0, true),
  ('MED-CBL-01', 3, '/images/products/cbl-net-8831-main.jpg', 'Cat6A Shielded Cable - 500m Reel', 'image', 0, true),
  ('MED-SNS-01', 4, '/images/products/sns-ind-2010-main.jpg', 'Inductive Proximity Sensor M18', 'image', 0, true),
  ('MED-CYL-01', 5, '/images/products/cyl-pne-5510-main.jpg', 'Pneumatic Linear Cylinder ISO 15552', 'image', 0, true),
  ('MED-FIL-01', 6, '/images/products/fil-hyd-3300-main.jpg', 'Hydraulic Return Filter Element', 'image', 0, true),
  ('MED-PLC-01', 7, '/images/products/plc-sim-7701-main.jpg', 'Siemens S7-1200 CPU 1214C', 'image', 0, true),
  ('MED-VFD-01', 8, '/images/products/vfd-abb-6600-main.jpg', 'ABB ACS580 VFD 2.2kW', 'image', 0, true),
  ('MED-GBX-01', 9, '/images/products/gbx-apex-8810-main.jpg', 'Apex Dynamics Planetary Gearbox', 'image', 0, true),
  ('MED-SEN-01', 10, '/images/products/sen-abb-1100-main.jpg', 'ABB Flow Meter Prometheus', 'image', 0, true),
  ('MED-PSU-01', 11, '/images/products/psu-2200-main.jpg', 'Switching Power Supply 24VDC', 'image', 0, true),
  ('MED-BRG-01', 12, '/images/products/brg-skf-4400-main.jpg', 'SKF Deep Groove Ball Bearing 6205', 'image', 0, true);

-- ============================================================
-- 14. PRODUCT ATTRIBUTES
-- ============================================================
-- VLV-IND-9021 (Solenoid Valve)
INSERT INTO catalog.product_attribute (product_id, attribute_id, text_value) VALUES
  (1, 4, 'IP67'),
  (1, 5, '-10°C to +60°C'),
  (1, 6, 'Aluminum'),
  (1, 7, 'G1/4 BSP'),
  (1, 8, true),
  (1, 9, true);

-- MTR-EL-4402 (Stepper Motor)
INSERT INTO catalog.product_attribute (product_id, attribute_id, number_value) VALUES
  (2, 2, 24.0);

INSERT INTO catalog.product_attribute (product_id, attribute_id, text_value) VALUES
  (2, 4, 'IP65'),
  (2, 6, 'Aluminum/Steel');

-- CBL-NET-8831 (Cable)
INSERT INTO catalog.product_attribute (product_id, attribute_id, text_value) VALUES
  (3, 4, 'IP67'),
  (3, 6, 'LSZH');

-- SNS-IND-2010 (Sensor)
INSERT INTO catalog.product_attribute (product_id, attribute_id, number_value) VALUES
  (4, 1, 8.0),
  (4, 2, 24.0);

INSERT INTO catalog.product_attribute (product_id, attribute_id, text_value) VALUES
  (4, 4, 'IP69K'),
  (4, 5, '-25°C to +85°C'),
  (4, 6, 'Stainless Steel'),
  (4, 7, 'M12 connector');

-- CYL-PNE-5510 (Cylinder)
INSERT INTO catalog.product_attribute (product_id, attribute_id, number_value) VALUES
  (5, 1, 10.0),
  (5, 3, 200.0);

INSERT INTO catalog.product_attribute (product_id, attribute_id, text_value) VALUES
  (5, 4, 'IP67'),
  (5, 6, 'Anodized Aluminum');

-- FIL-HYD-3300 (Filter)
INSERT INTO catalog.product_attribute (product_id, attribute_id, text_value) VALUES
  (6, 4, 'IP67'),
  (6, 6, 'Glass Fiber/Steel');

-- PLC-SIM-7701 (PLC)
INSERT INTO catalog.product_attribute (product_id, attribute_id, number_value) VALUES
  (7, 2, 24.0);

INSERT INTO catalog.product_attribute (product_id, attribute_id, text_value) VALUES
  (7, 4, 'IP20');

-- ============================================================
-- 15. PRODUCT UNITS
-- ============================================================
INSERT INTO catalog.product_unit (product_id, unit_id, conversion_factor) VALUES
  (1, 1, 1),   -- Solenoid valve - each
  (2, 1, 1),   -- Stepper motor - each
  (3, 4, 1),   -- Cable - pallet (500m reel)
  (4, 1, 1),   -- Sensor - each
  (5, 1, 1),   -- Cylinder - each
  (6, 1, 1),   -- Filter - each
  (7, 1, 1),   -- PLC - each
  (8, 1, 1),   -- VFD - each
  (9, 1, 1),   -- Gearbox - each
  (10, 1, 1),  -- Flow meter - each
  (11, 1, 1),  -- PSU - each
  (12, 3, 1);  -- Bearing - case (10-pack)

-- ============================================================
-- 16. INVENTORY / STOCK LEVELS
-- ============================================================
INSERT INTO inventory.stock_level (code, product_id, supplier_id, available_quantity, reserved_quantity, total_quantity, safety_stock) VALUES
  ('SL-VLV-01', 1, 1, 156, 12, 168, 20),
  ('SL-MTR-01', 2, 4, 23, 3, 26, 5),
  ('SL-CBL-01', 3, 5, 45, 8, 53, 10),
  ('SL-SNS-01', 4, 1, 312, 18, 330, 50),
  ('SL-CYL-01', 5, 1, 87, 6, 93, 15),
  ('SL-FIL-01', 6, 2, 420, 24, 444, 60),
  ('SL-PLC-01', 7, 3, 8, 2, 10, 3),
  ('SL-VFD-01', 8, 3, 5, 1, 6, 2),
  ('SL-GBX-01', 9, 4, 18, 4, 22, 5),
  ('SL-SEN-01', 10, 3, 12, 0, 12, 3),
  ('SL-PSU-01', 11, 5, 210, 15, 225, 30),
  ('SL-BRG-01', 12, 6, 580, 40, 620, 100);

-- ============================================================
-- 17. PRICING
-- ============================================================
-- Active price list
INSERT INTO pricing.price_list (code, name, description, status, price_type, currency, effective_from) VALUES
  ('PL-STD-2025', 'Standard Price List 2025', 'Standard B2B pricing for all catalog items', 'active', 'standard', 'USD', '2025-01-01T00:00:00Z');

-- Price list items (one per product, base price)
INSERT INTO pricing.price_list_item (price_list_id, product_id, unit_id, currency, price_minor, min_quantity) VALUES
  (1, 1, 1, 'USD', 18950, 1),    -- Solenoid valve
  (1, 2, 1, 'USD', 42500, 1),    -- Stepper motor
  (1, 3, 4, 'USD', 89900, 1),    -- Cable
  (1, 4, 1, 'USD', 6750, 1),     -- Sensor
  (1, 5, 1, 'USD', 12400, 1),    -- Cylinder
  (1, 6, 1, 'USD', 4200, 1),     -- Filter
  (1, 7, 1, 'USD', 285000, 1),   -- PLC
  (1, 8, 1, 'USD', 345000, 1),   -- VFD
  (1, 9, 1, 'USD', 52800, 1),    -- Gearbox
  (1, 10, 1, 'USD', 78000, 1),   -- Flow meter
  (1, 11, 1, 'USD', 15800, 1),   -- PSU
  (1, 12, 3, 'USD', 3200, 1);    -- Bearing (per case)

-- Quantity tiers for solenoid valve (price_list_item_id = 1)
INSERT INTO pricing.quantity_tier (price_list_item_id, min_quantity, max_quantity, price_minor) VALUES
  (1, 1, 9, 18950),
  (1, 10, 49, 17055),     -- -10%
  (1, 50, 99, 15735),     -- -17%
  (1, 100, NULL, 14020);  -- -26%

-- Quantity tiers for stepper motor (price_list_item_id = 2)
INSERT INTO pricing.quantity_tier (price_list_item_id, min_quantity, max_quantity, price_minor) VALUES
  (2, 1, 4, 42500),
  (2, 5, 19, 40375),      -- -5%
  (2, 20, 49, 38250),     -- -10%
  (2, 50, NULL, 35275);   -- -17%

-- Quantity tiers for sensor (price_list_item_id = 4)
INSERT INTO pricing.quantity_tier (price_list_item_id, min_quantity, max_quantity, price_minor) VALUES
  (4, 1, 9, 6750),
  (4, 10, 49, 6075),      -- -10%
  (4, 50, 99, 5400),      -- -20%
  (4, 100, NULL, 4725);   -- -30%

-- Quantity tiers for cylinder (price_list_item_id = 5)
INSERT INTO pricing.quantity_tier (price_list_item_id, min_quantity, max_quantity, price_minor) VALUES
  (5, 1, 9, 12400),
  (5, 10, 49, 11780),     -- -5%
  (5, 50, 99, 10540),     -- -15%
  (5, 100, NULL, 9300);   -- -25%

-- Quantity tiers for filter (price_list_item_id = 6)
INSERT INTO pricing.quantity_tier (price_list_item_id, min_quantity, max_quantity, price_minor) VALUES
  (6, 1, 9, 4200),
  (6, 10, 49, 3780),      -- -10%
  (6, 50, 99, 3360),      -- -20%
  (6, 100, NULL, 2940);   -- -30%

-- Quantity tiers for bearing (price_list_item_id = 12)
INSERT INTO pricing.quantity_tier (price_list_item_id, min_quantity, max_quantity, price_minor) VALUES
  (12, 1, 9, 3200),
  (12, 10, 49, 2880),     -- -10%
  (12, 50, 99, 2560),     -- -20%
  (12, 100, NULL, 2240);  -- -30%
