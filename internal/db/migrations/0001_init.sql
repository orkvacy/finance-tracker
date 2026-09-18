-- Skema awal. Lihat README §9 untuk alasan di balik keputusan modelnya.
--
-- Tiga hal yang sengaja:
--   1. amount INTEGER rupiah + direction terpisah — tidak pernah float,
--      dan tidak pernah bergantung pada tanda negatif.
--   2. transfer diikat transfer_group_id, bukan tipe transaksi khusus,
--      supaya laporan cukup memfilter transfer_group_id IS NULL.
--   3. occurred_at disimpan UTC, occurred_date disimpan sebagai tanggal lokal
--      (WITA). Batas "hari" sangat menentukan di aplikasi keuangan, dan
--      menghitungnya ulang di setiap query itu mahal sekaligus rawan salah.

CREATE TABLE accounts (
  id              TEXT    PRIMARY KEY,
  name            TEXT    NOT NULL,
  type            TEXT    NOT NULL CHECK (type IN ('bank','ewallet','cash')),
  opening_balance INTEGER NOT NULL DEFAULT 0,
  currency        TEXT    NOT NULL DEFAULT 'IDR',
  is_archived     INTEGER NOT NULL DEFAULT 0,
  sort_order      INTEGER NOT NULL DEFAULT 0,
  created_at      TEXT    NOT NULL,
  updated_at      TEXT    NOT NULL
);

CREATE TABLE categories (
  id         TEXT    PRIMARY KEY,
  parent_id  TEXT    REFERENCES categories(id),
  name       TEXT    NOT NULL,
  kind       TEXT    NOT NULL CHECK (kind IN ('income','expense')),
  icon       TEXT    NOT NULL DEFAULT '',
  sort_order INTEGER NOT NULL DEFAULT 0,
  created_at TEXT    NOT NULL,
  updated_at TEXT    NOT NULL
);

CREATE TABLE transactions (
  id                TEXT    PRIMARY KEY,
  account_id        TEXT    NOT NULL REFERENCES accounts(id),
  category_id       TEXT    REFERENCES categories(id),
  amount            INTEGER NOT NULL CHECK (amount > 0),
  direction         TEXT    NOT NULL CHECK (direction IN ('in','out')),
  occurred_at       TEXT    NOT NULL,
  occurred_date     TEXT    NOT NULL,
  note              TEXT    NOT NULL DEFAULT '',
  merchant          TEXT    NOT NULL DEFAULT '',
  source            TEXT    NOT NULL DEFAULT 'manual'
                            CHECK (source IN ('manual','text_import','ocr','recurring')),
  status            TEXT    NOT NULL DEFAULT 'confirmed'
                            CHECK (status IN ('draft','confirmed')),
  transfer_group_id TEXT,
  attachment_id     TEXT,
  dedup_hash        TEXT,
  created_at        TEXT    NOT NULL,
  updated_at        TEXT    NOT NULL,
  deleted_at        TEXT
);

CREATE INDEX idx_tx_date    ON transactions(occurred_date DESC, id DESC) WHERE deleted_at IS NULL;
CREATE INDEX idx_tx_account ON transactions(account_id)                  WHERE deleted_at IS NULL;
CREATE INDEX idx_tx_group   ON transactions(transfer_group_id)           WHERE transfer_group_id IS NOT NULL;

-- FR-2.6 deduplikasi ditegakkan di level database, bukan di kode aplikasi.
-- Impor yang periodenya tumpang-tindih akan ditolak SQLite, bukan diandalkan
-- pada pengecekan yang bisa terlewat.
CREATE UNIQUE INDEX idx_tx_dedup ON transactions(dedup_hash)
  WHERE dedup_hash IS NOT NULL AND deleted_at IS NULL;
