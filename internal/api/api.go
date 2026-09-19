// Package api memuat router HTTP dan handler-nya.
package api

import (
	"encoding/csv"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"github.com/orkvacy/finance-track/internal/store"
)

type Server struct {
	store *store.Store
	web   fs.FS
	tz    *time.Location
	log   *slog.Logger
}

func New(s *store.Store, web fs.FS, tz *time.Location, log *slog.Logger) http.Handler {
	srv := &Server{store: s, web: web, tz: tz, log: log}

	r := chi.NewRouter()
	r.Use(middleware.Recoverer)
	r.Use(middleware.RealIP)

	r.Get("/healthz", srv.health)

	r.Route("/api", func(r chi.Router) {
		r.Get("/accounts", srv.listAccounts)
		r.Post("/accounts", srv.createAccount)

		r.Get("/categories", srv.listCategories)
		r.Post("/categories", srv.createCategory)

		r.Get("/currencies", srv.listCurrencies)

		r.Get("/transactions", srv.listTransactions)
		r.Post("/transactions", srv.createTransaction)
		r.Put("/transactions/{id}", srv.updateTransaction)
		r.Delete("/transactions/{id}", srv.deleteTransaction)

		r.Get("/summary", srv.summary)
		r.Get("/export.csv", srv.exportCSV)

		r.NotFound(func(w http.ResponseWriter, r *http.Request) {
			fail(w, http.StatusNotFound, "endpoint tidak ada")
		})
	})

	r.NotFound(srv.serveSPA)
	return r
}

// ---------------------------------------------------------------- bantu

func ok(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	if v != nil {
		json.NewEncoder(w).Encode(v)
	}
}

func fail(w http.ResponseWriter, code int, msg string) {
	ok(w, code, map[string]string{"error": msg})
}

func decode(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		fail(w, http.StatusBadRequest, "JSON tidak valid: "+err.Error())
		return false
	}
	return true
}

func (s *Server) oops(w http.ResponseWriter, err error, msg string) {
	switch {
	case errors.Is(err, store.ErrNotFound):
		fail(w, http.StatusNotFound, err.Error())
	case errors.Is(err, store.ErrConflict):
		fail(w, http.StatusConflict, err.Error())
	default:
		s.log.Error(msg, "err", err)
		fail(w, http.StatusInternalServerError, "kesalahan internal")
	}
}

func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	// Health check Coolify menyentuh database sungguhan, bukan sekadar
	// membuktikan prosesnya hidup — container yang berjalan tanpa volume
	// ter-mount harus dianggap tidak sehat (§13.2).
	if _, err := s.store.ListCategories(r.Context()); err != nil {
		s.log.Error("health check gagal", "err", err)
		fail(w, http.StatusServiceUnavailable, "database tidak bisa dibaca")
		return
	}
	ok(w, http.StatusOK, map[string]string{"status": "ok"})
}

// ---------------------------------------------------------------- akun

func (s *Server) listAccounts(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.ListAccounts(r.Context())
	if err != nil {
		s.oops(w, err, "list accounts")
		return
	}
	ok(w, http.StatusOK, list)
}

var accountTypes = map[string]bool{"bank": true, "ewallet": true, "cash": true}

func (s *Server) createAccount(w http.ResponseWriter, r *http.Request) {
	var a store.Account
	if !decode(w, r, &a) {
		return
	}
	if strings.TrimSpace(a.Name) == "" {
		fail(w, http.StatusBadRequest, "nama akun wajib diisi")
		return
	}
	if !accountTypes[a.Type] {
		fail(w, http.StatusBadRequest, "type harus bank, ewallet, atau cash")
		return
	}
	created, err := s.store.CreateAccount(r.Context(), a)
	if err != nil {
		s.oops(w, err, "create account")
		return
	}
	ok(w, http.StatusCreated, created)
}

// ---------------------------------------------------------------- kategori

func (s *Server) listCategories(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.ListCategories(r.Context())
	if err != nil {
		s.oops(w, err, "list categories")
		return
	}
	ok(w, http.StatusOK, list)
}

func (s *Server) createCategory(w http.ResponseWriter, r *http.Request) {
	var c store.Category
	if !decode(w, r, &c) {
		return
	}
	if strings.TrimSpace(c.Name) == "" {
		fail(w, http.StatusBadRequest, "nama kategori wajib diisi")
		return
	}
	if c.Kind != "income" && c.Kind != "expense" {
		fail(w, http.StatusBadRequest, "kind harus income atau expense")
		return
	}
	created, err := s.store.CreateCategory(r.Context(), c)
	if err != nil {
		s.oops(w, err, "create category")
		return
	}
	ok(w, http.StatusCreated, created)
}

func (s *Server) listCurrencies(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.ListCurrencies(r.Context())
	if err != nil {
		s.oops(w, err, "list currencies")
		return
	}
	ok(w, http.StatusOK, list)
}

// ---------------------------------------------------------------- transaksi

func (s *Server) listTransactions(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	limit, _ := strconv.Atoi(q.Get("limit"))
	list, err := s.store.ListTransactions(r.Context(), store.TxFilter{
		From:      q.Get("from"),
		To:        q.Get("to"),
		AccountID: q.Get("accountId"),
		Status:    q.Get("status"),
		Limit:     limit,
	})
	if err != nil {
		s.oops(w, err, "list transactions")
		return
	}
	ok(w, http.StatusOK, list)
}

// validateTx memusatkan aturan yang tidak boleh dilanggar siapa pun.
func validateTx(t store.Transaction) string {
	if t.AccountID == "" {
		return "accountId wajib diisi"
	}
	if t.Amount <= 0 {
		// Nominal selalu positif; arah dibawa field direction (README §9).
		return "amount harus lebih dari 0"
	}
	if t.Direction != "in" && t.Direction != "out" {
		return "direction harus in atau out"
	}
	if t.Status != "" && t.Status != "draft" && t.Status != "confirmed" {
		return "status harus draft atau confirmed"
	}
	// Aplikasi tidak pernah mengarang kurs. Untuk mata uang asing, pemanggil
	// harus menyebut rupiah yang benar-benar diterima (amountIdr) atau kursnya.
	if t.Currency != "" && t.Currency != "IDR" && t.AmountIDR <= 0 && t.RateToIDR <= 0 {
		return "transaksi mata uang asing butuh amountIdr (rupiah yang benar-benar diterima) atau rate"
	}
	if t.RateToIDR < 0 {
		return "rate tidak boleh negatif"
	}
	return ""
}

func (s *Server) createTransaction(w http.ResponseWriter, r *http.Request) {
	var t store.Transaction
	if !decode(w, r, &t) {
		return
	}
	if msg := validateTx(t); msg != "" {
		fail(w, http.StatusBadRequest, msg)
		return
	}
	created, err := s.store.CreateTransaction(r.Context(), t)
	if err != nil {
		s.oops(w, err, "create transaction")
		return
	}
	ok(w, http.StatusCreated, created)
}

func (s *Server) updateTransaction(w http.ResponseWriter, r *http.Request) {
	var t store.Transaction
	if !decode(w, r, &t) {
		return
	}
	if msg := validateTx(t); msg != "" {
		fail(w, http.StatusBadRequest, msg)
		return
	}
	if err := s.store.UpdateTransaction(r.Context(), chi.URLParam(r, "id"), t); err != nil {
		s.oops(w, err, "update transaction")
		return
	}
	ok(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) deleteTransaction(w http.ResponseWriter, r *http.Request) {
	if err := s.store.DeleteTransaction(r.Context(), chi.URLParam(r, "id")); err != nil {
		s.oops(w, err, "delete transaction")
		return
	}
	ok(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) summary(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	from, to := q.Get("from"), q.Get("to")
	if from == "" || to == "" {
		// Default: bulan berjalan menurut waktu lokal, bukan UTC.
		now := time.Now().In(s.tz)
		first := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, s.tz)
		from = first.Format("2006-01-02")
		to = first.AddDate(0, 1, -1).Format("2006-01-02")
	}
	sum, err := s.store.Summary(r.Context(), from, to)
	if err != nil {
		s.oops(w, err, "summary")
		return
	}
	ok(w, http.StatusOK, sum)
}

// ---------------------------------------------------------------- SPA

func (s *Server) serveSPA(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		fail(w, http.StatusMethodNotAllowed, "metode tidak didukung")
		return
	}

	name := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
	if name == "" || name == "." {
		name = "index.html"
	}

	if f, err := s.web.Open(name); err == nil {
		defer f.Close()
		if st, err := f.Stat(); err == nil && !st.IsDir() {
			switch {
			case strings.HasPrefix(name, "assets/"):
				// Nama berkasnya ber-hash, jadi isinya tidak akan pernah berubah.
				w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")

			case name == "sw.js" || name == "registerSW.js" || strings.HasPrefix(name, "workbox-"):
				// Service worker WAJIB tidak di-cache. Kalau browser menyajikan
				// sw.js lama dari cache, pengguna terkunci pada versi aplikasi
				// lama dan tidak pernah menerima pembaruan - gejalanya "sudah
				// deploy tapi HP masih menampilkan yang lama".
				w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

			case strings.HasSuffix(name, ".webmanifest"):
				// Go tidak mengenali ekstensi ini, dan ServeContent akan
				// menebaknya sebagai teks biasa - sebagian browser lalu menolak
				// manifest-nya dan opsi pasang tidak muncul sama sekali.
				w.Header().Set("Content-Type", "application/manifest+json")
			}
			serve(w, r, name, st.ModTime(), f)
			return
		}
	}

	// Rute SPA apa pun jatuh ke index.html supaya reload di /laporan tetap jalan.
	index, err := s.web.Open("index.html")
	if err != nil {
		http.Error(w, "Frontend belum di-build. Jalankan: npm --prefix web ci && npm --prefix web run build",
			http.StatusServiceUnavailable)
		return
	}
	defer index.Close()
	st, _ := index.Stat()
	w.Header().Set("Cache-Control", "no-cache")
	serve(w, r, "index.html", st.ModTime(), index)
}

// serve memakai ServeContent bila berkasnya bisa di-seek (embed.FS bisa),
// supaya Range request dan ETag ditangani sendiri oleh net/http. Assertion-nya
// diperiksa, bukan dipaksa, agar implementasi fs.FS lain tidak memicu panic.
func serve(w http.ResponseWriter, r *http.Request, name string, mod time.Time, f fs.File) {
	if rs, ok := f.(io.ReadSeeker); ok {
		http.ServeContent(w, r, name, mod, rs)
		return
	}
	if ctype := mime.TypeByExtension(path.Ext(name)); ctype != "" {
		w.Header().Set("Content-Type", ctype)
	}
	io.Copy(w, f)
}

// ---------------------------------------------------------------- ekspor

// exportCSV menghasilkan berkas yang bisa langsung dibuka Excel/Sheets.
//
// Dua keputusan format yang disengaja:
//   - BOM UTF-8 di awal, supaya Excel tidak merusak karakter non-ASCII
//   - kolom amountIdr terpisah dari amount, supaya penjumlahan di spreadsheet
//     tetap benar walau barisnya tercampur beberapa mata uang
func (s *Server) exportCSV(w http.ResponseWriter, r *http.Request) {
	list, err := s.store.ExportAll(r.Context())
	if err != nil {
		s.oops(w, err, "export csv")
		return
	}

	name := "finance-track-" + time.Now().In(s.tz).Format("2006-01-02") + ".csv"
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	w.Write([]byte{0xEF, 0xBB, 0xBF}) // BOM untuk Excel

	cw := csv.NewWriter(w)
	defer cw.Flush()

	cw.Write([]string{
		"tanggal", "jam", "akun", "kategori", "merchant", "catatan",
		"arah", "mata_uang", "nominal", "kurs_ke_idr", "nominal_idr",
		"status", "sumber", "transfer", "id",
	})
	for _, x := range list {
		cw.Write([]string{
			x.Date, x.Time, x.Account, x.Category, x.Merchant, x.Note,
			x.Direction, x.Currency,
			strconv.FormatInt(x.Amount, 10),
			strconv.FormatFloat(x.Rate, 'f', -1, 64),
			strconv.FormatInt(x.AmountIDR, 10),
			x.Status, x.Source, strconv.FormatBool(x.Transfer), x.ID,
		})
	}
}
