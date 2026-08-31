import { Check, Eye, Pencil, Trash2, X } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { mutations } from "../api";
import { formValue } from "../form";
import { DocumentKind } from "../gen/prx/v1/prx_pb";
import { useDomainMutation } from "../hooks";
import { documentKindLabel } from "../i18n/domain";
import { IconButton } from "./IconButton";
import { MutationError } from "./MutationError";
import { type TaskNodeDocument } from "./TaskNode";

interface ReferencesSectionProps {
  documents: TaskNodeDocument[];
  onPreview: (document: TaskNodeDocument) => void;
  readOnly?: boolean;
  compact?: boolean;
}

export function ReferencesSection({
  documents,
  onPreview,
  readOnly = false,
  compact = false,
}: ReferencesSectionProps) {
  const { t } = useTranslation();
  const updateDocument = useDomainMutation(mutations.updateDocument);
  const getDocument = useDomainMutation(mutations.getDocument);
  const [editing, setEditing] = useState<{
    id: string;
    content: string;
  } | null>(null);
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
    <section
      className={compact ? "documents-section is-compact" : "documents-section"}
    >
      <h3>{t("inspector.references")}</h3>
      {ordered.map((document) =>
        editing?.id === document.id ? (
          <MarkdownEditForm
            key={document.id}
            document={document}
            content={editing.content}
            compact={compact}
            onSubmit={(content) => {
              updateDocument.mutate({
                id: document.id,
                source: { case: "markdown", value: content },
              });
              setEditing(null);
            }}
            onCancel={() => {
              setEditing(null);
            }}
          />
        ) : (
          <DocumentRow
            key={document.id}
            document={document}
            onPreview={onPreview}
            onEdit={editMarkdown}
            canEdit={!readOnly}
            canDelete={false}
          />
        ),
      )}
      {documents.length === 0 && (
        <p className="read-only-empty">{t("inspector.noReferences")}</p>
      )}
      {!readOnly && (
        <>
          <MutationError error={updateDocument.error} />
          <MutationError error={getDocument.error} />
        </>
      )}
    </section>
  );
}

export function MarkdownEditForm({
  document,
  content,
  compact,
  onSubmit,
  onCancel,
}: {
  document: TaskNodeDocument;
  content: string;
  compact: boolean;
  onSubmit: (content: string) => void;
  onCancel: () => void;
}) {
  const { t } = useTranslation();
  const title = document.title || documentKindLabel(document.kind, t);
  return (
    <form
      className="stack-form"
      onSubmit={(event) => {
        event.preventDefault();
        const form = new FormData(event.currentTarget);
        onSubmit(formValue(form, "content"));
      }}
    >
      <textarea
        name="content"
        aria-label={t("inspector.editReference", { title })}
        defaultValue={content}
        rows={compact ? 4 : 8}
      />
      <div className="form-row">
        <IconButton
          icon={Check}
          label={t("inspector.saveReference")}
          variant="primary"
          type="submit"
        />
        <IconButton
          icon={X}
          label={t("inspector.cancelReferenceEdit")}
          variant="secondary"
          onClick={onCancel}
        />
      </div>
    </form>
  );
}

export function DocumentRow({
  document,
  onPreview,
  onDelete,
  onEdit,
  canEdit,
  canDelete,
}: {
  document: TaskNodeDocument;
  onPreview: (document: TaskNodeDocument) => void;
  onDelete?: (id: string) => void;
  onEdit: (document: TaskNodeDocument) => void;
  canEdit: boolean;
  canDelete: boolean;
}) {
  const { t } = useTranslation();
  const title = document.title || documentKindLabel(document.kind, t);
  return (
    <div
      className={`document-chip ${document.isImplementationPlan ? "is-plan" : ""}`}
    >
      {document.kind === DocumentKind.URL ? (
        <a href={document.locator} target="_blank" rel="noreferrer">
          <DocumentTitle title={title} plan={document.isImplementationPlan} />
          <small>{document.locator}</small>
        </a>
      ) : (
        <button
          type="button"
          className="document-preview"
          onClick={() => {
            onPreview(document);
          }}
        >
          <span className="document-preview-copy">
            <DocumentTitle title={title} plan={document.isImplementationPlan} />
            <small>
              {document.locator || documentKindLabel(document.kind, t)}
            </small>
          </span>
          <Eye aria-hidden="true" focusable="false" size={14} />
        </button>
      )}
      {(canEdit || canDelete) && (
        <>
          {canEdit && document.kind === DocumentKind.MARKDOWN && (
            <IconButton
              icon={Pencil}
              label={t("inspector.editReference", { title })}
              variant="secondary"
              size="compact"
              iconOnly
              onClick={() => {
                onEdit(document);
              }}
            />
          )}
          {canDelete && (
            <IconButton
              icon={Trash2}
              label={t("inspector.deleteReference", {
                title: document.title || t("inspector.referenceFallback"),
              })}
              variant="danger"
              size="compact"
              iconOnly
              onClick={() => {
                onDelete?.(document.id);
              }}
            />
          )}
        </>
      )}
    </div>
  );
}

function DocumentTitle({ title, plan }: { title: string; plan: boolean }) {
  return (
    <b>
      {plan && <span className="document-plan-label">PLAN</span>}
      {title}
    </b>
  );
}
