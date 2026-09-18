package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	// Menyatukan basis data zona waktu ke dalam binary (~450 KB).
	//
	// config.Load sengaja gagal keras kalau Asia/Makassar tidak bisa
	// di-resolve, karena diam-diam jatuh ke UTC berarti transaksi lewat
	// tengah malam tercatat di hari yang salah. Impor ini membuat binary
	// tidak lagi bergantung pada paket tzdata di base image — deploy ke
	// image minimal apa pun tetap benar.
	_ "time/tzdata"

	"github.com/orkvacy/finance-track/internal/api"
	"github.com/orkvacy/finance-track/internal/config"
	"github.com/orkvacy/finance-track/internal/db"
	"github.com/orkvacy/finance-track/internal/store"
	"github.com/orkvacy/finance-track/web"
)

func main() {
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	if err := run(log); err != nil {
		log.Error("server berhenti", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	abs, _ := filepath.Abs(cfg.DBPath)
	// Jalur database dicatat di setiap startup dengan sengaja: kalau volume
	// persisten lupa di-mount di Coolify, baris inilah yang mengungkapnya
	// sebelum data sungguhan masuk (§13.2).
	log.Info("konfigurasi", "db", abs, "tz", cfg.TZ.String(), "addr", cfg.Addr)

	database, err := db.Open(cfg.DBPath)
	if err != nil {
		return err
	}
	defer database.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := database.Migrate(ctx); err != nil {
		return err
	}
	log.Info("migrasi selesai")

	if err := os.MkdirAll(cfg.AttachmentDir, 0o755); err != nil {
		return err
	}

	webFS, err := web.FS()
	if err != nil {
		return err
	}

	handler := api.New(store.New(database, cfg.TZ), webFS, cfg.TZ, log)

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errc := make(chan error, 1)
	go func() {
		log.Info("mendengarkan", "addr", cfg.Addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errc <- err
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errc:
		return err
	case sig := <-stop:
		// Coolify mengirim SIGTERM saat redeploy. Menutup dengan rapi memberi
		// SQLite kesempatan menuntaskan checkpoint WAL.
		log.Info("sinyal diterima, menutup", "sinyal", sig.String())
	}

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()
	return srv.Shutdown(shutdownCtx)
}
