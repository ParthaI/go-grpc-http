INSERT INTO products (id, name, description, price_cents, currency, stock_quantity, sku, created_at, updated_at)
VALUES
  ('a0000001-0000-0000-0000-000000000001', 'Mechanical Keyboard', 'Cherry MX Blue switches, RGB backlight, full size', 12999, 'USD', 50, 'KB-MECH-001', NOW(), NOW()),
  ('a0000001-0000-0000-0000-000000000002', 'Wireless Mouse', 'Ergonomic design, 2.4GHz wireless, 6 buttons', 4999, 'USD', 120, 'MS-WIFI-001', NOW(), NOW()),
  ('a0000001-0000-0000-0000-000000000003', 'USB-C Hub', '7-in-1: HDMI, USB 3.0 x3, SD, microSD, PD charging', 3999, 'USD', 75, 'HUB-USB7-001', NOW(), NOW()),
  ('a0000001-0000-0000-0000-000000000004', '4K Monitor 27"', 'IPS panel, 144Hz, HDR400, USB-C input', 44999, 'USD', 20, 'MON-4K27-001', NOW(), NOW()),
  ('a0000001-0000-0000-0000-000000000005', 'Webcam HD 1080p', 'Auto-focus, dual microphone, privacy shutter', 5999, 'USD', 90, 'CAM-HD-001', NOW(), NOW()),
  ('a0000001-0000-0000-0000-000000000006', 'Bluetooth Speaker', 'Waterproof IPX7, 24hr battery, stereo pairing', 7999, 'USD', 60, 'SPK-BT-001', NOW(), NOW()),
  ('a0000001-0000-0000-0000-000000000007', 'Noise Cancelling Headphones', 'ANC, 30hr battery, foldable, multipoint', 24999, 'USD', 35, 'HP-ANC-001', NOW(), NOW()),
  ('a0000001-0000-0000-0000-000000000008', 'Laptop Stand', 'Aluminum, adjustable height, ventilated', 3499, 'USD', 100, 'STD-LAP-001', NOW(), NOW()),
  ('a0000001-0000-0000-0000-000000000009', 'Desk Lamp LED', 'Touch dimmer, 5 color temps, USB charging port', 2999, 'USD', 80, 'LMP-LED-001', NOW(), NOW()),
  ('a0000001-0000-0000-0000-000000000010', 'Wireless Charger Pad', 'Qi 15W fast charge, LED indicator, slim design', 1999, 'USD', 150, 'CHG-QI-001', NOW(), NOW()),
  ('a0000001-0000-0000-0000-000000000011', 'Smart Watch', 'GPS, heart rate, sleep tracking, 7-day battery', 29999, 'USD', 25, 'WATCH-SM-001', NOW(), NOW()),
  ('a0000001-0000-0000-0000-000000000012', 'Portable SSD 1TB', 'NVMe, 1050MB/s read, USB-C, shock resistant', 8999, 'USD', 45, 'SSD-1TB-001', NOW(), NOW()),
  ('a0000001-0000-0000-0000-000000000013', 'Ethernet Cable 3m', 'Cat8, 40Gbps, shielded, gold-plated connectors', 1299, 'USD', 200, 'CBL-ETH-001', NOW(), NOW()),
  ('a0000001-0000-0000-0000-000000000014', 'Mouse Pad XL', '900x400mm, stitched edges, non-slip rubber base', 1999, 'USD', 130, 'PAD-XL-001', NOW(), NOW()),
  ('a0000001-0000-0000-0000-000000000015', 'USB Microphone', 'Cardioid condenser, mute button, gain control', 6999, 'USD', 40, 'MIC-USB-001', NOW(), NOW()),
  ('a0000001-0000-0000-0000-000000000016', 'Graphics Tablet', '10x6 inch, 8192 pressure levels, wireless pen', 7999, 'USD', 30, 'TAB-GFX-001', NOW(), NOW()),
  ('a0000001-0000-0000-0000-000000000017', 'Cable Management Kit', '120 pieces: clips, ties, sleeves, labels', 1499, 'USD', 180, 'CBL-KIT-001', NOW(), NOW()),
  ('a0000001-0000-0000-0000-000000000018', 'Monitor Arm', 'Single arm, gas spring, VESA 75/100, clamp mount', 4499, 'USD', 55, 'ARM-MON-001', NOW(), NOW()),
  ('a0000001-0000-0000-0000-000000000019', 'Keyboard Wrist Rest', 'Memory foam, ergonomic, non-slip, washable cover', 1999, 'USD', 95, 'RST-WRT-001', NOW(), NOW()),
  ('a0000001-0000-0000-0000-000000000020', 'Privacy Screen Filter 27"', 'Anti-glare, anti-blue light, easy install', 3999, 'USD', 40, 'FLT-PRV-001', NOW(), NOW())
ON CONFLICT (sku) DO NOTHING;
