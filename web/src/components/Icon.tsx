/**
 * Satu set ikon garis, satu ketebalan (1.75), satu bahasa visual.
 *
 * Sengaja BUKAN emoji: emoji dirender berbeda di tiap platform dan bobotnya
 * tidak nyambung dengan teks di sebelahnya - salah satu penanda paling kentara
 * antarmuka hasil generate.
 */
const PATHS: Record<string, string> = {
  kopi: 'M4 9h12v6a4 4 0 0 1-4 4H8a4 4 0 0 1-4-4V9zM16 10h1.8a2.2 2.2 0 1 1 0 4.4H16M7 3.5v2M10.5 3.5v2M14 3.5v2',
  transport: 'M5 16.5V9.2A2.2 2.2 0 0 1 7.2 7h9.6A2.2 2.2 0 0 1 19 9.2v7.3M5 13h14M7.5 20v-2M16.5 20v-2',
  belanja: 'M4.5 7.5h15l-1.2 11a2 2 0 0 1-2 1.8H7.7a2 2 0 0 1-2-1.8zM9 7.5a3 3 0 0 1 6 0',
  kuliah: 'M12 4.2 21 8.5l-9 4.3-9-4.3zM6.5 10.6V16c0 1.4 2.5 2.6 5.5 2.6s5.5-1.2 5.5-2.6v-5.4',
  pulsa: 'M6.5 3h11v18h-11zM10.5 17.8h3',
  layar: 'M3.5 4.5h17v12h-17zM8 20h8',
  sehat: 'M12 6.5v11M6.5 12h11M3.2 3.2h17.6v17.6H3.2z',
  rumah: 'M3.5 10.2 12 3.6l8.5 6.6V19a1.6 1.6 0 0 1-1.6 1.6H5.1A1.6 1.6 0 0 1 3.5 19z',
  orang: 'M12 11.5a3.6 3.6 0 1 0 0-7.2 3.6 3.6 0 0 0 0 7.2zM4.5 20.4c0-3.4 3.4-5.6 7.5-5.6s7.5 2.2 7.5 5.6',
  kartu: 'M3 6h18v13H3zM3 10.5h18',
  bank: 'M3 6h18v13H3zM3 10.5h18',
  wallet: 'M3 5.5h18v14H3zM16.5 12.5h2.2',
  titik: 'M6 12h.01M12 12h.01M18 12h.01',
  masuk: 'M12 19V5M6 11l6-6 6 6',
  keluar: 'M12 5v14M6 13l6 6 6-6',
  tambah: 'M12 5.5v13M5.5 12h13',
  daftar: 'M4 7h16M4 12h16M4 17h10',
  grafik: 'M6 19V11M12 19V5.5M18 19v-5.5',
  setelan: 'M12 15.2a3.2 3.2 0 1 0 0-6.4 3.2 3.2 0 0 0 0 6.4zM12 2.5v2M12 19.5v2M2.5 12h2M19.5 12h2M5.2 5.2l1.5 1.5M17.3 17.3l1.5 1.5M18.8 5.2l-1.5 1.5M6.7 17.3l-1.5 1.5',
  unduh: 'M12 4v11M7.5 10.5 12 15l4.5-4.5M4.5 19.5h15',
  silang: 'M6 6l12 12M18 6L6 18',
}

export function Icon({ name, size = 20, stroke = 1.75 }: {
  name: keyof typeof PATHS | string
  size?: number
  stroke?: number
}) {
  const d = PATHS[name] ?? PATHS.titik
  return (
    <svg width={size} height={size} viewBox="0 0 24 24" fill="none"
         stroke="currentColor" stroke-width={stroke}
         stroke-linecap="round" stroke-linejoin="round" aria-hidden="true">
      <path d={d} />
    </svg>
  )
}

export const ikonKategori: Record<string, string> = {
  cat_makan: 'kopi',
  cat_transport: 'transport',
  cat_kos: 'rumah',
  cat_kuliah: 'kuliah',
  cat_pulsa: 'pulsa',
  cat_langganan: 'layar',
  cat_belanja: 'belanja',
  cat_kesehatan: 'sehat',
  cat_sosial: 'orang',
  cat_cicilan: 'kartu',
  cat_lain: 'titik',

  cat_kiriman: 'masuk',
  cat_beasiswa: 'kuliah',
  cat_freelance: 'layar',
  cat_refund: 'masuk',
}
