import { useCallback, useEffect, useState } from 'react'
import { api, type Account, type Category, type Currency, type Summary, type Transaction } from './api'
import { AddSheet, type Draft } from './components/AddSheet'
import { Icon } from './components/Icon'
import { Home } from './screens/Home'
import { Transaksi } from './screens/Transaksi'
import { Setelan } from './screens/Setelan'
import './App.css'

type Tab = 'beranda' | 'transaksi' | 'laporan' | 'setelan'

export default function App() {
  const [tab, setTab] = useState<Tab>('beranda')
  const [sheet, setSheet] = useState(false)
  const [error, setError] = useState('')

  const [accounts, setAccounts] = useState<Account[]>([])
  const [categories, setCategories] = useState<Category[]>([])
  const [currencies, setCurrencies] = useState<Currency[]>([])
  const [summary, setSummary] = useState<Summary | null>(null)
  const [transactions, setTransactions] = useState<Transaction[]>([])

  // Akun terakhir dipakai jadi default berikutnya (FR-1.2). Disimpan lokal
  // karena ini kenyamanan per-peranti, bukan data yang perlu disinkronkan.
  // Dibungkus try/catch: localStorage bisa melempar di mode privat Safari.
  const [lastAccount, setLastAccount] = useState(() => {
    try { return localStorage.getItem('ft.akunTerakhir') ?? '' } catch { return '' }
  })

  const reload = useCallback(async () => {
    try {
      const [a, c, cur, s, t] = await Promise.all([
        api.accounts(), api.categories(), api.currencies(),
        api.summary(), api.transactions({ limit: '100' }),
      ])
      setAccounts(a); setCategories(c); setCurrencies(cur)
      setSummary(s); setTransactions(t); setError('')
    } catch (e) {
      setError((e as Error).message)
    }
  }, [])

  useEffect(() => { reload() }, [reload])

  async function simpan(d: Draft) {
    await api.createTransaction({
      accountId: d.accountId,
      categoryId: d.categoryId,
      amount: d.amount,
      direction: d.direction,
      currency: d.currency,
      merchant: d.merchant,
      ...(d.amountIdr ? { amountIdr: d.amountIdr } : {}),
    })
    setLastAccount(d.accountId)
    try { localStorage.setItem('ft.akunTerakhir', d.accountId) } catch { /* mode privat */ }
    await reload()
  }

  async function hapus(id: string) {
    try { await api.deleteTransaction(id); await reload() }
    catch (e) { setError((e as Error).message) }
  }

  return (
    <div className="app">
      <main className="screen">
        {error && <p className="err-bar">{error}</p>}

        {tab === 'beranda' && (
          <Home
            summary={summary} accounts={accounts} categories={categories}
            currencies={currencies} transactions={transactions}
            onTambahAkun={() => setTab('setelan')}
          />
        )}
        {tab === 'transaksi' && (
          <Transaksi
            transactions={transactions} accounts={accounts}
            categories={categories} onHapus={hapus}
          />
        )}
        {tab === 'laporan' && (
          <section className="kosong">
            <div className="h">Laporan</div>
            <p>Grafik laju pemakaian dan peringkat kategori datang di M2,
               setelah anggaran per kategori ada.</p>
          </section>
        )}
        {tab === 'setelan' && (
          <Setelan accounts={accounts} currencies={currencies} onChange={reload} />
        )}
      </main>

      {/* Aksi utama sengaja BUKAN tab: tab hanya untuk navigasi (tab-bars.md).
          Posisinya di kanan bawah karena kanan atas tidak terjangkau jempol
          di layar 414x896 - NFR-1 menang atas konvensi penempatan itu. */}
      {accounts.length > 0 && (
        <button className="fab" onClick={() => setSheet(true)} aria-label="Catat transaksi baru">
          <Icon name="tambah" size={21} stroke={2.4} />
          Catat
        </button>
      )}

      <nav className="tabbar" role="tablist" aria-label="Bagian utama">
        <TabBtn id="beranda"    ikon="rumah"   label="Beranda"   aktif={tab} set={setTab} />
        <TabBtn id="transaksi"  ikon="daftar"  label="Transaksi" aktif={tab} set={setTab} />
        <TabBtn id="laporan"    ikon="grafik"  label="Laporan"   aktif={tab} set={setTab} />
        <TabBtn id="setelan"    ikon="setelan" label="Setelan"   aktif={tab} set={setTab} />
      </nav>

      <AddSheet
        open={sheet}
        onClose={() => setSheet(false)}
        onSave={simpan}
        accounts={accounts}
        categories={categories}
        currencies={currencies}
        defaultAccountId={lastAccount}
      />
    </div>
  )
}

function TabBtn({ id, ikon, label, aktif, set }: {
  id: Tab; ikon: string; label: string; aktif: Tab; set: (t: Tab) => void
}) {
  return (
    <button className="tab" role="tab" aria-selected={aktif === id} onClick={() => set(id)}>
      <Icon name={ikon} size={24} stroke={1.9} />
      <span>{label}</span>
    </button>
  )
}
