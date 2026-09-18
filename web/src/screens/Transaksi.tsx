import type { Account, Category, Transaction } from '../api'
import { rupiah, tanggalRamah } from '../lib/format'
import { Icon, ikonKategori } from '../components/Icon'

export function Transaksi({ transactions, accounts, categories, onHapus }: {
  transactions: Transaction[]
  accounts: Account[]
  categories: Category[]
  onHapus: (id: string) => void
}) {
  const kat = (id: string | null) => categories.find((c) => c.id === id)
  const akun = (id: string) => accounts.find((a) => a.id === id)

  // Dikelompokkan per tanggal LOKAL yang sudah dihitung server (occurredDate),
  // bukan dari occurredAt yang UTC - supaya batas hari selalu sama antara
  // yang ditampilkan dan yang dipakai laporan.
  const perHari = new Map<string, Transaction[]>()
  for (const t of transactions) {
    const arr = perHari.get(t.occurredDate) ?? []
    arr.push(t)
    perHari.set(t.occurredDate, arr)
  }

  if (transactions.length === 0) {
    return (
      <section className="kosong">
        <div className="h">Belum ada transaksi</div>
        <p>Tekan tombol Catat di kanan bawah untuk mulai.</p>
      </section>
    )
  }

  return (
    <>
      <div className="judul">
        <h1>Transaksi</h1>
        <div className="sub">{transactions.length} terakhir</div>
      </div>

      {[...perHari.entries()].map(([tgl, list]) => {
        const totalHari = list
          .filter((t) => t.direction === 'out' && !t.transferGroupId)
          .reduce((s, t) => s + t.amountIdr, 0)
        return (
          <div key={tgl}>
            <div className="seksi" style={{ display: 'flex', justifyContent: 'space-between' }}>
              <span>{tanggalRamah(tgl)}</span>
              {totalHari > 0 && <span style={{ textTransform: 'none' }}>{rupiah(totalHari)}</span>}
            </div>
            <section className="kartu rapat">
              {list.map((t) => (
                <div className="baris" key={t.id}>
                  <div className="glif">
                    <Icon name={ikonKategori[t.categoryId ?? ''] ?? 'titik'} size={19} />
                  </div>
                  <div className="isi">
                    <div className="t1">{t.merchant || kat(t.categoryId)?.name || 'Tanpa kategori'}</div>
                    <div className="t2">
                      {kat(t.categoryId)?.name ?? 'Tanpa kategori'} · {akun(t.accountId)?.name ?? '?'}
                      {t.status === 'draft' && ' · draft'}
                    </div>
                  </div>
                  <div className="nilai num">
                    {t.direction === 'out' ? '−' : '+'}{rupiah(t.amountIdr)}
                  </div>
                  <button
                    className="hapus"
                    aria-label={`Hapus ${t.merchant || 'transaksi'}`}
                    onClick={() => onHapus(t.id)}
                  >
                    <Icon name="silang" size={17} />
                  </button>
                </div>
              ))}
            </section>
          </div>
        )
      })}
    </>
  )
}
