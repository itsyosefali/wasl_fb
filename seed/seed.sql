INSERT INTO tenants (id, name, api_key)
VALUES (
    'a0000000-0000-4000-8000-000000000001',
    'Demo Tenant',
    'demo-api-key-change-in-production'
)
ON CONFLICT (api_key) DO NOTHING;
