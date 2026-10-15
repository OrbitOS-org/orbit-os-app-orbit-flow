/**
 * Splits CSV text into rows of cells, for showing a file as a table.
 * Handles quoted cells ("a,b", "say ""hi""") and CRLF; not a full parser
 * (the CSV node is the one flows use).
 */
export function parseCSV(text: string, delimiter = ','): string[][] {
  const rows: string[][] = []
  let row: string[] = []
  let cell = ''
  let quoted = false
  for (let i = 0; i < text.length; i++) {
    const c = text[i]
    if (quoted) {
      if (c === '"' && text[i + 1] === '"') { cell += '"'; i++ }
      else if (c === '"') quoted = false
      else cell += c
    } else if (c === '"') quoted = true
    else if (c === delimiter) { row.push(cell); cell = '' }
    else if (c === '\n') { row.push(cell.replace(/\r$/, '')); rows.push(row); row = []; cell = '' }
    else cell += c
  }
  if (cell !== '' || row.length > 0) { row.push(cell.replace(/\r$/, '')); rows.push(row) }
  return rows
}
