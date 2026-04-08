export function tryFormatJson(value: string): {
  formatted: string;
  isJson: boolean;
} {
  try {
    const parsed = JSON.parse(value);
    return { formatted: JSON.stringify(parsed, null, 2), isJson: true };
  } catch {
    return { formatted: value, isJson: false };
  }
}

export function formatValueForDisplay(
  value: string,
  prettyPrintJson: boolean
): string {
  if (!prettyPrintJson) return value;
  return tryFormatJson(value).formatted;
}
