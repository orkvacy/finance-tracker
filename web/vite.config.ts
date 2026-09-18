import { defineConfig } from 'vite'
import preact from '@preact/preset-vite'

// Preact, bukan React — keputusan berdasarkan pengukuran, bukan selera:
// React 19 saja sudah 72 KB gzip untuk aplikasi yang praktis kosong, sudah
// melewati anggaran NFR-11 (60 KB) sebelum satu fitur pun ditulis.
//
// Preset ini meng-alias react/react-dom ke preact/compat, jadi kode tetap
// ditulis dengan API React dan impor 'react' seperti biasa. Untuk kembali ke
// React asli: ganti paket, tukar plugin ini, hapus "paths" di tsconfig.
export default defineConfig({
  plugins: [preact()],
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    // Dipantau terhadap NFR-11 (bundle awal <= 60 KB gzip). Kalau peringatan ini
    // muncul, ukur dulu sebelum menambah dependensi baru.
    chunkSizeWarningLimit: 200,
  },
  server: {
    // host: true membuat dev server mendengarkan di semua antarmuka, supaya
    // iPhone di Wi-Fi yang sama bisa membukanya. Tanpa ini Vite hanya terikat
    // ke localhost dan HP tidak akan bisa konek.
    host: true,
    // Saat `npm run dev`, Vite di :5173 dan Go di :8080. Proxy ini membuat
    // frontend memanggil /api dengan jalur yang sama persis seperti di produksi,
    // sehingga tidak ada cabang kode khusus development.
    proxy: {
      '/api': 'http://localhost:8080',
      '/healthz': 'http://localhost:8080',
    },
  },
})
