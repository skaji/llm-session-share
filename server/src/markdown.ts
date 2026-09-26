import { micromark } from 'micromark';
import { gfm, gfmHtml } from 'micromark-extension-gfm';

export function renderMarkdown(text: string, messageIndex: number): string {
  return micromark(text, {
    // Conversation HTML stays literal text; links cannot use script protocols.
    allowDangerousHtml: false,
    allowDangerousProtocol: false,
    extensions: [gfm()],
    htmlExtensions: [gfmHtml({ clobberPrefix: `message-${messageIndex}-` })],
  });
}
