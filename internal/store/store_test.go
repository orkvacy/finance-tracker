package store_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/orkvacy/finance-track/internal/db"
	"github.com/orkvacy/finance-track/internal/store"
)

// fresh membuat database kosong di direktori sementara dan menjalankan seluruh
// migrasi. Setiap test berangkat dari nol, sehingga migrasi ikut teruji di
// setiap kali `go test` dijalankan.
func fresh(t *testing.T) *store.Store {
	t.Helper()
	d, err := db.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("membuka database: %v", err)
	}
	t.Cleanup(func() { d.Close() })

	if err := d.Migrate(context.Background()); err != nil {
		t.Fatalf("migrasi: %v", err)
	}
	tz, err := time.LoadLocation("Asia/Makassar")
	if err != nil {
		t.Fatalf("zona waktu: %v", err)
	}
	return store.New(d, tz)
}

func TestSaldoIkutTransaksi(t *testing.T) {
	ctx := context.Background()
	s := fresh(t)

	acc, err := s.CreateAccount(ctx, store.Account{Name: "BCA", Type: "bank", OpeningBalance: 500_000})
	if err != nil {
		t.Fatalf("membuat akun: %v", err)
	}

	for _, tx := range []store.Transaction{
		{AccountID: acc.ID, Amount: 18_000, Direction: "out"},
		{AccountID: acc.ID, Amount: 32_500, Direction: "out"},
		{AccountID: acc.ID, Amount: 100_000, Direction: "in"},
	} {
		if _, err := s.CreateTransaction(ctx, tx); err != nil {
			t.Fatalf("membuat transaksi: %v", err)
		}
	}

	list, err := s.ListAccounts(ctx)
	if err != nil {
		t.Fatalf("list akun: %v", err)
	}
	const want = 500_000 - 18_000 - 32_500 + 100_000
	if got := list[0].Balance; got != want {
		t.Errorf("saldo = %d, seharusnya %d", got, want)
	}
}

// Draft hasil impor belum boleh menggerakkan saldo sampai dikonfirmasi (FR-2.4).
func TestDraftTidakMenggerakkanSaldo(t *testing.T) {
	ctx := context.Background()
	s := fresh(t)

	acc, _ := s.CreateAccount(ctx, store.Account{Name: "GoPay", Type: "ewallet", OpeningBalance: 100_000})
	if _, err := s.CreateTransaction(ctx, store.Transaction{
		AccountID: acc.ID, Amount: 25_000, Direction: "out", Status: "draft",
	}); err != nil {
		t.Fatalf("membuat draft: %v", err)
	}

	list, _ := s.ListAccounts(ctx)
	if got := list[0].Balance; got != 100_000 {
		t.Errorf("saldo = %d, draft seharusnya tidak mengubah apa pun", got)
	}
}

// FR-2.6: unique index parsial menolak baris kembar, bukan kode aplikasi.
func TestDedupDitolakDatabase(t *testing.T) {
	ctx := context.Background()
	s := fresh(t)

	acc, _ := s.CreateAccount(ctx, store.Account{Name: "BCA", Type: "bank"})
	hash := "bca-17sep-32500-tokopedia"
	tx := store.Transaction{AccountID: acc.ID, Amount: 32_500, Direction: "out", DedupHash: &hash}

	if _, err := s.CreateTransaction(ctx, tx); err != nil {
		t.Fatalf("impor pertama seharusnya berhasil: %v", err)
	}
	_, err := s.CreateTransaction(ctx, tx)
	if err == nil {
		t.Fatal("impor kedua seharusnya ditolak, tapi diterima")
	}
	if !isConflict(err) {
		t.Errorf("galat = %v, seharusnya ErrConflict", err)
	}
}

func isConflict(err error) bool {
	for e := err; e != nil; {
		if e == store.ErrConflict {
			return true
		}
		u, ok := e.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		e = u.Unwrap()
	}
	return false
}

// Batas "hari" ditentukan zona lokal, bukan UTC. Transaksi 23.30 WITA jatuh
// pada 16.30 UTC di hari yang sama — tapi 00.30 WITA sudah hari berikutnya
// meski UTC masih di hari sebelumnya.
func TestTanggalLokalBukanUTC(t *testing.T) {
	ctx := context.Background()
	s := fresh(t)
	acc, _ := s.CreateAccount(ctx, store.Account{Name: "Tunai", Type: "cash"})

	// 2026-09-18T17:30:00Z  =  2026-09-19 01:30 WITA
	tx, err := s.CreateTransaction(ctx, store.Transaction{
		AccountID: acc.ID, Amount: 10_000, Direction: "out",
		OccurredAt: "2026-09-18T17:30:00Z",
	})
	if err != nil {
		t.Fatalf("membuat transaksi: %v", err)
	}
	if tx.OccurredDate != "2026-09-19" {
		t.Errorf("occurredDate = %q, seharusnya 2026-09-19 (WITA)", tx.OccurredDate)
	}
}

// Transfer antar akun sendiri bukan pemasukan maupun pengeluaran.
func TestTransferTidakMasukRingkasan(t *testing.T) {
	ctx := context.Background()
	s := fresh(t)

	bank, _ := s.CreateAccount(ctx, store.Account{Name: "BCA", Type: "bank", OpeningBalance: 500_000})
	wallet, _ := s.CreateAccount(ctx, store.Account{Name: "GoPay", Type: "ewallet"})

	grp := "grp-topup-1"
	when := "2026-09-18T04:00:00Z"
	s.CreateTransaction(ctx, store.Transaction{AccountID: bank.ID, Amount: 50_000, Direction: "out", TransferGroupID: &grp, OccurredAt: when})
	s.CreateTransaction(ctx, store.Transaction{AccountID: wallet.ID, Amount: 50_000, Direction: "in", TransferGroupID: &grp, OccurredAt: when})
	s.CreateTransaction(ctx, store.Transaction{AccountID: wallet.ID, Amount: 18_000, Direction: "out", OccurredAt: when})

	sum, err := s.Summary(ctx, "2026-09-01", "2026-09-30")
	if err != nil {
		t.Fatalf("ringkasan: %v", err)
	}
	if sum.Out != 18_000 {
		t.Errorf("keluar = %d, seharusnya 18000 — top-up tidak boleh dihitung pengeluaran", sum.Out)
	}
	if sum.In != 0 {
		t.Errorf("masuk = %d, seharusnya 0 — top-up tidak boleh dihitung pemasukan", sum.In)
	}

	// Saldo tiap akun tetap harus bergerak.
	accs, _ := s.ListAccounts(ctx)
	byName := map[string]int64{}
	for _, a := range accs {
		byName[a.Name] = a.Balance
	}
	if byName["BCA"] != 450_000 {
		t.Errorf("saldo BCA = %d, seharusnya 450000", byName["BCA"])
	}
	if byName["GoPay"] != 32_000 {
		t.Errorf("saldo GoPay = %d, seharusnya 32000", byName["GoPay"])
	}
}

// ---------------------------------------------------------------- multi-currency

// Kurs paling jujur adalah rupiah yang benar-benar mendarat, karena angka itu
// sudah memotong biaya Wise/PayPal. Bila amountIdr diberikan, ia yang menang
// dan kursnya diturunkan dari sana.
func TestRupiahDiterimaMengalahkanKurs(t *testing.T) {
	ctx := context.Background()
	s := fresh(t)
	acc, _ := s.CreateAccount(ctx, store.Account{Name: "Wise", Type: "ewallet", Currency: "USD"})

	// 450,00 USD masuk; yang benar-benar mendarat Rp 7.200.000
	tx, err := s.CreateTransaction(ctx, store.Transaction{
		AccountID: acc.ID, Amount: 45_000, Direction: "in",
		Currency: "USD", AmountIDR: 7_200_000,
		RateToIDR: 99_999, // sengaja ngawur — harus diabaikan
	})
	if err != nil {
		t.Fatalf("membuat transaksi: %v", err)
	}
	if tx.AmountIDR != 7_200_000 {
		t.Errorf("amountIdr = %d, seharusnya 7200000", tx.AmountIDR)
	}
	if got := tx.RateToIDR; got != 16_000 {
		t.Errorf("rate = %v, seharusnya 16000 (7.200.000 ÷ 450)", got)
	}
}

func TestKursMenurunkanRupiah(t *testing.T) {
	ctx := context.Background()
	s := fresh(t)
	acc, _ := s.CreateAccount(ctx, store.Account{Name: "Wise", Type: "ewallet", Currency: "AUD"})

	tx, err := s.CreateTransaction(ctx, store.Transaction{
		AccountID: acc.ID, Amount: 20_000, Direction: "in", // 200,00 AUD
		Currency: "AUD", RateToIDR: 10_700,
	})
	if err != nil {
		t.Fatalf("membuat transaksi: %v", err)
	}
	if tx.AmountIDR != 2_140_000 {
		t.Errorf("amountIdr = %d, seharusnya 2140000", tx.AmountIDR)
	}
}

// KRW tidak berdesimal: 150000 berarti ₩150.000, bukan ₩1.500,00.
func TestMataUangTanpaDesimal(t *testing.T) {
	ctx := context.Background()
	s := fresh(t)
	acc, _ := s.CreateAccount(ctx, store.Account{Name: "Payoneer", Type: "ewallet", Currency: "KRW"})

	tx, err := s.CreateTransaction(ctx, store.Transaction{
		AccountID: acc.ID, Amount: 150_000, Direction: "in",
		Currency: "KRW", RateToIDR: 11.8,
	})
	if err != nil {
		t.Fatalf("membuat transaksi: %v", err)
	}
	if tx.AmountIDR != 1_770_000 {
		t.Errorf("amountIdr = %d, seharusnya 1770000 (150000 × 11,8)", tx.AmountIDR)
	}
}

// Tanpa amountIdr maupun rate, permintaan harus DITOLAK. Kurs tebakan pada
// catatan keuangan lebih berbahaya daripada galat.
func TestTanpaKursDitolak(t *testing.T) {
	ctx := context.Background()
	s := fresh(t)
	acc, _ := s.CreateAccount(ctx, store.Account{Name: "Wise", Type: "ewallet", Currency: "USD"})

	if _, err := s.CreateTransaction(ctx, store.Transaction{
		AccountID: acc.ID, Amount: 45_000, Direction: "in", Currency: "USD",
	}); err == nil {
		t.Fatal("seharusnya ditolak, tapi diterima dengan kurs karangan")
	}
}

// Inti dari seluruh desain ini: laporan periode lama TIDAK BOLEH berubah saat
// kurs bergerak. Kalau test ini gagal, angka Septembermu akan berbeda setiap
// kali dibuka.
func TestLaporanLamaTidakBerubahSaatKursBergerak(t *testing.T) {
	ctx := context.Background()
	s := fresh(t)
	acc, _ := s.CreateAccount(ctx, store.Account{Name: "Wise", Type: "ewallet", Currency: "USD"})

	// September: 100,00 USD pada kurs 16.000
	if _, err := s.CreateTransaction(ctx, store.Transaction{
		AccountID: acc.ID, Amount: 10_000, Direction: "out",
		Currency: "USD", RateToIDR: 16_000, OccurredAt: "2026-09-10T04:00:00Z",
	}); err != nil {
		t.Fatalf("transaksi september: %v", err)
	}

	before, _ := s.Summary(ctx, "2026-09-01", "2026-09-30")
	if before.Out != 1_600_000 {
		t.Fatalf("keluar september = %d, seharusnya 1600000", before.Out)
	}

	// Oktober: kurs melonjak ke 18.000
	if _, err := s.CreateTransaction(ctx, store.Transaction{
		AccountID: acc.ID, Amount: 10_000, Direction: "out",
		Currency: "USD", RateToIDR: 18_000, OccurredAt: "2026-10-10T04:00:00Z",
	}); err != nil {
		t.Fatalf("transaksi oktober: %v", err)
	}

	after, _ := s.Summary(ctx, "2026-09-01", "2026-09-30")
	if after.Out != before.Out {
		t.Errorf("laporan september berubah dari %d jadi %d setelah kurs bergerak — "+
			"kurs historis tidak dibekukan", before.Out, after.Out)
	}
}

// Ringkasan menjumlahkan ekuivalen rupiah, bukan nominal asli yang tercampur
// mata uang — tanpa ini 100 USD dan 100 IDR akan dijumlahkan jadi 200.
func TestRingkasanMemakaiEkuivalenRupiah(t *testing.T) {
	ctx := context.Background()
	s := fresh(t)
	idr, _ := s.CreateAccount(ctx, store.Account{Name: "BCA", Type: "bank"})
	usd, _ := s.CreateAccount(ctx, store.Account{Name: "Wise", Type: "ewallet", Currency: "USD"})

	when := "2026-09-18T04:00:00Z"
	s.CreateTransaction(ctx, store.Transaction{AccountID: idr.ID, Amount: 50_000, Direction: "out", OccurredAt: when})
	s.CreateTransaction(ctx, store.Transaction{AccountID: usd.ID, Amount: 1_000, Direction: "out",
		Currency: "USD", RateToIDR: 16_000, OccurredAt: when}) // 10,00 USD = 160.000

	sum, _ := s.Summary(ctx, "2026-09-01", "2026-09-30")
	if sum.Out != 210_000 {
		t.Errorf("keluar = %d, seharusnya 210000 (50.000 + 160.000)", sum.Out)
	}
}

// Saldo mata uang berdesimal disimpan dalam minor unit. Tanpa pembagian
// 10^decimals sebelum dikalikan kurs, taksiran rupiahnya meleset 100x lipat.
func TestTaksiranSaldoMenghormatiDesimal(t *testing.T) {
	ctx := context.Background()
	s := fresh(t)
	acc, _ := s.CreateAccount(ctx, store.Account{Name: "Wise", Type: "ewallet", Currency: "USD"})

	// 450,00 USD masuk pada kurs 16.000
	if _, err := s.CreateTransaction(ctx, store.Transaction{
		AccountID: acc.ID, Amount: 45_000, Direction: "in",
		Currency: "USD", RateToIDR: 16_000,
	}); err != nil {
		t.Fatalf("membuat transaksi: %v", err)
	}

	accs, _ := s.ListAccounts(ctx)
	if accs[0].Balance != 45_000 {
		t.Errorf("saldo = %d sen, seharusnya 45000", accs[0].Balance)
	}
	if accs[0].BalanceIDR != 7_200_000 {
		t.Errorf("taksiran = %d, seharusnya 7200000 (450 USD x 16.000), bukan 720 juta",
			accs[0].BalanceIDR)
	}
}
