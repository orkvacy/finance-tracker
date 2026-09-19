import { useCallback, useEffect, useRef, useState } from 'react'
import { api, type Account, type Category, type Currency, type Summary, type Transaction } from './api'
import { AddSheet, type Draft } from './components/AddSheet'
import { Icon } from './components/Icon'
import { PasangHint } from './components/PasangHint'
import { Toast } from './components/Toast'
import { Home } from './screens/Home'
import { Transaksi } from './screens/Transaksi'
import { Setelan } from './screens/Setelan'
import './App.css'

type Tab = 'beranda' | 'transaksi' | 'laporan' | 'setelan'

export default function App() {
  const [tab, setTab] = useState<Tab>('beranda')
  const [sheet, setSheet] = useState(false)
  const [edit, setEdit] = useState<Transaction | null>(null)
  const [error, setError] = useState('')

  // Nonce dipakai sebagai key Toast, supaya pesan yang sama dua kali berturut-
  // turut tetap memulai hitungan 6 detiknya dari awal.
  const [toast, setToast] = useState<{ n: number; pesan: string; undoId?: string } | null>(null)
  const nonce = useRef(0)

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

  function beriTahu(pesan: string, undoId?: string) {
    nonce.current += 1
    setToast({ n: nonce.current, pesan, undoId })
  }

  function bukaUbah(t: Transaction) {
    setEdit(t)
    setSheet(true)
  }

  function tutupSheet() {
    setSheet(false)
    // Dikosongkan belakangan supaya isinya tidak berkedip jadi form kosong
    // selagi sheet-nya masih meluncur turun.
    setTimeout(() => setEdit(null), 300)
  }

  async function simpan(d: Draft) {
    const isi = {
      accountId: d.accountId,
      categoryId: d.categoryId,
      amount: d.amount,
      direction: d.direction,
      currency: d.currency,
      merchant: d.merchant,
      ...(d.amountIdr ? { amountIdr: d.amountIdr } : {}),
    }

    if (d.id) {
      await api.updateTransaction(d.id, {
        ...isi, occurredAt: d.occurredAt, note: d.note, status: d.status,
      })
      await reload()
      beriTahu('Perubahan disimpan')
      return
    }

    await api.createTransaction(isi)
    setLastAccount(d.accountId)
    try { localStorage.setItem('ft.akunTerakhir', d.accountId) } catch { /* mode privat */ }
    await reload()
  }

  async function hapus(id: string) {
    tutupSheet()
    try {
      await api.deleteTransaction(id)
      await reload()
      beriTahu('Transaksi dihapus', id)
    } catch (e) { setError((e as Error).message) }
  }

  async function urungkan(id: string) {
    setToast(null)
    try { await api.restoreTransaction(id); await reload() }
    catch (e) { setError((e as Error).message) }
  }

  return (
    <div className="app">
      <main className="screen">
        {error && <p className="err-bar">{error}</p>}
        {tab === 'beranda' && <PasangHint />}

        {tab === 'beranda' && (
          <Home
            summary={summary} accounts={accounts} categories={categories}
            currencies={currencies} transactions={transactions}
            onTambahAkun={() => setTab('setelan')}
            onPilih={bukaUbah}
          />
        )}
        {tab === 'transaksi' && (
          <Transaksi
            transactions={transactions} accounts={accounts}
            categories={categories} currencies={currencies} onPilih={bukaUbah}
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
        <button className="fab" onClick={() => { setEdit(null); setSheet(true) }}
                aria-label="Catat transaksi baru">
          <Icon name="tambah" size={21} stroke={2.4} />
          Catat
        </button>
      )}

      {toast && (
        <Toast
          key={toast.n}
          pesan={toast.pesan}
          aksi={toast.undoId ? 'Urungkan' : undefined}
          onAksi={() => toast.undoId && urungkan(toast.undoId)}
          onTutup={() => setToast(null)}
        />
      )}

      <nav className="tabbar" role="tablist" aria-label="Bagian utama">
        <TabBtn id="beranda"    ikon="rumah"   label="Beranda"   aktif={tab} set={setTab} />
        <TabBtn id="transaksi"  ikon="daftar"  label="Transaksi" aktif={tab} set={setTab} />
        <TabBtn id="laporan"    ikon="grafik"  label="Laporan"   aktif={tab} set={setTab} />
        <TabBtn id="setelan"    ikon="setelan" label="Setelan"   aktif={tab} set={setTab} />
      </nav>

      <AddSheet
        open={sheet}
        onClose={tutupSheet}
        onSave={simpan}
        onHapus={hapus}
        edit={edit}
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
