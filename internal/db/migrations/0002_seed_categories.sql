-- Kategori awal (FR-3.3). ID sengaja dibuat stabil dan terbaca supaya
-- migrasi ini idempoten dan aman dijalankan ulang di mesin mana pun.

INSERT INTO categories (id, parent_id, name, kind, icon, sort_order, created_at, updated_at) VALUES
  ('cat_makan',     NULL, 'Makan & Minum',   'expense', 'kopi',      10, datetime('now'), datetime('now')),
  ('cat_transport', NULL, 'Transport',       'expense', 'transport', 20, datetime('now'), datetime('now')),
  ('cat_kos',       NULL, 'Kos',             'expense', 'rumah',     30, datetime('now'), datetime('now')),
  ('cat_kuliah',    NULL, 'Kuliah',          'expense', 'kuliah',    40, datetime('now'), datetime('now')),
  ('cat_pulsa',     NULL, 'Pulsa & Internet','expense', 'pulsa',     50, datetime('now'), datetime('now')),
  ('cat_langganan', NULL, 'Langganan',       'expense', 'layar',     60, datetime('now'), datetime('now')),
  ('cat_belanja',   NULL, 'Belanja',         'expense', 'belanja',   70, datetime('now'), datetime('now')),
  ('cat_kesehatan', NULL, 'Kesehatan',       'expense', 'sehat',     80, datetime('now'), datetime('now')),
  ('cat_sosial',    NULL, 'Sosial',          'expense', 'orang',     90, datetime('now'), datetime('now')),
  ('cat_cicilan',   NULL, 'Cicilan',         'expense', 'kartu',    100, datetime('now'), datetime('now')),
  ('cat_lain',      NULL, 'Lain-lain',       'expense', 'titik',    110, datetime('now'), datetime('now')),

  ('cat_kiriman',   NULL, 'Kiriman',         'income',  'masuk',     10, datetime('now'), datetime('now')),
  ('cat_beasiswa',  NULL, 'Beasiswa',        'income',  'masuk',     20, datetime('now'), datetime('now')),
  ('cat_freelance', NULL, 'Freelance',       'income',  'masuk',     30, datetime('now'), datetime('now')),
  ('cat_refund',    NULL, 'Refund',          'income',  'masuk',     40, datetime('now'), datetime('now'));
