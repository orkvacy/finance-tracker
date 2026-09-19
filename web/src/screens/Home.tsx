import type { Account, Category, Currency, Summary, Transaction } from '../api'
import { hariIni, namaBulan, rupiah, singkat, uang } from '../lib/format'
import { Icon } from '../components/Icon'
import { BarisTx } from '../components/BarisTx'

export function Home({ summary, accounts, categories, currencies, transactions, onTambahAkun, onPilih }: {
  summary: Summary | null
  accounts: Account[]
  categories: Category[]
  currencies: Currency[]
  transactions: Transaction[]
  onTambahAkun: () => void
  onPilih: (t: Transaction) => void
}) {
  if (accounts.length === 0) {
    return (
      <section className="kosong">
        <div className="h">Belum ada akun</div>
        <p>Tambahkan rekening atau e-wallet dulu, baru transaksi bisa dicatat.</p>
        <p style={{ marginTop: 16 }}>
          <button className="tag" onClick={onTambahAkun} style={{ cursor: 'pointer', border: 0 }}>
            Buka Setelan
          </button>
        </p>
      </section>
    )
  }

  const cur = (code: string) => currencies.find((c) => c.code === code)

  const hariIniList = transactions.filter((t) => t.occurredDate === hariIni())
  const keluarHariIni = hariIniList
    .filter((t) => t.direction === 'out' && !t.transferGroupId)
    .reduce((s, t) => s + t.amountIdr, 0)

  const totalIdr = accounts.reduce((s, a) => s + a.balanceIdr, 0)
  const adaAsing = accounts.some((a) => a.currency !== 'IDR')
  const maxKategori = Math.max(1, ...(summary?.byCategory.map((c) => c.total) ?? [1]))

  return (
    <>
      <div className="judul">
        <h1>Beranda</h1>
        <div className="sub">
          {summary ? `${namaBulan(summary.from)} ${summary.from.slice(0, 4)}` : ''}
        </div>
      </div>

      {/* Hero M1 = pengeluaran hari ini. Angka "aman jajan hari ini" butuh
          tagihan tetap dan target tabungan, yang baru ada di M2 (FR-4.1). */}
      <section className="kartu hero">
        <div className="lede">Keluar hari ini</div>
        <div className="angka">{rupiah(keluarHariIni)}</div>
        <div className="rinci">
          <span>Bulan ini keluar <b>{singkat(summary?.out ?? 0)}</b></span>
          <span>masuk <b>{singkat(summary?.in ?? 0)}</b></span>
        </div>
      </section>

      <section className="kartu">
        <div className="rank">
          <div className="atas">
            <span className="n">Saldo semua akun</span>
            <span className="v num">{rupiah(totalIdr)}</span>
          </div>
          {adaAsing && (
            // Ditandai taksiran karena saldo asing dinilai dengan kurs terakhir,
            // bukan dijumlahkan dari nilai historisnya.
            <div className="v" style={{ color: 'var(--content-3)' }}>
              termasuk taksiran kurs untuk akun mata uang asing
            </div>
          )}
        </div>
      </section>

      {hariIniList.length > 0 && (
        <>
          <div className="seksi">Hari ini</div>
          <section className="kartu rapat">
            {hariIniList.map((t) => (
              <BarisTx
                key={t.id} t={t} accounts={accounts} categories={categories}
                currencies={currencies} onPilih={onPilih}
              />
            ))}
          </section>
        </>
      )}

      {(summary?.byCategory.length ?? 0) > 0 && (
        <>
          <div className="seksi">Keluar per kategori</div>
          <section className="kartu">
            {summary!.byCategory.slice(0, 6).map((c) => (
              <div className="rank" key={c.categoryId || 'none'}>
                <div className="atas">
                  <span className="n">{c.name}</span>
                  <span className="v num">{rupiah(c.total)}</span>
                </div>
                {/* Batang peringkat: membandingkan panjang, bukan sudut.
                    Donut buruk untuk membandingkan lebih dari 3 irisan. */}
                <div className="rel"><i style={{ width: `${(c.total / maxKategori) * 100}%` }} /></div>
              </div>
            ))}
          </section>
        </>
      )}

      <div className="seksi">Akun</div>
      <section className="kartu rapat">
        {accounts.map((a) => (
          <div className="baris" key={a.id}>
            <div className="glif"><Icon name={a.type === 'bank' ? 'bank' : 'wallet'} size={19} /></div>
            <div className="isi">
              <div className="t1">{a.name}</div>
              <div className="t2">{a.type === 'bank' ? 'Rekening' : a.type === 'ewallet' ? 'E-wallet' : 'Tunai'}</div>
            </div>
            <div className="nilai num">
              {uang(a.balance, cur(a.currency))}
              {a.currency !== 'IDR' && <span className="est">~ {rupiah(a.balanceIdr)}</span>}
            </div>
          </div>
        ))}
      </section>
    </>
  )
}
