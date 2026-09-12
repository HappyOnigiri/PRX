import type { LucideIcon } from "lucide-react";

// バッジの形と補助的な視覚表現はここに集約する。既存の呼び出しは label だけを
// 渡せば従来どおり表示され、新しい状態バッジだけがアイコンと分割ラベルを使う。
export function StatusBadge({
  className,
  label,
  title,
  icon: Icon,
  mainLabel,
  secondaryLabel,
  accessibleLabel,
}: {
  className?: string;
  label: string;
  title?: string;
  icon?: LucideIcon;
  mainLabel?: string;
  secondaryLabel?: string;
  accessibleLabel?: string;
}) {
  const hasSplitLabel = mainLabel !== undefined || secondaryLabel !== undefined;
  const visibleMainLabel = mainLabel ?? label;
  return (
    <span
      aria-label={accessibleLabel}
      className={className ? `status-badge ${className}` : "status-badge"}
      role={accessibleLabel ? "img" : undefined}
      title={title}
    >
      {Icon && (
        <Icon
          aria-hidden="true"
          className="status-badge-icon"
          focusable="false"
          size={13}
        />
      )}
      {hasSplitLabel ? (
        <>
          <span className="status-badge-accessible">{label}</span>
          <span className="status-badge-visual" aria-hidden="true">
            <strong className="status-badge-main">{visibleMainLabel}</strong>
            {secondaryLabel !== undefined && (
              <>
                <span className="status-badge-divider">｜</span>
                <span className="status-badge-detail">{secondaryLabel}</span>
              </>
            )}
          </span>
        </>
      ) : (
        label
      )}
    </span>
  );
}
