const KEY = 'folio.answers.v1'

let current: boolean | null = null
const listeners = new Set<() => void>()

function load(): boolean {
  if (current !== null) return current
  try {
    const raw = typeof localStorage !== 'undefined' ? localStorage.getItem(KEY) : null
    current = raw === null ? true : raw === 'on'
  } catch {
    current = true
  }
  return current
}

function persist() {
  try {
    localStorage.setItem(KEY, current ? 'on' : 'off')
  } catch {
  }
}

function emit() {
  for (const l of listeners) l()
}

export const answersStore = {
  subscribe(cb: () => void) {
    listeners.add(cb)
    return () => {
      listeners.delete(cb)
    }
  },
  getSnapshot(): boolean {
    return load()
  },
  getServerSnapshot(): boolean {
    return true
  },
  toggle() {
    current = !load()
    persist()
    emit()
  },
}
