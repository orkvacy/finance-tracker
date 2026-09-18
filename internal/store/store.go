// Package store memuat seluruh akses database. Lapisan HTTP tidak pernah
// menulis SQL sendiri.
package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/orkvacy/finance-track/internal/db"
	"github.com/orkvacy/finance-track/internal/id"
)

var (
	ErrNotFound = errors.New("tidak ditemukan")
	ErrConflict = errors.New("bentrok")
)

type Store struct {
	db *db.DB
	tz *time.Location
}

func New(d *db.DB, tz *time.Location) *Store { return &Store{db: d, tz: tz} }

// ---------------------------------------------------------------- tipe

type Account struct {
	ID             string `json:"id"`
	Name           string `json:"name"`
	Type           string `json:"type"`
	OpeningBalance int64  `json:"openingBalance"`
	Currency       string `json:"currency"`
	IsArchived     bool   `json:"isArchived"`
	SortOrder      int    `json:"sortOrder"`
	// Balance dalam minor unit mata uang akun; dihitung, tidak disimpan.
	Balance int64 `json:"balance"`
	// BalanceIDR adalah TAKSIRAN memakai kurs terakhir yang diketahui, bukan
	// penjumlahan nilai historis. Nilainya bergerak mengikuti kurs — hanya untuk
	// tampilan total, tidak pernah dipakai sebagai dasar perhitungan.
	BalanceIDR int64 `json:"balanceIdr"`
}

type Category struct {
	ID        string  `json:"id"`
	ParentID  *string `json:"parentId"`
	Name      string  `json:"name"`
	Kind      string  `json:"kind"`
	Icon      string  `json:"icon"`
	SortOrder int     `json:"sortOrder"`
}

type Transaction struct {
	ID              string  `json:"id"`
	AccountID       string  `json:"accountId"`
	CategoryID      *string `json:"categoryId"`
	// Amount dalam MINOR UNIT dari Currency di bawah, selalu positif.
	Amount          int64   `json:"amount"`
	Direction       string  `json:"direction"`
	OccurredAt      string  `json:"occurredAt"`   // RFC3339 UTC
	OccurredDate    string  `json:"occurredDate"` // YYYY-MM-DD lokal
	Note            string  `json:"note"`
	Merchant        string  `json:"merchant"`
	Source          string  `json:"source"`
	Status          string  `json:"status"`
	TransferGroupID *string `json:"transferGroupId"`
	// Currency adalah mata uang asli transaksi; Amount di atas dinyatakan dalam
	// MINOR UNIT mata uang ini (sen untuk USD/AUD, satuan utuh untuk IDR/KRW).
	Currency string `json:"currency"`
	// RateToIDR dibekukan saat transaksi dibuat dan tidak pernah dihitung ulang,
	// supaya laporan periode lama tidak berubah saat kurs bergerak.
	RateToIDR float64 `json:"rateToIdr"`
	// AmountIDR adalah SATU-SATUNYA angka yang dipakai seluruh agregasi:
	// ringkasan, anggaran, jatah harian, grafik laju.
	AmountIDR int64 `json:"amountIdr"`
	// DedupHash menegakkan FR-2.6 lewat unique index parsial di database.
	// Impor mutasi yang periodenya tumpang-tindih ditolak SQLite, bukan
	// bergantung pada pengecekan di kode yang bisa terlewat.
	DedupHash *string `json:"dedupHash"`
}

// ---------------------------------------------------------------- akun

func (s *Store) ListAccounts(ctx context.Context) ([]Account, error) {
	// Saldo = saldo awal + masuk − keluar. Hanya transaksi confirmed yang
	// dihitung: draft hasil impor belum boleh menggerakkan saldo (FR-2.4).
	rows, err := s.db.R.QueryContext(ctx, `
		SELECT a.id, a.name, a.type, a.opening_balance, a.currency,
		       a.is_archived, a.sort_order,
		       a.opening_balance + COALESCE(SUM(
		         CASE WHEN t.direction = 'in' THEN t.amount ELSE -t.amount END), 0),
		       COALESCE(cur.rate_to_idr, 1), COALESCE(cur.decimals, 0)
		FROM accounts a
		LEFT JOIN currencies cur ON cur.code = a.currency
		LEFT JOIN transactions t
		       ON t.account_id = a.id
		      AND t.deleted_at IS NULL
		      AND t.status = 'confirmed'
		GROUP BY a.id
		ORDER BY a.is_archived, a.sort_order, a.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Account{}
	for rows.Next() {
		var a Account
		var rate float64
		var decimals int
		if err := rows.Scan(&a.ID, &a.Name, &a.Type, &a.OpeningBalance, &a.Currency,
			&a.IsArchived, &a.SortOrder, &a.Balance, &rate, &decimals); err != nil {
			return nil, err
		}
		// Balance dalam minor unit, jadi harus dibagi dulu ke satuan utuh
		// sebelum dikalikan kurs. Tanpa pembagian ini, saldo USD tertulis
		// 100x lipat karena sen diperlakukan sebagai dolar.
		a.BalanceIDR = int64(math.Round(float64(a.Balance) / math.Pow10(decimals) * rate))
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *Store) CreateAccount(ctx context.Context, a Account) (Account, error) {
	a.ID = id.New()
	if a.Currency == "" {
		a.Currency = "IDR"
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.W.ExecContext(ctx, `
		INSERT INTO accounts (id, name, type, opening_balance, currency,
		                      is_archived, sort_order, created_at, updated_at)
		VALUES (?,?,?,?,?,0,?,?,?)`,
		a.ID, a.Name, a.Type, a.OpeningBalance, a.Currency, a.SortOrder, now, now)
	if err != nil {
		return a, err
	}
	a.Balance = a.OpeningBalance
	a.BalanceIDR = a.OpeningBalance // akun baru: belum ada kurs yang lebih baik
	return a, nil
}

// ---------------------------------------------------------------- kategori

func (s *Store) ListCategories(ctx context.Context) ([]Category, error) {
	rows, err := s.db.R.QueryContext(ctx, `
		SELECT id, parent_id, name, kind, icon, sort_order
		FROM categories ORDER BY kind DESC, sort_order, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Category{}
	for rows.Next() {
		var c Category
		var parent sql.NullString
		if err := rows.Scan(&c.ID, &parent, &c.Name, &c.Kind, &c.Icon, &c.SortOrder); err != nil {
			return nil, err
		}
		if parent.Valid {
			c.ParentID = &parent.String
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *Store) CreateCategory(ctx context.Context, c Category) (Category, error) {
	c.ID = id.New()
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := s.db.W.ExecContext(ctx, `
		INSERT INTO categories (id, parent_id, name, kind, icon, sort_order, created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?)`,
		c.ID, c.ParentID, c.Name, c.Kind, c.Icon, c.SortOrder, now, now)
	return c, err
}

// ---------------------------------------------------------------- transaksi

type TxFilter struct {
	From, To  string // YYYY-MM-DD lokal, inklusif
	AccountID string
	Status     string
	Limit      int
}

func (s *Store) ListTransactions(ctx context.Context, f TxFilter) ([]Transaction, error) {
	q := strings.Builder{}
	q.WriteString(`
		SELECT id, account_id, category_id, amount, direction, occurred_at,
		       occurred_date, note, merchant, source, status, transfer_group_id,
		       dedup_hash, currency, rate_to_idr, amount_idr
		FROM transactions WHERE deleted_at IS NULL`)
	args := []any{}

	if f.From != "" {
		q.WriteString(" AND occurred_date >= ?")
		args = append(args, f.From)
	}
	if f.To != "" {
		q.WriteString(" AND occurred_date <= ?")
		args = append(args, f.To)
	}
	if f.AccountID != "" {
		q.WriteString(" AND account_id = ?")
		args = append(args, f.AccountID)
	}
	if f.Status != "" {
		q.WriteString(" AND status = ?")
		args = append(args, f.Status)
	}
	if f.Limit <= 0 || f.Limit > 500 {
		f.Limit = 200
	}
	q.WriteString(" ORDER BY occurred_date DESC, id DESC LIMIT ?")
	args = append(args, f.Limit)

	rows, err := s.db.R.QueryContext(ctx, q.String(), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Transaction{}
	for rows.Next() {
		t, err := scanTx(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func scanTx(rows *sql.Rows) (Transaction, error) {
	var t Transaction
	var cat, grp, dedup sql.NullString
	err := rows.Scan(&t.ID, &t.AccountID, &cat, &t.Amount, &t.Direction, &t.OccurredAt,
		&t.OccurredDate, &t.Note, &t.Merchant, &t.Source, &t.Status, &grp, &dedup,
		&t.Currency, &t.RateToIDR, &t.AmountIDR)
	if cat.Valid {
		t.CategoryID = &cat.String
	}
	if grp.Valid {
		t.TransferGroupID = &grp.String
	}
	if dedup.Valid {
		t.DedupHash = &dedup.String
	}
	return t, err
}

func (s *Store) CreateTransaction(ctx context.Context, t Transaction) (Transaction, error) {
	t.ID = id.New()

	at, err := s.normalizeTime(t.OccurredAt)
	if err != nil {
		return t, err
	}
	t.OccurredAt = at.UTC().Format(time.RFC3339)
	t.OccurredDate = at.In(s.tz).Format("2006-01-02")

	if t.Source == "" {
		t.Source = "manual"
	}
	if t.Status == "" {
		t.Status = "confirmed"
	}
	if t.Currency == "" {
		t.Currency = "IDR"
	}

	decimals, err := s.decimals(ctx, t.Currency)
	if err != nil {
		return t, err
	}
	if err := resolveIDR(&t, decimals); err != nil {
		return t, err
	}

	now := time.Now().UTC().Format(time.RFC3339)

	// Satu transaksi database: sisipan dan pembaruan kurs terakhir harus
	// berhasil bersama-sama atau gagal bersama-sama.
	tx, err := s.db.W.BeginTx(ctx, nil)
	if err != nil {
		return t, err
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO transactions (id, account_id, category_id, amount, direction,
		       occurred_at, occurred_date, note, merchant, source, status,
		       transfer_group_id, dedup_hash, currency, rate_to_idr, amount_idr,
		       created_at, updated_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		t.ID, t.AccountID, t.CategoryID, t.Amount, t.Direction, t.OccurredAt,
		t.OccurredDate, t.Note, t.Merchant, t.Source, t.Status, t.TransferGroupID,
		t.DedupHash, t.Currency, t.RateToIDR, t.AmountIDR, now, now)
	if err != nil {
		if strings.Contains(err.Error(), "FOREIGN KEY") {
			return t, fmt.Errorf("%w: akun atau kategori tidak ada", ErrNotFound)
		}
		if strings.Contains(err.Error(), "UNIQUE") {
			return t, fmt.Errorf("%w: transaksi ini sudah pernah masuk", ErrConflict)
		}
		return t, err
	}

	// Kurs terakhir dipakai HANYA untuk menaksir nilai saldo berjalan.
	// Transaksi yang sudah tercatat tidak pernah ikut berubah karenanya.
	if t.Currency != "IDR" {
		if _, err := tx.ExecContext(ctx,
			`UPDATE currencies SET rate_to_idr = ? WHERE code = ?`,
			t.RateToIDR, t.Currency); err != nil {
			return t, err
		}
	}
	return t, tx.Commit()
}

func (s *Store) decimals(ctx context.Context, code string) (int, error) {
	var d int
	err := s.db.R.QueryRowContext(ctx,
		`SELECT decimals FROM currencies WHERE code = ?`, code).Scan(&d)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, fmt.Errorf("%w: mata uang %q belum terdaftar", ErrNotFound, code)
	}
	return d, err
}

// resolveIDR mengisi AmountIDR dan RateToIDR, lalu membekukannya.
//
// Aplikasi TIDAK PERNAH mengarang kurs. Kalau mata uangnya bukan IDR dan
// pemanggil tidak memberi amountIdr maupun rate, permintaan ditolak — kurs
// tebakan pada catatan keuangan lebih berbahaya daripada galat.
//
// Bila keduanya diberikan, amountIdr yang menang dan rate diturunkan darinya:
// jumlah rupiah yang benar-benar mendarat di rekening adalah fakta, sedangkan
// kurs yang diiklankan belum memotong biaya Wise/PayPal.
func resolveIDR(t *Transaction, decimals int) error {
	if t.Currency == "IDR" {
		t.AmountIDR = t.Amount
		t.RateToIDR = 1
		return nil
	}

	units := float64(t.Amount) / math.Pow10(decimals) // mis. 45000 sen → 450.00 USD

	switch {
	case t.AmountIDR > 0:
		t.RateToIDR = float64(t.AmountIDR) / units
	case t.RateToIDR > 0:
		t.AmountIDR = int64(math.Round(units * t.RateToIDR))
	default:
		return fmt.Errorf("transaksi %s butuh amountIdr (rupiah yang benar-benar diterima) atau rate", t.Currency)
	}
	if t.AmountIDR <= 0 {
		return fmt.Errorf("ekuivalen rupiah harus lebih dari 0")
	}
	return nil
}

// normalizeTime menerima RFC3339, atau kosong (= sekarang). Tanggal polos
// "YYYY-MM-DD" ditafsirkan sebagai tengah hari waktu lokal, bukan tengah malam,
// supaya pergeseran zona tidak pernah memindahkannya ke hari sebelahnya.
func (s *Store) normalizeTime(v string) (time.Time, error) {
	if v == "" {
		return time.Now(), nil
	}
	if t, err := time.Parse(time.RFC3339, v); err == nil {
		return t, nil
	}
	if d, err := time.ParseInLocation("2006-01-02", v, s.tz); err == nil {
		return d.Add(12 * time.Hour), nil
	}
	return time.Time{}, fmt.Errorf("format waktu tidak dikenali: %q", v)
}

func (s *Store) UpdateTransaction(ctx context.Context, tid string, t Transaction) error {
	at, err := s.normalizeTime(t.OccurredAt)
	if err != nil {
		return err
	}
	if t.Currency == "" {
		t.Currency = "IDR"
	}

	// Ekuivalen rupiah WAJIB dihitung ulang di sini. Tanpa ini, mengubah nominal
	// transaksi USD akan meninggalkan amount_idr yang basi, dan seluruh laporan
	// diam-diam salah tanpa ada galat yang muncul.
	decimals, err := s.decimals(ctx, t.Currency)
	if err != nil {
		return err
	}
	if err := resolveIDR(&t, decimals); err != nil {
		return err
	}

	res, err := s.db.W.ExecContext(ctx, `
		UPDATE transactions
		   SET account_id = ?, category_id = ?, amount = ?, direction = ?,
		       occurred_at = ?, occurred_date = ?, note = ?, merchant = ?,
		       status = ?, currency = ?, rate_to_idr = ?, amount_idr = ?,
		       updated_at = ?
		 WHERE id = ? AND deleted_at IS NULL`,
		t.AccountID, t.CategoryID, t.Amount, t.Direction,
		at.UTC().Format(time.RFC3339), at.In(s.tz).Format("2006-01-02"),
		t.Note, t.Merchant, t.Status, t.Currency, t.RateToIDR, t.AmountIDR,
		time.Now().UTC().Format(time.RFC3339), tid)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// Currency adalah satu mata uang yang dikenal aplikasi.
type Currency struct {
	Code      string  `json:"code"`
	Decimals  int     `json:"decimals"`
	Symbol    string  `json:"symbol"`
	RateToIDR float64 `json:"rateToIdr"`
}

func (s *Store) ListCurrencies(ctx context.Context) ([]Currency, error) {
	rows, err := s.db.R.QueryContext(ctx,
		`SELECT code, decimals, symbol, rate_to_idr FROM currencies ORDER BY sort_order`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []Currency{}
	for rows.Next() {
		var c Currency
		if err := rows.Scan(&c.Code, &c.Decimals, &c.Symbol, &c.RateToIDR); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// DeleteTransaction menandai terhapus, tidak membuang barisnya — supaya undo
// (FR-1.5) mungkin dilakukan dan riwayat tetap utuh.
func (s *Store) DeleteTransaction(ctx context.Context, tid string) error {
	res, err := s.db.W.ExecContext(ctx,
		`UPDATE transactions SET deleted_at = ?, updated_at = ?
		  WHERE id = ? AND deleted_at IS NULL`,
		time.Now().UTC().Format(time.RFC3339), time.Now().UTC().Format(time.RFC3339), tid)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return ErrNotFound
	}
	return nil
}

// ---------------------------------------------------------------- ringkasan

type Summary struct {
	From       string          `json:"from"`
	To         string          `json:"to"`
	In         int64           `json:"in"`
	Out        int64           `json:"out"`
	Net        int64           `json:"net"`
	ByCategory []CategoryTotal `json:"byCategory"`
}

type CategoryTotal struct {
	CategoryID string `json:"categoryId"`
	Name       string `json:"name"`
	Total      int64  `json:"total"`
}

func (s *Store) Summary(ctx context.Context, from, to string) (Summary, error) {
	sum := Summary{From: from, To: to, ByCategory: []CategoryTotal{}}

	// transfer_group_id IS NULL: pindah dana antar akun sendiri bukan
	// pemasukan maupun pengeluaran (README §9).
	err := s.db.R.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(CASE WHEN direction='in'  THEN amount_idr END), 0),
		       COALESCE(SUM(CASE WHEN direction='out' THEN amount_idr END), 0)
		FROM transactions
		WHERE deleted_at IS NULL AND status = 'confirmed'
		  AND transfer_group_id IS NULL
		  AND occurred_date BETWEEN ? AND ?`, from, to).Scan(&sum.In, &sum.Out)
	if err != nil {
		return sum, err
	}
	sum.Net = sum.In - sum.Out

	rows, err := s.db.R.QueryContext(ctx, `
		SELECT t.category_id, COALESCE(c.name, 'Tanpa kategori'), SUM(t.amount_idr)
		FROM transactions t
		LEFT JOIN categories c ON c.id = t.category_id
		WHERE t.deleted_at IS NULL AND t.status = 'confirmed'
		  AND t.direction = 'out' AND t.transfer_group_id IS NULL
		  AND t.occurred_date BETWEEN ? AND ?
		GROUP BY t.category_id
		ORDER BY SUM(t.amount_idr) DESC`, from, to)
	if err != nil {
		return sum, err
	}
	defer rows.Close()

	for rows.Next() {
		var ct CategoryTotal
		var cid sql.NullString
		if err := rows.Scan(&cid, &ct.Name, &ct.Total); err != nil {
			return sum, err
		}
		ct.CategoryID = cid.String
		sum.ByCategory = append(sum.ByCategory, ct)
	}
	return sum, rows.Err()
}

// ---------------------------------------------------------------- ekspor

// ExportRow adalah satu baris CSV dengan nama akun & kategori sudah di-resolve,
// supaya berkasnya bisa dibaca manusia tanpa perlu tabel lain.
type ExportRow struct {
	Date      string
	Time      string
	Account   string
	Category  string
	Merchant  string
	Note      string
	Direction string
	Currency  string
	Amount    int64
	Rate      float64
	AmountIDR int64
	Status    string
	Source    string
	Transfer  bool
	ID        string
}

// ExportAll mengembalikan seluruh transaksi yang belum dihapus, terlama dahulu.
// FR-5.6 menjadikan ini wajib sejak M1: aplikasi ini tidak boleh menyandera
// datanya sendiri.
func (s *Store) ExportAll(ctx context.Context) ([]ExportRow, error) {
	rows, err := s.db.R.QueryContext(ctx, `
		SELECT t.occurred_date, t.occurred_at,
		       COALESCE(a.name, ''), COALESCE(c.name, ''),
		       t.merchant, t.note, t.direction, t.currency, t.amount,
		       t.rate_to_idr, t.amount_idr, t.status, t.source,
		       t.transfer_group_id IS NOT NULL, t.id
		FROM transactions t
		LEFT JOIN accounts   a ON a.id = t.account_id
		LEFT JOIN categories c ON c.id = t.category_id
		WHERE t.deleted_at IS NULL
		ORDER BY t.occurred_date, t.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []ExportRow{}
	for rows.Next() {
		var r ExportRow
		var at string
		if err := rows.Scan(&r.Date, &at, &r.Account, &r.Category, &r.Merchant,
			&r.Note, &r.Direction, &r.Currency, &r.Amount, &r.Rate, &r.AmountIDR,
			&r.Status, &r.Source, &r.Transfer, &r.ID); err != nil {
			return nil, err
		}
		// Waktu disimpan UTC tapi diekspor dalam jam lokal, karena berkas ini
		// dibaca manusia yang hidup di WITA.
		if ts, err := time.Parse(time.RFC3339, at); err == nil {
			r.Time = ts.In(s.tz).Format("15:04")
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
