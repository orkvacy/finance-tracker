import { useEffect, useState } from 'react'
import { Icon } from './Icon'
import './PasangHint.css'

const DITUTUP = 'ft.hintPasangDitutup'

/** Sudah berjalan sebagai aplikasi terpasang? */
function terpasang(): boolean {
  if (window.matchMedia?.('(display-mode: standalone)').matches) return true
  // Safari iOS tidak mendukung display-mode standalone sampai versi cukup baru,
  // jadi properti non-standar ini masih jadi penanda paling andal di sana.
  return (navigator as unknown as { standalone?: boolean }).standalone === true
}

function iOS(): boolean {
  return /iPad|iPhone|iPod/.test(navigator.userAgent)
}

/**
 * US-02 / FR-6.1 — memandu pemasangan ke Home Screen.
 *
 * iOS tidak menyediakan `beforeinstallprompt`; tidak ada API apa pun untuk
 * memunculkan dialog pasang. Satu-satunya jalan adalah menunjukkan langkahnya,
 * jadi kartu ini memang berisi instruksi, bukan tombol.
 *
 * Tampil sebagai kartu yang bisa ditutup, bukan modal: onboarding sebaiknya
 * mengajar sambil dipakai, bukan menghalangi layar pertama.
 */
export function PasangHint() {
  const [tampil, setTampil] = useState(false)
  const [prompt, setPrompt] = useState<Event | null>(null)

  useEffect(() => {
    if (terpasang()) return
    try {
      if (localStorage.getItem(DITUTUP)) return
    } catch { /* mode privat: anggap belum pernah ditutup */ }

    // Android dan desktop memberi event ini, sehingga pemasangan bisa satu tap.
    const onPrompt = (e: Event) => {
      e.preventDefault()
      setPrompt(e)
      setTampil(true)
    }
    window.addEventListener('beforeinstallprompt', onPrompt)

    // Di iOS event itu tidak akan pernah datang, jadi kartunya ditampilkan
    // langsung dengan instruksi manual.
    if (iOS()) setTampil(true)

    return () => window.removeEventListener('beforeinstallprompt', onPrompt)
  }, [])

  if (!tampil) return null

  function tutup() {
    setTampil(false)
    try { localStorage.setItem(DITUTUP, '1') } catch { /* mode privat */ }
  }

  async function pasang() {
    const p = prompt as unknown as { prompt: () => Promise<void> } | null
    if (!p) return
    await p.prompt()
    tutup()
  }

  return (
    <section className="pasang">
      <div className="pasang-atas">
        <div className="pasang-judul">Pasang ke Layar Utama</div>
        <button className="pasang-tutup" onClick={tutup} aria-label="Tutup petunjuk">
          <Icon name="silang" size={18} />
        </button>
      </div>

      {prompt ? (
        <>
          <p>Buka lebih cepat, tanpa bilah alamat browser.</p>
          <button className="pasang-aksi" onClick={pasang}>Pasang sekarang</button>
        </>
      ) : (
        <>
          <p>Terbuka penuh layar tanpa bilah alamat, dan muncul seperti aplikasi biasa.</p>
          <ol className="pasang-langkah">
            <li>
              Tekan <Icon name="bagikan" size={16} /> <b>Bagikan</b> di bilah bawah Safari
            </li>
            <li>Gulir, pilih <b>Add to Home Screen</b></li>
            <li>Tekan <b>Add</b></li>
          </ol>
        </>
      )}
    </section>
  )
}
