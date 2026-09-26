import { mkdir, readFile, rename, rm, writeFile } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { randomUUID } from 'node:crypto';
import type { Store } from './store.js';

export function localStore(directory: string): Store {
  return {
    async delete(key) {
      await rm(join(directory, key), { force: true });
    },
    async get(key) {
      try {
        return await readFile(join(directory, key), 'utf8');
      } catch (error) {
        if ((error as NodeJS.ErrnoException).code === 'ENOENT') return null;
        throw error;
      }
    },
    async put(key, body) {
      const target = join(directory, key);
      await mkdir(dirname(target), { recursive: true });
      const temporary = `${target}.${randomUUID()}.tmp`;
      try {
        await writeFile(temporary, body);
        await rename(temporary, target);
      } finally {
        await rm(temporary, { force: true });
      }
    },
  };
}
