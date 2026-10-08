-- Seed data for development
-- Password for all users: "password123" (hashed with bcrypt)

-- Insert admin user
INSERT INTO users (id, email, password, first_name, last_name, role, active, created_at, updated_at)
VALUES (
    '11111111-1111-1111-1111-111111111111',
    'admin@forgestock.com',
    '$2a$10$8K1p/a0dURXAm7QiTRqUzuLNzJ8J8J8J8J8J8J8J8J8J8J8J8J8J8J',
    'Admin',
    'User',
    'admin',
    true,
    NOW(),
    NOW()
);

-- Insert manager user
INSERT INTO users (id, email, password, first_name, last_name, role, active, created_at, updated_at)
VALUES (
    '22222222-2222-2222-2222-222222222222',
    'manager@forgestock.com',
    '$2a$10$8K1p/a0dURXAm7QiTRqUzuLNzJ8J8J8J8J8J8J8J8J8J8J8J8J8J8J',
    'Manager',
    'User',
    'manager',
    true,
    NOW(),
    NOW()
);

-- Insert warehouse operator user
INSERT INTO users (id, email, password, first_name, last_name, role, active, created_at, updated_at)
VALUES (
    '33333333-3333-3333-3333-333333333333',
    'operator@forgestock.com',
    '$2a$10$8K1p/a0dURXAm7QiTRqUzuLNzJ8J8J8J8J8J8J8J8J8J8J8J8J8J8J',
    'Warehouse',
    'Operator',
    'warehouse_operator',
    true,
    NOW(),
    NOW()
);

-- Insert customer user
INSERT INTO users (id, email, password, first_name, last_name, role, active, created_at, updated_at)
VALUES (
    '44444444-4444-4444-4444-444444444444',
    'customer@forgestock.com',
    '$2a$10$8K1p/a0dURXAm7QiTRqUzuLNzJ8J8J8J8J8J8J8J8J8J8J8J8J8J8J',
    'Customer',
    'User',
    'customer',
    true,
    NOW(),
    NOW()
);

-- Insert categories
INSERT INTO categories (id, name, description, created_at, updated_at) VALUES
('aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 'Electronics', 'Electronic devices and accessories', NOW(), NOW()),
('bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 'Clothing', 'Apparel and fashion items', NOW(), NOW()),
('cccccccc-cccc-cccc-cccc-cccccccccccc', 'Home & Garden', 'Home improvement and garden supplies', NOW(), NOW()),
('dddddddd-dddd-dddd-dddd-dddddddddddd', 'Sports & Outdoors', 'Sports equipment and outdoor gear', NOW(), NOW());

-- Insert products
INSERT INTO products (id, sku, name, description, category_id, price, cost, currency, active, created_at, updated_at) VALUES
('11111111-1111-1111-1111-111111111111', 'ELEC-001', 'Wireless Headphones', 'Premium noise-cancelling wireless headphones', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 199.99, 120.00, 'USD', true, NOW(), NOW()),
('22222222-2222-2222-2222-222222222222', 'ELEC-002', 'Smart Watch', 'Fitness tracking smartwatch with heart rate monitor', 'aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa', 299.99, 180.00, 'USD', true, NOW(), NOW()),
('33333333-3333-3333-3333-333333333333', 'CLTH-001', 'Cotton T-Shirt', '100% organic cotton t-shirt', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 24.99, 12.00, 'USD', true, NOW(), NOW()),
('44444444-4444-4444-4444-444444444444', 'CLTH-002', 'Denim Jeans', 'Classic fit denim jeans', 'bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb', 59.99, 30.00, 'USD', true, NOW(), NOW()),
('55555555-5555-5555-5555-555555555555', 'HOME-001', 'Coffee Maker', 'Programmable drip coffee maker', 'cccccccc-cccc-cccc-cccc-cccccccccccc', 79.99, 45.00, 'USD', true, NOW(), NOW()),
('66666666-6666-6666-6666-666666666666', 'SPRT-001', 'Yoga Mat', 'Non-slip exercise yoga mat', 'dddddddd-dddd-dddd-dddd-dddddddddddd', 29.99, 15.00, 'USD', true, NOW(), NOW());

-- Insert inventory for default warehouse
INSERT INTO inventory (id, product_id, warehouse_id, available_qty, reserved_qty, damaged_qty, reorder_threshold, version, created_at, updated_at) VALUES
('11111111-1111-1111-1111-111111111111', '11111111-1111-1111-1111-111111111111', 'default', 100, 0, 0, 10, 1, NOW(), NOW()),
('22222222-2222-2222-2222-222222222222', '22222222-2222-2222-2222-222222222222', 'default', 50, 0, 0, 5, 1, NOW(), NOW()),
('33333333-3333-3333-3333-333333333333', '33333333-3333-3333-3333-333333333333', 'default', 200, 0, 0, 20, 1, NOW(), NOW()),
('44444444-4444-4444-4444-444444444444', '44444444-4444-4444-4444-444444444444', 'default', 150, 0, 0, 15, 1, NOW(), NOW()),
('55555555-5555-5555-5555-555555555555', '55555555-5555-5555-5555-555555555555', 'default', 75, 0, 0, 8, 1, NOW(), NOW()),
('66666666-6666-6666-6666-666666666666', '66666666-6666-6666-6666-666666666666', 'default', 120, 0, 0, 12, 1, NOW(), NOW());

-- Insert sample orders
INSERT INTO orders (id, customer_id, status, subtotal, tax, discount, total, currency, notes, created_at, updated_at) VALUES
('11111111-1111-1111-1111-111111111111', '44444444-4444-4444-4444-444444444444', 'delivered', 224.98, 22.50, 0.00, 247.48, 'USD', 'First order', NOW() - INTERVAL '10 days', NOW() - INTERVAL '8 days'),
('22222222-2222-2222-2222-222222222222', '44444444-4444-4444-4444-444444444444', 'shipped', 59.99, 6.00, 0.00, 65.99, 'USD', '', NOW() - INTERVAL '5 days', NOW() - INTERVAL '3 days'),
('33333333-3333-3333-3333-333333333333', '44444444-4444-4444-4444-444444444444', 'processing', 379.97, 38.00, 0.00, 417.97, 'USD', 'Gift order', NOW() - INTERVAL '2 days', NOW() - INTERVAL '1 day');

-- Insert order items
INSERT INTO order_items (id, order_id, product_id, sku, name, quantity, price, total) VALUES
('11111111-1111-1111-1111-111111111111', '11111111-1111-1111-1111-111111111111', '11111111-1111-1111-1111-111111111111', 'ELEC-001', 'Wireless Headphones', 1, 199.99, 199.99),
('22222222-2222-2222-2222-222222222222', '11111111-1111-1111-1111-111111111111', '33333333-3333-3333-3333-333333333333', 'CLTH-001', 'Cotton T-Shirt', 1, 24.99, 24.99),
('33333333-3333-3333-3333-333333333333', '22222222-2222-2222-2222-222222222222', '44444444-4444-4444-4444-444444444444', 'CLTH-002', 'Denim Jeans', 1, 59.99, 59.99),
('44444444-4444-4444-4444-444444444444', '33333333-3333-3333-3333-333333333333', '22222222-2222-2222-2222-222222222222', 'ELEC-002', 'Smart Watch', 1, 299.99, 299.99),
('55555555-5555-5555-5555-555555555555', '33333333-3333-3333-3333-333333333333', '55555555-5555-5555-5555-555555555555', 'HOME-001', 'Coffee Maker', 1, 79.99, 79.99);

-- Insert payments
INSERT INTO payments (id, order_id, amount, currency, status, provider, provider_ref, created_at, updated_at) VALUES
('11111111-1111-1111-1111-111111111111', '11111111-1111-1111-1111-111111111111', 247.48, 'USD', 'captured', 'mock', 'mock_capture_11111111-1111-1111-1111-111111111111', NOW() - INTERVAL '10 days', NOW() - INTERVAL '8 days'),
('22222222-2222-2222-2222-222222222222', '22222222-2222-2222-2222-222222222222', 65.99, 'USD', 'captured', 'mock', 'mock_capture_22222222-2222-2222-2222-222222222222', NOW() - INTERVAL '5 days', NOW() - INTERVAL '3 days'),
('33333333-3333-3333-3333-333333333333', '33333333-3333-3333-3333-333333333333', 417.97, 'USD', 'authorized', 'mock', 'mock_auth_33333333-3333-3333-3333-333333333333', NOW() - INTERVAL '2 days', NOW() - INTERVAL '1 day');