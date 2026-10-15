/**
 * Copies text to the clipboard; resolves to whether it worked.
 *
 * navigator.clipboard only exists in secure contexts (https, localhost);
 * the editor is usually opened as http://<device IP>:<port>, so this falls
 * back to a hidden text area and document.execCommand('copy'), which still
 * works there when called from a click.
 */
export async function copyText(text: string): Promise<boolean> {
  if (navigator.clipboard) {
    try {
      await navigator.clipboard.writeText(text)
      return true
    } catch {
      // fall through to the old way
    }
  }
  const ta = document.createElement('textarea')
  ta.value = text
  ta.setAttribute('readonly', '')
  ta.style.position = 'fixed'
  ta.style.opacity = '0'
  document.body.appendChild(ta)
  ta.select()
  let ok = false
  try {
    ok = document.execCommand('copy')
  } catch {
    ok = false
  }
  ta.remove()
  return ok
}
