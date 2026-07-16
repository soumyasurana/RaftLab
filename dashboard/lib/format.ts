export function formatNumber(value: number | undefined | null) {
  if (value === undefined || value === null || Number.isNaN(value)) {
    return "0";
  }

  return new Intl.NumberFormat("en-US", {
    maximumFractionDigits: 0,
  }).format(value);
}

export function formatDuration(value: string | number | undefined | null) {
  if (value === undefined || value === null) {
    return "n/a";
  }

  if (typeof value === "number") {
    const ms = value / 1_000_000;
    return ms >= 1000 ? `${(ms / 1000).toFixed(1)}s` : `${ms.toFixed(0)}ms`;
  }

  if (/^\d+$/.test(value)) {
    const ns = Number(value);
    const ms = ns / 1_000_000;
    return ms >= 1000 ? `${(ms / 1000).toFixed(1)}s` : `${ms.toFixed(0)}ms`;
  }

  return value;
}

export function formatBytes(value: number | undefined | null) {
  if (value === undefined || value === null) {
    return "0 B";
  }

  if (value === 0) {
    return "0 B";
  }

  const units = ["B", "KB", "MB", "GB"];
  let index = 0;
  let current = value;

  while (current >= 1024 && index < units.length - 1) {
    current /= 1024;
    index += 1;
  }

  return `${current.toFixed(current >= 10 || index === 0 ? 0 : 1)} ${units[index]}`;
}

export function formatPercent(value: number | undefined | null) {
  if (value === undefined || value === null) {
    return "0%";
  }

  return `${value.toFixed(1)}%`;
}

export function formatShortDate(value: string | number | undefined | null) {
  if (!value) {
    return "n/a";
  }

  const date = new Date(value);
  if (Number.isNaN(date.getTime())) {
    return String(value);
  }

  return new Intl.DateTimeFormat("en-US", {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  }).format(date);
}
