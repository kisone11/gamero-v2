/**
 * Strip markdown formatting from text for use in content snippets.
 * Removes: links, images, bold, italic, headers, code, blockquotes, lists.
 */
export function stripMarkdown(text: string): string {
  if (!text) return ''
  return text
    .replace(/!?\[([^\]]*)\]\([^)]*\)/g, '$1')
    .replace(/\*\*([^*]+)\*\*/g, '$1')
    .replace(/__([^_]+)__/g, '$1')
    .replace(/(?<!\w)[*_]([^*_]+)[*_](?!\w)/g, '$1')
    .replace(/^#{1,6}\s/gm, '')
    .replace(/`{1,3}[^`]*`{1,3}/g, '')
    .replace(/^[>\-*+]\s/gm, '')
    .replace(/\s+/g, ' ')
    .trim()
}
