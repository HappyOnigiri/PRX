// タスクや feature が並ぶ場所ではこのバッジ 1 つが状態を示す。形はここに集約
// し、呼び出し元は色付けのクラス名だけを指定する。バッジにグリフは付けない。
// アイコンが示すのはレコードの状態ではなく種別だから。
export function StatusBadge({
  className,
  label,
  title,
}: {
  className?: string;
  label: string;
  title?: string;
}) {
  return (
    <span
      className={className ? `status-badge ${className}` : "status-badge"}
      title={title}
    >
      {label}
    </span>
  );
}
