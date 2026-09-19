import { useEffect } from 'react'
import './Toast.css'

/**
 * Pemberitahuan singkat dengan satu aksi opsional.
 *
 * Dipakai untuk undo setelah hapus (US-05 / FR-1.5). Pilihan ini menggantikan
 * dialog konfirmasi dengan sengaja: konfirmasi menambah satu tap pada SETIAP
 * penghapusan demi mencegah yang jarang salah, dan setelah beberapa kali
 * ditekan refleks — ia berhenti melindungi apa pun. Undo membalik urutannya:
 * aksi berjalan langsung, dan yang salah bisa ditarik kembali.
 *
 * Komponennya sengaja tanpa state waktu sendiri. Pemanggil me-remount lewat
 * `key`, sehingga pesan yang sama dua kali berturut-turut tetap mengulang
 * hitungannya dari awal.
 */
export function Toast({ pesan, aksi, onAksi, onTutup, ms = 6000 }: {
  pesan: string
  aksi?: string
  onAksi?: () => void
  onTutup: () => void
  ms?: number
}) {
  useEffect(() => {
    const id = setTimeout(onTutup, ms)
    return () => clearTimeout(id)
    // onTutup sengaja tidak ikut: identitasnya berubah tiap render induk, dan
    // ikut-serta membuat hitungannya di-reset terus sampai tidak pernah habis.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [ms])

  return (
    // assertive, bukan polite: jendelanya cuma 6 detik, jadi pembaca layar
    // harus menyebutkannya sekarang, bukan menunggu jeda bicara berikutnya.
    <div className="toast" role="alert" aria-live="assertive">
      <span>{pesan}</span>
      {aksi && <button onClick={onAksi}>{aksi}</button>}
    </div>
  )
}
