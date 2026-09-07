// One badge states a status wherever a task or a feature is listed, so the
// shape stays here and each caller only names the class that colours it. The
// badge carries no glyph: an icon names the kind of record, not its state.
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
