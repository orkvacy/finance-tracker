import type { Account, Category, Currency, Transaction } from '../api'
import { rupiah, uang } from '../lib/format'
import { Icon, ikonKategori } from './Icon'

/**
 * Satu baris transaksi — dipakai Beranda maupun Transaksi.
 *
 * Elemennya <button>, bukan <div> berisi tombol hapus. Sebelumnya tiap baris
 * punya ikon silang permanen, dan itu menaruh aksi merusak tepat di jalur
 * jempol pada daftar yang digulir terus-menerus. Sekarang seluruh barisnya
 * membuka penyunting, dan hapus berada di dalam sana — satu tap lebih jauh
 * untuk aksi yang jarang, nol risiko untuk aksi yang sering.
 */
export function BarisTx({ t, accounts, categories, currencies, onPilih }: {
  t: Transaction
  accounts: Account[]
  categories: Category[]
  currencies: Currency[]
  onPilih: (t: Transaction) => void
}) {
  const kat = categories.find((c) => c.id === t.categoryId)
  const akun = accounts.find((a) => a.id === t.accountId)
  const cur = currencies.find((c) => c.code === t.currency)
  const judul = t.merchant || kat?.name || 'Tanpa kategori'

  return (
    <button className="baris" onClick={() => onPilih(t)} aria-label={`Ubah ${judul}`}>
      <div className="glif">
        <Icon name={ikonKategori[t.categoryId ?? ''] ?? 'titik'} size={19} />
      </div>
      <div className="isi">
        <div className="t1">{judul}</div>
        <div className="t2">
          {kat?.name ?? 'Tanpa kategori'} · {akun?.name ?? '?'}
          {t.status === 'draft' && ' · draft'}
        </div>
      </div>
      <div className="nilai num">
        {t.direction === 'out' ? '−' : '+'}{rupiah(t.amountIdr)}
        {t.currency !== 'IDR' && <span className="est">{uang(t.amount, cur)}</span>}
      </div>
      {/* Chevron: tanpa ini tidak ada yang menandakan barisnya bisa ditekan. */}
      <Icon name="kanan" size={15} stroke={2.2} />
    </button>
  )
}
