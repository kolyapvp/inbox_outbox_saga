INSERT INTO users (id, email) VALUES 
('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'user1@example.com'),
('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12', 'user2@example.com'),
('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13', 'user3@example.com'),
('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a14', 'user4@example.com'),
('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a15', 'user5@example.com')
ON CONFLICT (email) DO NOTHING;
