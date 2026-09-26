import { Storage } from '@google-cloud/storage';
import type { Store } from './store.js';

export function gcsStore(bucketName: string): Store {
  const bucket = new Storage().bucket(bucketName);
  return {
    async delete(key) {
      await bucket.file(key).delete({ ignoreNotFound: true });
    },
    async get(key) {
      try {
        const [body] = await bucket.file(key).download();
        return new TextDecoder().decode(body);
      } catch (error) {
        if ((error as { code?: number }).code === 404) return null;
        throw error;
      }
    },
    async put(key, body) {
      await bucket.file(key).save(body, {
        resumable: false,
        metadata: { contentType: 'application/json', cacheControl: 'private, no-cache' },
      });
    },
  };
}
