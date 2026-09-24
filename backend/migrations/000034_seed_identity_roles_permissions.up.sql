-- Seed identity roles and permissions (migrated from removed 000031)
-- Uses 000002 schema: role.code, permission.resource + permission.action

-- Seed default roles
INSERT INTO identity.role (code, name, description, is_system) VALUES
    ('super_admin', 'super_admin', 'Full system access', TRUE),
    ('admin', 'admin', 'Administrative access', TRUE),
    ('staff', 'staff', 'Standard staff access', TRUE),
    ('manager', 'manager', 'Managerial access', TRUE)
ON CONFLICT (code) DO NOTHING;

-- Seed permissions (resource.action pattern)
INSERT INTO identity.permission (code, name, description, resource, action) VALUES
    ('buyers.view', 'View buyer list and details', 'View buyer list and details', 'buyers', 'view'),
    ('buyers.manage', 'Manage buyer accounts', 'Manage buyer accounts', 'buyers', 'manage'),
    ('buyers.scope.manage', 'Manage purchase scopes', 'Manage purchase scopes', 'buyers', 'scope_manage'),
    ('verification.review', 'Review business verifications', 'Review business verifications', 'verification', 'review'),
    ('verification.configure', 'Configure verification rules', 'Configure verification rules', 'verification', 'configure'),
    ('orders.view', 'View orders', 'View orders', 'orders', 'view'),
    ('orders.edit', 'Edit orders', 'Edit orders', 'orders', 'edit'),
    ('orders.hold', 'Hold/release orders', 'Hold/release orders', 'orders', 'hold'),
    ('orders.dispatch', 'Dispatch orders', 'Dispatch orders', 'orders', 'dispatch'),
    ('shipments.manage', 'Manage shipments', 'Manage shipments', 'shipments', 'manage'),
    ('returns.manage', 'Manage returns and credit notes', 'Manage returns and credit notes', 'returns', 'manage'),
    ('credit.manage', 'Manage credit accounts', 'Manage credit accounts', 'credit', 'manage'),
    ('inventory.view', 'View inventory', 'View inventory', 'inventory', 'view'),
    ('inventory.adjust', 'Adjust stock levels', 'Adjust stock levels', 'inventory', 'adjust'),
    ('inventory.quarantine', 'Quarantine/release lots', 'Quarantine/release lots', 'inventory', 'quarantine'),
    ('promotions.view', 'View promotions', 'View promotions', 'promotions', 'view'),
    ('promotions.manage', 'Manage promotions', 'Manage promotions', 'promotions', 'manage'),
    ('partners.manage', 'Manage partners and referrals', 'Manage partners and referrals', 'partners', 'manage'),
    ('leads.view', 'View leads', 'View leads', 'leads', 'view'),
    ('leads.configure', 'Configure lead routing', 'Configure lead routing', 'leads', 'configure'),
    ('suppliers.view', 'View suppliers', 'View suppliers', 'suppliers', 'view'),
    ('suppliers.manage', 'Manage supplier profiles', 'Manage supplier profiles', 'suppliers', 'manage'),
    ('cms.view', 'View CMS content', 'View CMS content', 'cms', 'view'),
    ('cms.edit', 'Edit CMS content', 'Edit CMS content', 'cms', 'edit'),
    ('cms.legal', 'Manage legal documents', 'Manage legal documents', 'cms', 'legal'),
    ('reports.view', 'View reports', 'View reports', 'reports', 'view'),
    ('reports.export', 'Export reports', 'Export reports', 'reports', 'export'),
    ('feature_flags.view', 'View feature flags', 'View feature flags', 'feature_flags', 'view'),
    ('feature_flags.manage', 'Manage feature flags', 'Manage feature flags', 'feature_flags', 'manage'),
    ('audit_log.view', 'View audit log', 'View audit log', 'audit_log', 'view'),
    ('integration_traffic.view', 'View integration traffic', 'View integration traffic', 'integration_traffic', 'view'),
    ('system.ai', 'AI system administration', 'AI system administration', 'system', 'ai'),
    ('erp.view', 'View ERP sync jobs', 'View ERP sync jobs', 'erp', 'view'),
    ('erp.manage', 'Manage ERP sync', 'Manage ERP sync', 'erp', 'manage'),
    ('system.view', 'System administration', 'System administration', 'system', 'view')
ON CONFLICT (code) DO NOTHING;

-- Assign all permissions to super_admin
INSERT INTO identity.role_permission (role_id, permission_id)
SELECT r.id, p.id FROM identity.role r, identity.permission p
WHERE r.code = 'super_admin'
ON CONFLICT DO NOTHING;

-- Assign common permissions to admin
INSERT INTO identity.role_permission (role_id, permission_id)
SELECT r.id, p.id FROM identity.role r, identity.permission p
WHERE r.code = 'admin' AND p.resource IN ('buyers', 'verification', 'orders', 'shipments', 'returns', 'credit', 'inventory', 'promotions', 'partners', 'leads', 'suppliers', 'cms', 'reports', 'feature_flags', 'erp')
ON CONFLICT DO NOTHING;

-- Assign staff permissions
INSERT INTO identity.role_permission (role_id, permission_id)
SELECT r.id, p.id FROM identity.role r, identity.permission p
WHERE r.code = 'staff' AND p.resource IN ('buyers', 'orders', 'shipments', 'returns', 'inventory', 'promotions', 'suppliers', 'cms', 'reports')
ON CONFLICT DO NOTHING;

-- Assign manager permissions
INSERT INTO identity.role_permission (role_id, permission_id)
SELECT r.id, p.id FROM identity.role r, identity.permission p
WHERE r.code = 'manager' AND p.resource IN ('buyers', 'verification', 'orders', 'shipments', 'returns', 'credit', 'inventory', 'promotions', 'partners', 'leads', 'suppliers', 'cms', 'reports')
ON CONFLICT DO NOTHING;
