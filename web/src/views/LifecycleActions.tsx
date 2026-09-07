import { Archive, ArchiveRestore, Trash2 } from "lucide-react";
import { IconButton } from "./IconButton";

// アーカイブと削除のブロックは project と feature で同一なので、文言は固定の
// namespace から読まずに props で受け取る。
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
  // 渡されるのは最大 1 つ。コンテナがアーカイブ済みでレコードが読み取り専用の
  // ときは両方とも省く。アーカイブも復元もサーバーが受け付ける書き込みには
  // ならないため。
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
