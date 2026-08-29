import { useTranslation } from "react-i18next";
import { mutations } from "../api";
import { formValue } from "../form";
import { DocumentKind } from "../gen/prx/v1/prx_pb";
import { useDomainMutation } from "../hooks";
import { documentKindLabel } from "../i18n/domain";
import { MutationError } from "./MutationError";
import { type TaskNodeDocument } from "./TaskNode";

interface ReferencesSectionProps {
  taskId: string;
  documents: TaskNodeDocument[];
  onPreview: (document: TaskNodeDocument) => void;
}

export function ReferencesSection({
  taskId,
  documents,
  onPreview,
}: ReferencesSectionProps) {
  const { t } = useTranslation();
  const addDocument = useDomainMutation(mutations.addDocument);
  const deleteDocument = useDomainMutation(mutations.deleteDocument);

  return (
    <section>
      <h3>{t("inspector.references")}</h3>
      {documents.map((document) => (
        <div className="document-chip" key={document.id}>
          {document.kind === DocumentKind.URL ? (
            <a href={document.value} target="_blank" rel="noreferrer">
              <b>{document.title || documentKindLabel(document.kind, t)}</b>
              <small>{document.value}</small>
            </a>
          ) : (
            <button
              className="document-preview"
              onClick={() => {
                onPreview(document);
              }}
            >
              <b>{document.title || documentKindLabel(document.kind, t)}</b>
              <small>{document.value}</small>
            </button>
          )}
          <button
            aria-label={t("inspector.deleteReference", {
              title: document.title || t("inspector.referenceFallback"),
            })}
            onClick={() => {
              deleteDocument.mutate(document.id);
            }}
          >
            ×
          </button>
        </div>
      ))}
      <form
        className="stack-form"
        onSubmit={(event) => {
          event.preventDefault();
          const form = new FormData(event.currentTarget);
          addDocument.mutate({
            taskId,
            kind: Number(form.get("kind")),
            title: formValue(form, "title"),
            value: formValue(form, "value"),
          });
        }}
      >
        <div className="form-row">
          <select name="kind">
            <option value={DocumentKind.URL}>
              {documentKindLabel(DocumentKind.URL, t)}
            </option>
            <option value={DocumentKind.MARKDOWN_PATH}>
              {documentKindLabel(DocumentKind.MARKDOWN_PATH, t)}
            </option>
          </select>
          <input name="title" placeholder={t("inspector.designNotes")} />
        </div>
        <input
          name="value"
          required
          placeholder={t("inspector.referenceValue")}
        />
        <button>{t("inspector.addReference")}</button>
      </form>
      <MutationError error={addDocument.error} />
      <MutationError error={deleteDocument.error} />
    </section>
  );
}
