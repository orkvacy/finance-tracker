-- Multi-currency. Freelance dibayar USD/AUD/KRW, pengeluaran tetap IDR.
--
-- Keputusan pokok:
--
--   1. IDR adalah mata uang PELAPORAN. Seluruh ringkasan, anggaran, jatah
--      harian, dan grafik laju memakai amount_idr. Satu angka tidak mungkin
--      dihitung dari empat mata uang sekaligus.
--
--   2. rate_to_idr DIBEKUKAN saat transaksi dibuat, tidak pernah dihitung ulang.
--      Kalau laporan lama ikut berubah mengikuti kurs hari ini, pengeluaran
--      bulan September akan berbeda setiap kali dibuka — dan laporan yang
--      berubah sendiri tidak bisa dipercaya.
--
--   3. amount_idr INTEGER adalah satu-satunya sumber untuk semua perhitungan.
--      rate_to_idr disimpan sebagai REAL hanya untuk audit dan tampilan, jadi
--      tidak ada aritmetika pecahan yang pernah masuk ke agregasi.
--
--   4. amount kini dalam MINOR UNIT mata uangnya: sen untuk USD/AUD,
--      satuan utuh untuk IDR dan KRW yang tidak berdesimal.
--      Baris lama sudah benar apa adanya karena IDR berdesimal nol.

CREATE TABLE currencies (
  code       TEXT    PRIMARY KEY,
  decimals   INTEGER NOT NULL,
  symbol     TEXT    NOT NULL,
  -- Kurs terakhir yang diketahui, dipakai HANYA untuk menaksir nilai saldo
  -- berjalan dalam IDR. Tidak pernah dipakai untuk menghitung ulang transaksi
  -- yang sudah tercatat.
  rate_to_idr REAL   NOT NULL DEFAULT 1,
  sort_order INTEGER NOT NULL DEFAULT 0
);

INSERT INTO currencies (code, decimals, symbol, rate_to_idr, sort_order) VALUES
  ('IDR', 0, 'Rp',     1, 10),
  ('USD', 2, '$',      1, 20),
  ('AUD', 2, 'A$',     1, 30),
  ('KRW', 0, '₩',      1, 40);

ALTER TABLE transactions ADD COLUMN currency    TEXT    NOT NULL DEFAULT 'IDR';
ALTER TABLE transactions ADD COLUMN rate_to_idr REAL    NOT NULL DEFAULT 1;
ALTER TABLE transactions ADD COLUMN amount_idr  INTEGER NOT NULL DEFAULT 0;

-- Baris yang sudah ada seluruhnya IDR, jadi ekuivalennya sama persis.
UPDATE transactions SET amount_idr = amount WHERE amount_idr = 0;

CREATE INDEX idx_tx_currency ON transactions(currency) WHERE currency <> 'IDR';
