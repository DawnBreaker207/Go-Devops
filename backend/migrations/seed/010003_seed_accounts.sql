-- Password for every account: Demo@12345. Never use this seed in production.

INSERT INTO users (id, email, password, full_name, phone, role, accepted_terms_version)
VALUES
    ('30000000-0000-0000-0000-000000000001', 'staff1@demo.local', '$2a$10$ad0rOcwWNRgFUhtQBTeL6uArfCpd6LG0SQ4tTONVbp8CAK0h5e4TS', 'Nguyen Van Staff', '0900000001', 'staff', 1),
    ('30000000-0000-0000-0000-000000000002', 'staff2@demo.local', '$2a$10$ad0rOcwWNRgFUhtQBTeL6uArfCpd6LG0SQ4tTONVbp8CAK0h5e4TS', 'Tran Thi Staff',   '0900000002', 'staff', 1),
    ('30000000-0000-0000-0000-000000000011', 'customer1@demo.local', '$2a$10$ad0rOcwWNRgFUhtQBTeL6uArfCpd6LG0SQ4tTONVbp8CAK0h5e4TS', 'Le Van Khach',     '0900000011', 'customer', 1),
    ('30000000-0000-0000-0000-000000000012', 'customer2@demo.local', '$2a$10$ad0rOcwWNRgFUhtQBTeL6uArfCpd6LG0SQ4tTONVbp8CAK0h5e4TS', 'Pham Thi Khach',   '0900000012', 'customer', 1),
    ('30000000-0000-0000-0000-000000000013', 'customer3@demo.local', '$2a$10$ad0rOcwWNRgFUhtQBTeL6uArfCpd6LG0SQ4tTONVbp8CAK0h5e4TS', 'Hoang Van Khach',  '0900000013', 'customer', 1)
ON CONFLICT (id) DO NOTHING;
