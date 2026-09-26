import { createApp } from './app.js';
import { r2Store } from './storage/r2.js';

export default {
  fetch(request: Request, env: { BUCKET: R2Bucket }) {
    return createApp(r2Store(env.BUCKET)).fetch(request);
  },
};
