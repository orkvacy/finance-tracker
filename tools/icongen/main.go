// Command icongen membuat ikon PWA.
//
// Memakai image/png dari stdlib, jadi tidak ada dependensi pengolah gambar
// yang perlu ditambahkan hanya untuk empat berkas yang jarang berubah.
//
//	go run ./tools/icongen
//
// Marka ikonnya mengacu pada elemen khas aplikasi ini: batang jajan harian
// dengan garis jatah melintang, dan satu batang yang melewatinya.
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

var (
	biru  = color.NRGBA{0x2A, 0x5B, 0xD7, 0xFF} // --accent
	putih = color.NRGBA{0xFF, 0xFF, 0xFF, 0xFF}
	lewat = color.NRGBA{0xF2, 0x8B, 0x7D, 0xFF} // --over versi gelap
)

// ss adalah faktor supersampling. Render pada kelipatan ini lalu dirata-rata
// saat diperkecil - itu yang memberi tepi halus tanpa pustaka gambar.
const ss = 4

func main() {
	dir := filepath.Join("web", "public")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		panic(err)
	}
	for _, t := range []struct {
		nama   string
		ukuran int
	}{
		{"icon-192.png", 192},
		{"icon-512.png", 512},
		{"apple-touch-icon.png", 180}, // iOS membulatkan sendiri; kirim persegi penuh
		{"favicon-32.png", 32},
	} {
		if err := tulis(filepath.Join(dir, t.nama), t.ukuran); err != nil {
			panic(err)
		}
		fmt.Printf("  %-22s %dx%d\n", t.nama, t.ukuran, t.ukuran)
	}
}

func tulis(path string, size int) error {
	besar := gambar(size * ss)
	kecil := perkecil(besar, size)

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, kecil)
}

func gambar(S int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, S, S))
	s := float64(S)

	// Latar penuh tanpa sudut membulat: iOS memasang maskernya sendiri, dan
	// Android memotong sesuai bentuk peluncurnya.
	for y := 0; y < S; y++ {
		for x := 0; x < S; x++ {
			img.SetNRGBA(x, y, biru)
		}
	}

	// Isi ikon dijaga dalam lingkaran aman maskable (80% dari sisi), supaya
	// tidak terpotong di peluncur Android yang membulatkan agresif.
	// Garis jatah harus memotong TUBUH kelompok batang, bukan melayang di
	// atasnya - kalau semua batang lebih pendek dari garis, ikonnya terbaca
	// sebagai meja berkaki, bukan grafik.
	const (
		kiri      = 0.20
		kanan     = 0.80
		dasar     = 0.80
		garisY    = 0.44
		jmlBatang = 4
	)
	lebar := (kanan - kiri) / (jmlBatang + float64(jmlBatang-1)*0.42)
	jarak := lebar * 0.42

	tinggi := []float64{0.26, 0.46, 0.19, 0.36}
	warna := []color.NRGBA{putih, lewat, putih, putih}

	for i := 0; i < jmlBatang; i++ {
		x0 := (kiri + float64(i)*(lebar+jarak)) * s
		x1 := x0 + lebar*s
		y1 := dasar * s
		y0 := y1 - tinggi[i]*s
		batang(img, x0, y0, x1, y1, lebar*s/2, warna[i])
	}

	// Garis jatah harian: melintang penuh, tipis, setengah transparan.
	gy := garisY * s
	tebal := math.Max(1, s*0.022)
	for y := int(gy - tebal/2); y < int(gy+tebal/2); y++ {
		for x := int(kiri * s); x < int(kanan*s); x++ {
			campur(img, x, y, color.NRGBA{0xFF, 0xFF, 0xFF, 0xB0})
		}
	}
	return img
}

// batang menggambar persegi panjang dengan ujung atas membulat.
func batang(img *image.NRGBA, x0, y0, x1, y1, r float64, c color.NRGBA) {
	for y := int(y0); y <= int(y1); y++ {
		for x := int(x0); x <= int(x1); x++ {
			fy := float64(y)
			if fy < y0+r {
				// Di area ujung atas, hanya piksel di dalam lingkaran yang diisi.
				cx := (x0 + x1) / 2
				cy := y0 + r
				if math.Hypot(float64(x)-cx, fy-cy) > r {
					continue
				}
			}
			campur(img, x, y, c)
		}
	}
}

func campur(img *image.NRGBA, x, y int, c color.NRGBA) {
	if !(image.Point{x, y}).In(img.Bounds()) {
		return
	}
	if c.A == 0xFF {
		img.SetNRGBA(x, y, c)
		return
	}
	d := img.NRGBAAt(x, y)
	a := float64(c.A) / 255
	img.SetNRGBA(x, y, color.NRGBA{
		uint8(float64(c.R)*a + float64(d.R)*(1-a)),
		uint8(float64(c.G)*a + float64(d.G)*(1-a)),
		uint8(float64(c.B)*a + float64(d.B)*(1-a)),
		0xFF,
	})
}

// perkecil merata-ratakan blok ss x ss menjadi satu piksel - antialias
// sederhana tanpa pustaka tambahan.
func perkecil(src *image.NRGBA, size int) *image.NRGBA {
	dst := image.NewNRGBA(image.Rect(0, 0, size, size))
	n := float64(ss * ss)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			var r, g, b float64
			for dy := 0; dy < ss; dy++ {
				for dx := 0; dx < ss; dx++ {
					p := src.NRGBAAt(x*ss+dx, y*ss+dy)
					r += float64(p.R)
					g += float64(p.G)
					b += float64(p.B)
				}
			}
			dst.SetNRGBA(x, y, color.NRGBA{uint8(r / n), uint8(g / n), uint8(b / n), 0xFF})
		}
	}
	return dst
}
