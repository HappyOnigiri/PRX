import { Check, Eye, Pencil, Plus, Trash2, X } from "lucide-react";
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
  featureId?: string;
  taskId?: string;
  documents: TaskNodeDocument[];
  onPreview: (document: TaskNodeDocument) => void;
  readOnly?: boolean;
  compact?: boolean;
}

export function ReferencesSection({
  featureId,
  taskId,
  documents,
  onPreview,
  readOnly = false,
  compact = false,
}: ReferencesSectionProps) {
  const { t } = useTranslation();
  const addDocument = useDomainMutation(mutations.addDocument);
  const deleteDocument = useDomainMutation(mutations.deleteDocument);
  const updateDocument = useDomainMutation(mutations.updateDocument);
  const [selectedKind, setSelectedKind] = useState(DocumentKind.URL);
  const [editing, setEditing] = useState<{
    id: string;
    content: string;
  } | null>(null);
  const ordered = [...documents].sort(
    (left, right) =>
      Number(right.isImplementationPlan) - Number(left.isImplementationPlan),
  );

  async function editMarkdown(document: TaskNodeDocument) {
    try {
      const response = await mutations.getDocument(document.id);
      setEditing({ id: document.id, content: response.content });
    } catch {
      // MutationError renders update failures; read errors keep the document unchanged.
    }
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
            onDelete={(id) => {
              deleteDocument.mutate(id);
            }}
            onEdit={(document) => {
              void editMarkdown(document);
            }}
            readOnly={readOnly}
          />
        ),
      )}
      {documents.length === 0 && readOnly && (
        <p className="read-only-empty">{t("inspector.noReferences")}</p>
      )}
      {!readOnly && (
        <form
          className="stack-form"
          onSubmit={(event) => {
            event.preventDefault();
            const form = new FormData(event.currentTarget);
            const kind = Number(form.get("kind"));
            const input: Parameters<typeof mutations.addDocument>[0] = {
              kind,
              title: formValue(form, "title"),
              value: formValue(form, "value"),
              isImplementationPlan:
                taskId !== undefined && form.get("implementationPlan") === "on",
            };
            if (featureId !== undefined) input.featureId = featureId;
            if (taskId !== undefined) input.taskId = taskId;
            addDocument.mutate(input);
          }}
        >
          <DocumentSourceFields
            kind={selectedKind}
            compact={compact}
            onKindChange={setSelectedKind}
          />
          {taskId && (
            <label className="document-plan-toggle">
              <input name="implementationPlan" type="checkbox" />
              {t("inspector.implementationPlan")}
            </label>
          )}
          <IconButton
            icon={Plus}
            label={t("inspector.addReference")}
            variant="primary"
            type="submit"
          />
        </form>
      )}
      {!readOnly && (
        <>
          <MutationError error={addDocument.error} />
          <MutationError error={deleteDocument.error} />
          <MutationError error={updateDocument.error} />
        </>
      )}
    </section>
  );
}

function MarkdownEditForm({
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

function DocumentSourceFields({
  kind,
  compact,
  onKindChange,
}: {
  kind: DocumentKind;
  compact: boolean;
  onKindChange: (kind: DocumentKind) => void;
}) {
  const { t } = useTranslation();
  return (
    <>
      <div className="form-row">
        <select
          name="kind"
          value={kind}
          onChange={(event) => {
            onKindChange(Number(event.currentTarget.value));
          }}
        >
          <option value={DocumentKind.URL}>
            {documentKindLabel(DocumentKind.URL, t)}
          </option>
          <option value={DocumentKind.LOCAL_FILE}>
            {documentKindLabel(DocumentKind.LOCAL_FILE, t)}
          </option>
          <option value={DocumentKind.MARKDOWN}>
            {documentKindLabel(DocumentKind.MARKDOWN, t)}
          </option>
        </select>
        <input name="title" placeholder={t("inspector.designNotes")} />
      </div>
      {kind === DocumentKind.MARKDOWN ? (
        <textarea
          name="value"
          required
          rows={compact ? 2 : 4}
          placeholder={t("inspector.referenceMarkdown")}
        />
      ) : (
        <input
          name="value"
          required
          type={kind === DocumentKind.URL ? "url" : "text"}
          placeholder={
            kind === DocumentKind.URL
              ? t("inspector.referenceUrl")
              : t("inspector.referencePath")
          }
        />
      )}
    </>
  );
}

function DocumentRow({
  document,
  onPreview,
  onDelete,
  onEdit,
  readOnly,
}: {
  document: TaskNodeDocument;
  onPreview: (document: TaskNodeDocument) => void;
  onDelete: (id: string) => void;
  onEdit: (document: TaskNodeDocument) => void;
  readOnly: boolean;
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
      {!readOnly && (
        <>
          {document.kind === DocumentKind.MARKDOWN && (
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
          <IconButton
            icon={Trash2}
            label={t("inspector.deleteReference", {
              title: document.title || t("inspector.referenceFallback"),
            })}
            variant="danger"
            size="compact"
            iconOnly
            onClick={() => {
              onDelete(document.id);
            }}
          />
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
