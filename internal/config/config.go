// Package config membaca seluruh pengaturan dari environment.
package config

import (
	"fmt"
	"os"
	"time"
)

type Config struct {
	Addr          string
	DBPath        string
	AttachmentDir string
	TZ            *time.Location
}

// Load membaca environment. Semua jalur berkas WAJIB berasal dari sini —
// jangan pernah menulis jalur relatif terhadap working directory, karena
// container Coolify diganti total setiap redeploy dan apa pun di luar volume
// persisten akan hilang bersamanya (§13.2).
//
// Default di bawah ini dibuat nyaman untuk development lokal; Dockerfile
// menimpanya ke /data supaya default di dalam container selalu benar.
func Load() (Config, error) {
	c := Config{
		Addr:          env("ADDR", ":8080"),
		DBPath:        env("DB_PATH", "data/app.db"),
		AttachmentDir: env("ATTACHMENT_DIR", "data/attachments"),
	}

	name := env("APP_TZ", "Asia/Makassar")
	tz, err := time.LoadLocation(name)
	if err != nil {
		// Kalau zona tidak bisa di-resolve, Go diam-diam jatuh ke UTC dan
		// transaksi jam 00.30 WITA tercatat di hari sebelumnya. Lebih baik
		// gagal keras di startup daripada salah tanggal tanpa ketahuan.
		return c, fmt.Errorf("zona waktu %q tidak dikenali (image butuh tzdata): %w", name, err)
	}
	c.TZ = tz
	return c, nil
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
