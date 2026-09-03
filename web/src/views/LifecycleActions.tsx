import { Archive, ArchiveRestore, Trash2 } from "lucide-react";
import { IconButton } from "./IconButton";

// The archive-or-delete block is identical for a project and a feature, so the
// wording arrives as props rather than being read from a fixed namespace.
export interface LifecycleLabels {
  section: string;
  detail: string;
  archive: string;
  restore: string;
  remove: string;
}

interface LifecycleActionsProps {
  labels: LifecycleLabels;
  updatePending: boolean;
  deletePending: boolean;
  // At most one of these is supplied. Both are omitted when the record is
  // read-only because its container is archived: neither archiving nor
  // restoring it would be a write the server accepts.
  onArchive?: () => void;
  onRestore?: () => void;
  onDelete: () => void;
}

export function LifecycleActions({
  labels,
  updatePending,
  deletePending,
  onArchive,
  onRestore,
  onDelete,
}: LifecycleActionsProps) {
  return (
    <section className="feature-lifecycle" aria-label={labels.section}>
      <div>
        <h3 className="feature-lifecycle-title">{labels.section}</h3>
        <p className="feature-lifecycle-detail">{labels.detail}</p>
      </div>
      <div className="feature-lifecycle-actions">
        {onRestore && (
          <IconButton
            icon={ArchiveRestore}
            label={labels.restore}
            variant="primary"
            disabled={updatePending}
            onClick={onRestore}
          />
        )}
        {onArchive && (
          <IconButton
            icon={Archive}
            label={labels.archive}
            variant="secondary"
            disabled={updatePending}
            onClick={onArchive}
          />
        )}
        <IconButton
          icon={Trash2}
          label={labels.remove}
          variant="danger"
          disabled={deletePending}
          onClick={onDelete}
        />
      </div>
    </section>
  );
}
