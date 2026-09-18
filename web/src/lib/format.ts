import type { Currency } from '../api'

/** Rp 47.000 - dipakai untuk semua angka pelaporan. */
export const rupiah = (n: number) => 'Rp ' + n.toLocaleString('id-ID')

/**
 * Bentuk pendek khas percakapan Indonesia: 47rb, 2,1jt.
 *
 * Dipakai di posisi sekunder saja (label, sumbu, keterangan). Angka utama
 * tetap ditulis penuh, karena di situlah ketelitian dibaca.
 */
export function singkat(n: number): string {
  const abs = Math.abs(n)
  if (abs >= 1_000_000) {
    const jt = n / 1_000_000
    return (Number.isInteger(jt) ? jt : jt.toFixed(1)).toString().replace('.', ',') + 'jt'
  }
  if (abs >= 1_000) {
    const rb = n / 1_000
    return (Number.isInteger(rb) ? rb : rb.toFixed(1)).toString().replace('.', ',') + 'rb'
  }
  return n.toString()
}

/** Memformat minor unit sesuai desimal mata uangnya: 45000 sen USD -> $ 450,00 */
export function uang(minor: number, c?: Currency): string {
  if (!c) return minor.toLocaleString('id-ID')
  const v = minor / 10 ** c.decimals
  return c.symbol + ' ' + v.toLocaleString('id-ID', {
    minimumFractionDigits: c.decimals,
    maximumFractionDigits: c.decimals,
  })
}

/**
 * Tanggal hari ini menurut zona LOKAL peranti, bukan UTC.
 *
 * `toISOString()` memakai UTC, jadi pada jam 01.00 WITA ia mengembalikan hari
 * kemarin - dan transaksi tercatat di tanggal yang salah. Ini bug yang sama
 * yang sudah dijaga di sisi server lewat kolom occurred_date.
 */
export function hariIni(): string {
  const d = new Date()
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}`
}

export function awalBulan(): string {
  const d = new Date()
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-01`
}

const HARI = ['Minggu', 'Senin', 'Selasa', 'Rabu', 'Kamis', 'Jumat', 'Sabtu']
const BULAN = ['Jan', 'Feb', 'Mar', 'Apr', 'Mei', 'Jun', 'Jul', 'Agu', 'Sep', 'Okt', 'Nov', 'Des']

/** "Hari ini", "Kemarin", atau "Rab, 17 Sep". */
export function tanggalRamah(iso: string): string {
  if (iso === hariIni()) return 'Hari ini'

  const kemarin = new Date()
  kemarin.setDate(kemarin.getDate() - 1)
  const p = (n: number) => String(n).padStart(2, '0')
  const isoKemarin = `${kemarin.getFullYear()}-${p(kemarin.getMonth() + 1)}-${p(kemarin.getDate())}`
  if (iso === isoKemarin) return 'Kemarin'

  const [y, m, d] = iso.split('-').map(Number)
  const dt = new Date(y, m - 1, d)
  return `${HARI[dt.getDay()].slice(0, 3)}, ${d} ${BULAN[m - 1]}`
}

export function namaBulan(iso: string): string {
  const [, m] = iso.split('-').map(Number)
  return BULAN[m - 1]
}
