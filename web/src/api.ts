export type Account = {
  id: string
  name: string
  type: 'bank' | 'ewallet' | 'cash'
  openingBalance: number
  currency: string
  isArchived: boolean
  sortOrder: number
  /** Minor unit mata uang akun (sen untuk USD/AUD, satuan utuh untuk IDR/KRW). */
  balance: number
  /** Taksiran memakai kurs terakhir — bergerak mengikuti kurs, hanya untuk tampilan. */
  balanceIdr: number
}

export type Currency = {
  code: string
  decimals: number
  symbol: string
  rateToIdr: number
}

export type Category = {
  id: string
  parentId: string | null
  name: string
  kind: 'income' | 'expense'
  icon: string
  sortOrder: number
}

export type Transaction = {
  id: string
  accountId: string
  categoryId: string | null
  /** Minor unit dari `currency`, selalu positif. Arah dibawa `direction`. */
  amount: number
  currency: string
  /** Dibekukan saat dibuat; tidak pernah dihitung ulang. */
  rateToIdr: number
  /** Satu-satunya angka yang dipakai semua agregasi. */
  amountIdr: number
  direction: 'in' | 'out'
  occurredAt: string
  occurredDate: string
  note: string
  merchant: string
  source: string
  status: 'draft' | 'confirmed'
  transferGroupId: string | null
}

export type Summary = {
  from: string
  to: string
  in: number
  out: number
  net: number
  byCategory: { categoryId: string; name: string; total: number }[]
}

async function req<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(path, {
    ...init,
    headers: init?.body ? { 'Content-Type': 'application/json' } : undefined,
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({ error: res.statusText }))
    throw new Error(body.error ?? `HTTP ${res.status}`)
  }
  return res.status === 204 ? (undefined as T) : res.json()
}

export const api = {
  accounts: () => req<Account[]>('/api/accounts'),
  createAccount: (a: Partial<Account>) =>
    req<Account>('/api/accounts', { method: 'POST', body: JSON.stringify(a) }),

  categories: () => req<Category[]>('/api/categories'),
  currencies: () => req<Currency[]>('/api/currencies'),

  transactions: (q: Record<string, string> = {}) =>
    req<Transaction[]>('/api/transactions?' + new URLSearchParams(q)),
  createTransaction: (t: Partial<Transaction>) =>
    req<Transaction>('/api/transactions', { method: 'POST', body: JSON.stringify(t) }),
  updateTransaction: (id: string, t: Partial<Transaction>) =>
    req<{ status: string }>(`/api/transactions/${id}`, { method: 'PUT', body: JSON.stringify(t) }),
  deleteTransaction: (id: string) =>
    req<{ status: string }>(`/api/transactions/${id}`, { method: 'DELETE' }),
  /** Membatalkan hapus (FR-1.5). Mungkin karena hapusnya lunak, bukan permanen. */
  restoreTransaction: (id: string) =>
    req<{ status: string }>(`/api/transactions/${id}/restore`, { method: 'POST' }),

  summary: (q: Record<string, string> = {}) =>
    req<Summary>('/api/summary?' + new URLSearchParams(q)),
}

export const rupiah = (n: number) => 'Rp ' + n.toLocaleString('id-ID')

/** Memformat minor unit sesuai desimal mata uangnya: 45000 sen USD -> $450,00 */
export function uang(minor: number, c?: Currency) {
  if (!c) return minor.toLocaleString('id-ID')
  const v = minor / 10 ** c.decimals
  return c.symbol + ' ' + v.toLocaleString('id-ID', {
    minimumFractionDigits: c.decimals,
    maximumFractionDigits: c.decimals,
  })
}
