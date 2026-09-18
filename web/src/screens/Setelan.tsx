import { useState } from 'react'
import type { FormEvent } from 'react'
import { api, type Account, type Currency } from '../api'
import { rupiah, uang } from '../lib/format'
import { Icon } from '../components/Icon'

export function Setelan({ accounts, currencies, onChange }: {
  accounts: Account[]
  currencies: Currency[]
  onChange: () => void
}) {
  const [nama, setNama] = useState('')
  const [tipe, setTipe] = useState<Account['type']>('bank')
  const [mata, setMata] = useState('IDR')
  const [saldo, setSaldo] = useState('')
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState('')

  const cur = (code: string) => currencies.find((c) => c.code === code)
  const curTerpilih = cur(mata)

  async function tambah(e: FormEvent) {
    e.preventDefault()
    setBusy(true)
    try {
      await api.createAccount({
        name: nama.trim(), type: tipe, currency: mata,
        openingBalance: Number(saldo) || 0,
      })
      setNama(''); setSaldo(''); setError('')
      onChange()
    } catch (err) {
      setError((err as Error).message)
    } finally {
      setBusy(false)
    }
  }

  return (
    <>
      <div className="judul"><h1>Setelan</h1></div>

      <div className="seksi">Data</div>
      <section className="kartu rapat">
        {/* FR-5.6: ekspor wajib ada sejak M1. Aplikasi ini tidak boleh
            menyandera datanya sendiri. */}
        <a className="tautan" href="/api/export.csv" download>
          <span>Ekspor semua transaksi (CSV)</span>
          <Icon name="unduh" size={20} />
        </a>
      </section>

      <div className="seksi">Akun</div>
      <section className="kartu rapat">
        {accounts.length === 0 && (
          <div className="baris"><div className="isi t2">Belum ada akun.</div></div>
        )}
        {accounts.map((a) => (
          <div className="baris" key={a.id}>
            <div className="glif"><Icon name={a.type === 'bank' ? 'bank' : 'wallet'} size={19} /></div>
            <div className="isi">
              <div className="t1">{a.name}</div>
              <div className="t2">
                {a.type === 'bank' ? 'Rekening' : a.type === 'ewallet' ? 'E-wallet' : 'Tunai'} · {a.currency}
              </div>
            </div>
            <div className="nilai num">
              {uang(a.balance, cur(a.currency))}
              {a.currency !== 'IDR' && <span className="est">~ {rupiah(a.balanceIdr)}</span>}
            </div>
          </div>
        ))}
      </section>

      <div className="seksi">Tambah akun</div>
      <section className="kartu">
        {error && <p className="err-bar">{error}</p>}
        <form className="form" onSubmit={tambah}>
          <input placeholder="Nama akun" value={nama} required
                 onInput={(e) => setNama((e.target as HTMLInputElement).value)} />
          <select value={tipe} onChange={(e) => setTipe((e.target as HTMLSelectElement).value as Account['type'])}>
            <option value="bank">Bank</option>
            <option value="ewallet">E-wallet</option>
            <option value="cash">Tunai</option>
          </select>
          <select value={mata} onChange={(e) => setMata((e.target as HTMLSelectElement).value)}>
            {currencies.map((c) => <option key={c.code} value={c.code}>{c.code}</option>)}
          </select>
          <input type="number" inputMode="numeric" value={saldo}
                 placeholder={curTerpilih && curTerpilih.decimals > 0
                   ? `Saldo awal (dalam sen ${mata})`
                   : 'Saldo awal'}
                 onInput={(e) => setSaldo((e.target as HTMLInputElement).value)} />
          <button disabled={busy}>{busy ? 'Menyimpan...' : 'Tambah akun'}</button>
        </form>
        {/* Mata uang akun tidak bisa diubah setelah dibuat: mengubahnya berarti
            menafsirkan ulang seluruh riwayatnya. Lebih baik dilarang daripada
            diam-diam merusak data. */}
        <p className="t2" style={{ marginTop: 10, color: 'var(--content-3)', fontSize: 'var(--t-foot)' }}>
          Mata uang tidak bisa diubah setelah akun dibuat.
        </p>
      </section>
    </>
  )
}
