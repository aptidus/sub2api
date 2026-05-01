import { getConfiguredTableDefaultPageSize, normalizeTablePageSize } from '@/utils/tablePreferences'

const STORAGE_KEY = 'table-page-size'
const LEGACY_SOURCE_KEY = 'table-page-size-source'

export function getPersistedPageSize(fallback = getConfiguredTableDefaultPageSize()): number {
  if (typeof window !== 'undefined') {
    try {
      // Older builds wrote a separate source marker. If it is still present,
      // prefer the current server-provided default over stale browser state.
      if (window.localStorage.getItem(LEGACY_SOURCE_KEY) === 'user') {
        return normalizeTablePageSize(fallback)
      }
      const stored = window.localStorage.getItem(STORAGE_KEY)
      if (stored !== null) {
        const parsed = Number(stored)
        if (Number.isFinite(parsed)) {
          return normalizeTablePageSize(parsed)
        }
      }
    } catch (error) {
      console.warn('Failed to read persisted page size:', error)
    }
  }
  return normalizeTablePageSize(getConfiguredTableDefaultPageSize() || fallback)
}

export function setPersistedPageSize(size: number): void {
  if (typeof window === 'undefined') return
  try {
    window.localStorage.setItem(STORAGE_KEY, String(size))
    window.localStorage.removeItem(LEGACY_SOURCE_KEY)
  } catch (error) {
    console.warn('Failed to persist page size:', error)
  }
}
