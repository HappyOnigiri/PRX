import { ChevronDown, Plus } from "lucide-react";
import { useEffect, useId, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { mutations } from "../api";
import { useDomainMutation } from "../hooks";
import { AddDocumentDialog } from "./AddDocumentDialog";
import { IconButton } from "./IconButton";
import { MutationError } from "./MutationError";
import { DocumentRow, MarkdownEditForm } from "./TaskInspectorReferences";
import type { TaskNodeDocument } from "./TaskNode";

interface FeatureReferencesProps {
  featureId: string;
  documents: TaskNodeDocument[];
  onPreview: (document: TaskNodeDocument) => void;
  readOnly?: boolean;
}

export function FeatureReferences({
  featureId,
  documents,
  onPreview,
  readOnly = false,
}: FeatureReferencesProps) {
  const { t } = useTranslation();
  const deleteDocument = useDomainMutation(mutations.deleteDocument);
  const updateDocument = useDomainMutation(mutations.updateDocument);
  const getDocument = useDomainMutation(mutations.getDocument);
  const [open, setOpen] = useState(false);
  const [showAddDialog, setShowAddDialog] = useState(false);
  const [addTrigger, setAddTrigger] = useState<HTMLElement | null>(null);
  const [editing, setEditing] = useState<{
    id: string;
    content: string;
  } | null>(null);
  const rootRef = useRef<HTMLDivElement>(null);
  const triggerRef = useRef<HTMLButtonElement>(null);
  const triggerId = useId();
  const panelId = useId();

  useEffect(() => {
    if (!open) return;
    const closeOnOutsidePointer = (event: PointerEvent) => {
      if (
        event.target instanceof Node &&
        !rootRef.current?.contains(event.target)
      )
        setOpen(false);
    };
    const closeOnEscape = (event: KeyboardEvent) => {
      if (event.key !== "Escape") return;
      setOpen(false);
      triggerRef.current?.focus();
    };
    document.addEventListener("pointerdown", closeOnOutsidePointer);
    document.addEventListener("keydown", closeOnEscape);
    return () => {
      document.removeEventListener("pointerdown", closeOnOutsidePointer);
      document.removeEventListener("keydown", closeOnEscape);
    };
  }, [open]);

  const ordered = [...documents].sort(
    (left, right) =>
      Number(right.isImplementationPlan) - Number(left.isImplementationPlan),
  );

  function editMarkdown(document: TaskNodeDocument) {
    getDocument.mutate(document.id, {
      onSuccess: (response) => {
        setEditing({ id: document.id, content: response.content });
      },
    });
  }

  return (
    <div className="feature-references" ref={rootRef}>
      <IconButton
        ref={triggerRef}
        id={triggerId}
        icon={ChevronDown}
        label={t("workspace.references")}
        className="feature-references-trigger"
        variant="secondary"
        aria-expanded={open}
        aria-controls={panelId}
        onClick={() => {
          setOpen((current) => !current);
        }}
      />
      {open && (
        <FeatureReferencesPanel
          id={panelId}
          labelledBy={triggerId}
          documents={ordered}
          editing={editing}
          readOnly={readOnly}
          errors={[
            deleteDocument.error,
            updateDocument.error,
            getDocument.error,
          ]}
          onPreview={(selected) => {
            setOpen(false);
            onPreview(selected);
          }}
          onDelete={(id) => {
            deleteDocument.mutate(id);
          }}
          onEdit={editMarkdown}
          onUpdate={(document, content) => {
            updateDocument.mutate({
              id: document.id,
              source: { case: "markdown", value: content },
            });
            setEditing(null);
          }}
          onCancelEdit={() => {
            setEditing(null);
          }}
          onAdd={() => {
            setOpen(false);
            setEditing(null);
            setAddTrigger(triggerRef.current);
            setShowAddDialog(true);
          }}
        />
      )}
      {showAddDialog && (
        <AddDocumentDialog
          featureId={featureId}
          trigger={addTrigger}
          onClose={() => {
            setShowAddDialog(false);
          }}
        />
      )}
    </div>
  );
}

function FeatureReferencesPanel({
  id,
  labelledBy,
  documents,
  editing,
  readOnly,
  errors,
  onPreview,
  onDelete,
  onEdit,
  onUpdate,
  onCancelEdit,
  onAdd,
}: {
  id: string;
  labelledBy: string;
  documents: TaskNodeDocument[];
  editing: { id: string; content: string } | null;
  readOnly: boolean;
  errors: (Error | null)[];
  onPreview: (document: TaskNodeDocument) => void;
  onDelete: (id: string) => void;
  onEdit: (document: TaskNodeDocument) => void;
  onUpdate: (document: TaskNodeDocument, content: string) => void;
  onCancelEdit: () => void;
  onAdd: () => void;
}) {
  const { t } = useTranslation();
  return (
    <section
      id={id}
      className="feature-references-panel"
      aria-labelledby={labelledBy}
    >
      <div className="feature-references-list">
        {documents.map((document) =>
          editing?.id === document.id ? (
            <MarkdownEditForm
              key={document.id}
              document={document}
              content={editing.content}
              compact
              onSubmit={(content) => {
                onUpdate(document, content);
              }}
              onCancel={onCancelEdit}
            />
          ) : (
            <DocumentRow
              key={document.id}
              document={document}
              onPreview={onPreview}
              onDelete={onDelete}
              onEdit={onEdit}
              canEdit={!readOnly}
              canDelete={!readOnly}
            />
          ),
        )}
        {documents.length === 0 && (
          <p className="read-only-empty">{t("inspector.noReferences")}</p>
        )}
        {errors.map((error, index) => (
          <MutationError key={index} error={error} />
        ))}
      </div>
      {!readOnly && (
        <div className="feature-references-footer">
          <IconButton
            icon={Plus}
            label={t("workspace.addReference")}
            variant="quiet"
            onClick={onAdd}
          />
        </div>
      )}
    </section>
  );
}
