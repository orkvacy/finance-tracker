import type { Account, Category, Currency, Transaction } from '../api'
import { rupiah, tanggalRamah } from '../lib/format'
import { BarisTx } from '../components/BarisTx'

export function Transaksi({ transactions, accounts, categories, currencies, onPilih }: {
  transactions: Transaction[]
  accounts: Account[]
  categories: Category[]
  currencies: Currency[]
  onPilih: (t: Transaction) => void
}) {
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
        <div className="sub">{transactions.length} terakhir · ketuk untuk mengubah</div>
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
                <BarisTx
                  key={t.id} t={t} accounts={accounts} categories={categories}
                  currencies={currencies} onPilih={onPilih}
                />
              ))}
            </section>
          </div>
        )
      })}
    </>
  )
}
