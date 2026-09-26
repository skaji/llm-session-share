import type { Store } from './store.js';

export function r2Store(bucket: R2Bucket): Store {
  return {
    async delete(key) {
      await bucket.delete(key);
    },
    async get(key) {
      const object = await bucket.get(key);
      return object ? object.text() : null;
    },
    async put(key, body) {
      await bucket.put(key, body, {
        httpMetadata: { contentType: 'application/json', cacheControl: 'private, no-cache' },
      });
    },
  };
}
