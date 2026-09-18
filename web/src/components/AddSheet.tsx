import { useEffect, useRef, useState } from 'react'
import type { Account, Category, Currency } from '../api'
import { uang } from '../lib/format'
import { Icon, ikonKategori } from './Icon'
import './AddSheet.css'

export type Draft = {
  amount: number
  accountId: string
  categoryId: string | null
  direction: 'in' | 'out'
  currency: string
  amountIdr?: number
  merchant: string
}

/**
 * Quick-add (US-01 / FR-1.1).
 *
 * Targetnya NFR-1: dari tap ikon sampai tersimpan <= 5 detik dan <= 4
 * interaksi. Karena itu:
 *   - keypad sudah terlihat begitu sheet terbuka, tanpa perlu fokus ke input
 *     (keyboard sistem iOS butuh satu tap lagi dan memakan setengah layar)
 *   - kategori sudah terpilih otomatis berdasarkan jam (FR-1.2)
 *   - akun default = akun terakhir dipakai
 * Jalur tercepatnya: ketik nominal, tekan Simpan. Dua interaksi.
 */
export function AddSheet({ open, onClose, onSave, accounts, categories, currencies, defaultAccountId }: {
  open: boolean
  onClose: () => void
  onSave: (d: Draft) => Promise<void>
  accounts: Account[]
  categories: Category[]
  currencies: Currency[]
  defaultAccountId: string
}) {
  const [raw, setRaw] = useState('')
  const [direction, setDirection] = useState<'in' | 'out'>('out')
  const [accountId, setAccountId] = useState(defaultAccountId)
  const [categoryId, setCategoryId] = useState<string | null>(null)
  const [merchant, setMerchant] = useState('')
  const [idrRaw, setIdrRaw] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')
  const chipsRef = useRef<HTMLDivElement>(null)

  const relevan = categories.filter((c) => c.kind === (direction === 'in' ? 'income' : 'expense'))
  const akun = accounts.find((a) => a.id === accountId) ?? accounts[0]
  const currency = akun?.currency ?? 'IDR'
  const cur = currencies.find((c) => c.code === currency)
  const asing = currency !== 'IDR'

  useEffect(() => {
    if (!open) return
    setRaw(''); setMerchant(''); setIdrRaw(''); setError(''); setDirection('out')
    setAccountId(defaultAccountId || accounts[0]?.id || '')
    setCategoryId(tebakKategori(categories))
  }, [open, defaultAccountId])

  // Chip terpilih digeser ke tampak. Tebakan kategori tidak ada gunanya kalau
  // chip-nya berada di luar layar - pengguna akan mengira tidak ada yang
  // terpilih, lalu menekan satu chip lagi. Tepat satu tap yang ingin dihemat.
  useEffect(() => {
    if (!open || !categoryId) return
    const el = chipsRef.current?.querySelector('[aria-pressed="true"]')
    el?.scrollIntoView({ block: 'nearest', inline: 'center' })
  }, [open, categoryId])

  // Esc menutup sheet; di iOS gestur geser ke bawah yang dipakai, tapi di
  // desktop keyboard harus tetap punya jalan keluar (accessibility.md).
  useEffect(() => {
    if (!open) return
    const h = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose() }
    window.addEventListener('keydown', h)
    return () => window.removeEventListener('keydown', h)
  }, [open, onClose])

  const nominal = Number(raw) || 0
  const idr = Number(idrRaw) || 0
  const bisaSimpan = nominal > 0 && !!accountId && (!asing || idr > 0) && !busy

  function tekan(k: string) {
    setError('')
    if (k === 'del') setRaw((v) => v.slice(0, -1))
    else if (k === '000') setRaw((v) => (v === '' ? '' : (v + '000').slice(0, 12)))
    else setRaw((v) => (v + k).replace(/^0+/, '').slice(0, 12))
  }

  async function simpan() {
    if (!bisaSimpan) return
    setBusy(true)
    try {
      await onSave({
        amount: nominal,
        accountId,
        categoryId,
        direction,
        currency,
        merchant: merchant.trim(),
        ...(asing ? { amountIdr: idr } : {}),
      })
      onClose()
    } catch (e) {
      setError((e as Error).message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <>
      <div className={'scrim' + (open ? ' on' : '')} onClick={onClose} />
      <section
        className={'sheet' + (open ? ' on' : '')}
        role="dialog"
        aria-modal="true"
        aria-label={direction === 'out' ? 'Catat pengeluaran' : 'Catat pemasukan'}
        aria-hidden={!open}
      >
        <div className="grabber" />

        <header className="sheethead">
          {/* sheets.md: Cancel di tepi kiri, Done di tepi kanan, dan Done
              selalu berpasangan dengan jalan keluar lain. */}
          <button className="txt" onClick={onClose}>Batal</button>
          <div className="seg" role="group" aria-label="Arah transaksi">
            <button
              aria-pressed={direction === 'out'}
              onClick={() => { setDirection('out'); setCategoryId(tebakKategori(categories)) }}
            >Keluar</button>
            <button
              aria-pressed={direction === 'in'}
              onClick={() => { setDirection('in'); setCategoryId(null) }}
            >Masuk</button>
          </div>
          <button className="txt strong" disabled={!bisaSimpan} onClick={simpan}>
            {busy ? '...' : 'Simpan'}
          </button>
        </header>

        <div className="entry">
          <div className="cur">{cur?.symbol ?? 'Rp'}</div>
          <div className={'num-besar' + (raw === '' ? ' kosong' : '')}
               aria-live="polite"
               aria-label={`Nominal ${nominal}`}>
            {raw === '' ? '0' : uang(nominal, cur).replace((cur?.symbol ?? '') + ' ', '')}
          </div>
          {asing && (
            <input
              className="idr"
              type="number"
              inputMode="numeric"
              placeholder="Rupiah yang benar-benar diterima"
              value={idrRaw}
              onInput={(e) => setIdrRaw((e.target as HTMLInputElement).value)}
            />
          )}
        </div>

        {error && <p className="err">{error}</p>}

        <div className="chips" ref={chipsRef} role="group" aria-label="Kategori">
          {relevan.map((c) => (
            <button
              key={c.id}
              className="chip"
              aria-pressed={categoryId === c.id}
              onClick={() => setCategoryId(categoryId === c.id ? null : c.id)}
            >
              <Icon name={ikonKategori[c.id] ?? c.icon ?? 'titik'} size={17} />
              {c.name.split(' ')[0]}
            </button>
          ))}
        </div>

        <input
          className="merchant"
          placeholder="Merchant atau catatan (opsional)"
          value={merchant}
          onInput={(e) => setMerchant((e.target as HTMLInputElement).value)}
        />

        <div className="keys">
          {['1', '2', '3', '4', '5', '6', '7', '8', '9'].map((k) => (
            <button key={k} className="key num" onClick={() => tekan(k)}>{k}</button>
          ))}
          <button className="key kecil" onClick={() => tekan('000')}>000</button>
          <button className="key num" onClick={() => tekan('0')}>0</button>
          <button className="key kecil" onClick={() => tekan('del')} aria-label="Hapus satu angka">&#9003;</button>
        </div>

        <div className="sheetfoot">
          <select
            className="akun"
            value={accountId}
            onChange={(e) => setAccountId((e.target as HTMLSelectElement).value)}
            aria-label="Akun"
          >
            {accounts.map((a) => (
              <option key={a.id} value={a.id}>{a.name}{a.currency !== 'IDR' ? ` (${a.currency})` : ''}</option>
            ))}
          </select>
          <button className="primary" disabled={!bisaSimpan} onClick={simpan}>
            {busy ? 'Menyimpan...' : 'Simpan'}
          </button>
        </div>
      </section>
    </>
  )
}

/**
 * FR-1.2: kategori ditebak dari jam.
 *
 * Tebakan sederhana ini sudah menghapus satu tap pada mayoritas pencatatan,
 * dan itu langsung memotong waktu menuju target NFR-1. Versi berikutnya
 * sebaiknya belajar dari kebiasaan nyata, bukan dari jam saja.
 */
function tebakKategori(categories: Category[]): string | null {
  const jam = new Date().getHours()
  const ada = (id: string) => (categories.some((c) => c.id === id) ? id : null)
  // Jam makan mahasiswa: sarapan, makan siang, dan jajan malam yang panjang
  // sampai lewat tengah malam.
  if (jam >= 6 && jam < 10) return ada('cat_makan')
  if (jam >= 11 && jam < 15) return ada('cat_makan')
  if (jam >= 17 || jam < 2) return ada('cat_makan')
  return ada('cat_lain')
}
