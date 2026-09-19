import { defineConfig } from 'vite'
import preact from '@preact/preset-vite'
import { VitePWA } from 'vite-plugin-pwa'

// Preact, bukan React — keputusan berdasarkan pengukuran, bukan selera:
// React 19 saja sudah 72 KB gzip untuk aplikasi yang praktis kosong, sudah
// melewati anggaran NFR-11 (60 KB) sebelum satu fitur pun ditulis.
//
// Preset ini meng-alias react/react-dom ke preact/compat, jadi kode tetap
// ditulis dengan API React dan impor 'react' seperti biasa. Untuk kembali ke
// React asli: ganti paket, tukar plugin ini, hapus "paths" di tsconfig.
export default defineConfig({
  plugins: [
    preact(),
    VitePWA({
      registerType: 'autoUpdate',
      includeAssets: ['apple-touch-icon.png', 'favicon-32.png'],

      manifest: {
        name: 'Finance Track',
        short_name: 'Finance',
        description: 'Pelacak keuangan pribadi',
        lang: 'id',
        start_url: '/',
        scope: '/',
        display: 'standalone',
        orientation: 'portrait',
        // Disamakan dengan --surface mode terang. iOS memakai warna ini untuk
        // layar peluncuran, jadi kalau meleset akan terlihat sebagai kedipan
        // saat aplikasi dibuka.
        background_color: '#F2F2F7',
        theme_color: '#F2F2F7',
        icons: [
          { src: 'icon-192.png', sizes: '192x192', type: 'image/png' },
          { src: 'icon-512.png', sizes: '512x512', type: 'image/png' },
          { src: 'icon-512.png', sizes: '512x512', type: 'image/png', purpose: 'maskable' },
        ],
      },

      workbox: {
        globPatterns: ['**/*.{js,css,html,png,ico,svg,woff2}'],
        // Rute SPA apa pun jatuh ke index.html supaya membuka /laporan dalam
        // keadaan offline tetap memuat kerangka aplikasi.
        navigateFallback: 'index.html',
        navigateFallbackDenylist: [/^\/api/, /^\/healthz/],

        // /api SENGAJA tidak di-cache sama sekali.
        //
        // Men-cache respons keuangan berarti aplikasi bisa menampilkan saldo
        // lama tanpa memberi tahu bahwa angkanya basi - dan angka uang yang
        // salah lebih berbahaya daripada tidak ada angka. Saat offline,
        // kerangka aplikasi tetap terbuka dan galatnya ditampilkan apa adanya.
        //
        // Pencatatan saat offline ditangani antrean lokal di US-03, bukan oleh
        // cache service worker.
        runtimeCaching: [
          {
            urlPattern: ({ url }) => url.pathname.startsWith('/api'),
            handler: 'NetworkOnly',
          },
        ],
      },

      devOptions: { enabled: false },
    }),
  ],

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
