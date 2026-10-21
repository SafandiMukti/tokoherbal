-- Data contoh produk. Aman dijalankan berulang: hanya mengisi jika tabel masih kosong.
-- Jalankan: docker compose exec postgres psql -U postgres -d shop_db -f /seed/seed.sql

INSERT INTO products (name, description, category, price, stock, image_url)
SELECT * FROM (VALUES
    ('Jamu Kunyit Asam',      'Minuman herbal tradisional dari kunyit dan asam jawa.',        'jamu',     15000, 50, ''),
    ('Jamu Beras Kencur',     'Jamu segar berbahan beras dan kencur.',                        'jamu',     15000, 40, ''),
    ('Jamu Temulawak',        'Jamu temulawak untuk menjaga stamina.',                        'jamu',     18000, 30,  ''),
    ('Madu Hutan 250 ml',     'Madu murni dari hutan, tanpa campuran gula.',                  'madu',     85000, 25, ''),
    ('Madu Kelengkeng 500 ml','Madu kelengkeng dengan rasa lembut.',                          'madu',    120000, 15, ''),
    ('Minyak Kayu Putih 60 ml','Minyak kayu putih asli untuk menghangatkan badan.',           'minyak',   22000, 80, ''),
    ('Minyak Zaitun 100 ml',  'Minyak zaitun untuk perawatan rambut dan kulit.',              'minyak',   45000, 20, ''),
    ('Lulur Tradisional',     'Lulur rempah untuk perawatan kulit.',                          'kosmetik', 35000, 35, ''),
    ('Sabun Herbal Sereh',    'Sabun batang dengan ekstrak sereh.',                           'kosmetik', 12000, 60, ''),
    ('Teh Jahe Merah',        'Teh celup jahe merah isi 20 kantong.',                         'teh',      28000, 45, ''),
    ('Teh Daun Kelor',        'Teh herbal daun kelor isi 20 kantong.',                        'teh',      30000,  5, ''),
    ('Bedak Dingin Beras',    'Bedak dingin dari tepung beras, stok hampir habis.',           'kosmetik', 18000,  3, '')
) AS v(name, description, category, price, stock, image_url)
WHERE NOT EXISTS (SELECT 1 FROM products);
