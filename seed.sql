INSERT INTO products (name, price, category, stock) VALUES
('ACER Aspire 5', 3500.00, 'Eletrônicos', 5),
('Geladeira Brastemp', 2500.00, 'Eletrodomésticos', 3),
('Desktop Gamer', 6000.00, 'Eletrônicos', 2),
('iPhone 15', 5500.00, 'Eletrônicos', 4),
('Samsung S26', 10000.00, 'Eletrônicos', 3),
('Fogão', 1200.00, 'Eletrodomésticos', 6),
('Cama Queen Ortobom', 1800.00, 'Móveis', 2),
('Tênis Nike Dunk SB', 1000.00, 'Sapatos', 8),
('Rolex', 30000.00, 'Acessórios', 1),
('Boné Lacoste', 300.00, 'Acessórios', 10);

INSERT INTO coupons (code, discount_type, discount_value, active) VALUES
('GO10', 'percentage', 10.0, 1),
('MERCADOGO50', 'fixed', 50.0, 1),
('PROMO20', 'percentage', 20.0, 1);

INSERT INTO users (id, password, name, email, phone, role, address_street, address_number, address_city, address_zip) VALUES
('elaine', '123456', 'Elaine Monteiro', 'elaine@email.com', '(11) 98888-1234', 'customer', 'Av. Paulista', '1578', 'São Paulo', '01310-200'),
('lucas', '123456', 'Lucas Monteiro Mobile', 'lucas.mobileit@outlook.com', '(11) 97777-5678', 'customer', 'Rua das Flores', '100', 'São Paulo', '01001-000'),
('admin', 'admin123', 'Administrador do Sistema', 'admin@mercadogo.com', '(11) 90000-0000', 'admin', 'Sede Central', '1', 'São Paulo', '01000-000');
