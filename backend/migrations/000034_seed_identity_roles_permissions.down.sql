-- Remove seeded role-permission assignments
DELETE FROM identity.role_permission WHERE role_id IN (
    SELECT id FROM identity.role WHERE code IN ('super_admin', 'admin', 'staff', 'manager')
);
DELETE FROM identity.permission WHERE code IN (
    'buyers.view', 'buyers.manage', 'buyers.scope.manage',
    'verification.review', 'verification.configure',
    'orders.view', 'orders.edit', 'orders.hold', 'orders.dispatch',
    'shipments.manage', 'returns.manage', 'credit.manage',
    'inventory.view', 'inventory.adjust', 'inventory.quarantine',
    'promotions.view', 'promotions.manage',
    'partners.manage', 'leads.view', 'leads.configure',
    'suppliers.view', 'suppliers.manage',
    'cms.view', 'cms.edit', 'cms.legal',
    'reports.view', 'reports.export',
    'feature_flags.view', 'feature_flags.manage',
    'audit_log.view', 'integration_traffic.view',
    'system.ai', 'erp.view', 'erp.manage', 'system.view'
);
DELETE FROM identity.role WHERE code IN ('super_admin', 'admin', 'staff', 'manager');
