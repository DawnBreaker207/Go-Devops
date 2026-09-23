/** Offline ticket-QR cache on raw browser IndexedDB (no `idb` dep; don't add one). Stores base64 PNG per ticket id. Best-effort: every failure (blocked, private mode, quota) is swallowed as a miss, never breaks the ticket screen. */

const DB_NAME = 'cp-ticket-qr';
const DB_VERSION = 1;
const STORE_NAME = 'qr';

let dbPromise: Promise<IDBDatabase | null> | null = null;

const openDb = (): Promise<IDBDatabase | null> => {
  if (dbPromise) return dbPromise;

  dbPromise = new Promise((resolve) => {
    if (typeof indexedDB === 'undefined') {
      resolve(null);
      return;
    }
    try {
      const request = indexedDB.open(DB_NAME, DB_VERSION);
      request.onupgradeneeded = () => {
        const db = request.result;
        if (!db.objectStoreNames.contains(STORE_NAME)) {
          db.createObjectStore(STORE_NAME);
        }
      };
      request.onsuccess = () => resolve(request.result);
      request.onerror = () => resolve(null);
    } catch {
      resolve(null);
    }
  });

  return dbPromise;
};

/** Read a cached QR (base64 PNG). `null` on miss/unreadable. */
export const getCachedQR = async (ticketId: string): Promise<string | null> => {
  const db = await openDb();
  if (!db) return null;

  return new Promise((resolve) => {
    try {
      const tx = db.transaction(STORE_NAME, 'readonly');
      const request = tx.objectStore(STORE_NAME).get(ticketId);
      request.onsuccess = () => resolve(typeof request.result === 'string' ? request.result : null);
      request.onerror = () => resolve(null);
    } catch {
      resolve(null);
    }
  });
};

/** Write/update a cached QR. Failures swallowed: cache is a convenience, never a hard path. */
export const setCachedQR = async (ticketId: string, qrBase64: string): Promise<void> => {
  const db = await openDb();
  if (!db) return;

  return new Promise((resolve) => {
    try {
      const tx = db.transaction(STORE_NAME, 'readwrite');
      tx.objectStore(STORE_NAME).put(qrBase64, ticketId);
      tx.oncomplete = () => resolve();
      tx.onerror = () => resolve();
    } catch {
      resolve();
    }
  });
};
