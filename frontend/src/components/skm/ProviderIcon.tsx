import { useEffect, useMemo, useState } from "react";
import { FolderTree } from "lucide-react";
import { cn } from "../../lib/utils";
import { getProviderIconKey, getProviderIconSvg, normalizeProviderIconKey } from "../../lib/provider-icons";
import type { Provider } from "../../lib/api";

type ProviderIconAsset = {
  svg: string;
  hex?: string;
  variants?: Record<string, string>;
};

function normalizeHexColor(hex?: string) {
  const normalized = (hex ?? "").trim().replace(/^#/, "");
  if (/^[0-9a-fA-F]{3}$/.test(normalized) || /^[0-9a-fA-F]{6}$/.test(normalized)) {
    return `#${normalized}`;
  }
  return null;
}

function hexToRgb(hex: string) {
  const normalized = hex.replace(/^#/, "");
  const value = normalized.length === 3
    ? normalized.split("").map((part) => `${part}${part}`).join("")
    : normalized;

  return {
    red: Number.parseInt(value.slice(0, 2), 16),
    green: Number.parseInt(value.slice(2, 4), 16),
    blue: Number.parseInt(value.slice(4, 6), 16),
  };
}

function toRgba(hex: string, alpha: number) {
  const { red, green, blue } = hexToRgb(hex);
  return `rgba(${red}, ${green}, ${blue}, ${alpha})`;
}

function isNeutralHex(hex?: string | null) {
  if (!hex) {
    return true;
  }

  const normalized = hex.replace(/^#/, "").toLowerCase();
  return normalized === "000" || normalized === "000000" || normalized === "fff" || normalized === "ffffff";
}

function hasPaleForeground(svg: string) {
  return /#fff(f{3})?\b|#edecec\b|fill:\s*#edecec|fill=\"#fff/i.test(svg);
}

function getDisplaySvg(icon: ProviderIconAsset) {
  if (icon.variants?.light && hasPaleForeground(icon.svg)) {
    return icon.variants.light;
  }

  return icon.svg;
}

function getIconSurfaceStyle(icon: ProviderIconAsset | null) {
  const brandHex = normalizeHexColor(icon?.hex);
  if (!brandHex || isNeutralHex(brandHex)) {
    return undefined;
  }

  return {
    backgroundColor: toRgba(brandHex, 0.12),
    color: toRgba(brandHex, 0.88),
    boxShadow: `inset 0 0 0 1px ${toRgba(brandHex, 0.18)}`,
  };
}

type ProviderIconProps = {
  provider?: Pick<Provider, "icon" | "type" | "name" | "rootPath"> | null;
  iconKey?: string;
  className?: string;
  title?: string;
};

export function ProviderIcon({ provider, iconKey, className, title }: ProviderIconProps) {
  const resolvedIconKey = useMemo(() => {
    const explicit = normalizeProviderIconKey(iconKey);
    if (explicit) {
      return explicit;
    }
    return getProviderIconKey(provider);
  }, [iconKey, provider]);
  const [icon, setIcon] = useState<ProviderIconAsset | null>(null);

  useEffect(() => {
    let active = true;

    if (!resolvedIconKey) {
      setIcon(null);
      return () => {
        active = false;
      };
    }

    void getProviderIconSvg(resolvedIconKey).then((nextIcon) => {
      if (!active) {
        return;
      }
      setIcon(nextIcon as ProviderIconAsset | null);
    });

    return () => {
      active = false;
    };
  }, [resolvedIconKey]);

  const displaySvg = icon ? getDisplaySvg(icon) : null;
  const surfaceStyle = getIconSurfaceStyle(icon);

  return (
    <span
      title={title}
      style={surfaceStyle}
      className={cn(
        "inline-flex shrink-0 items-center justify-center rounded-xl bg-white text-slate-700 shadow-sm ring-1 ring-slate-200/80 [&_svg]:h-full [&_svg]:w-full",
        className,
      )}
    >
      {displaySvg ? <span className="block h-full w-full" dangerouslySetInnerHTML={{ __html: displaySvg }} /> : <FolderTree className="h-[60%] w-[60%] text-slate-400" />}
    </span>
  );
}

type ProviderLabelProps = {
  provider?: Pick<Provider, "icon" | "type" | "name" | "rootPath"> | null;
  className?: string;
  iconClassName?: string;
  textClassName?: string;
};

export function ProviderLabel({ provider, className, iconClassName, textClassName }: ProviderLabelProps) {
  return (
    <span className={cn("inline-flex min-w-0 items-center gap-2", className)}>
      <ProviderIcon provider={provider} className={cn("h-6 w-6 rounded-lg p-1", iconClassName)} title={provider?.name} />
      <span className={cn("truncate", textClassName)}>{provider?.name ?? "Unknown"}</span>
    </span>
  );
}