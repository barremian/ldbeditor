export function getHexValidationError(
  display: string,
  label: string
): string {
  if (typeof display !== "string") return "";
  if (!display.startsWith("0x")) return "";

  const raw = display.slice(2);
  if (raw.length % 2 !== 0) {
    return `${label} hex input must have an even number of characters.`;
  }

  if (!/^[0-9a-fA-F]*$/.test(raw)) {
    return `${label} hex input may only contain 0-9 and a-f.`;
  }

  return "";
}
