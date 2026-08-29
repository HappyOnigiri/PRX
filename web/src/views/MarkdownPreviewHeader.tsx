import type { RefObject } from "react";
import { useTranslation } from "react-i18next";
import type { MarkdownCopy, MarkdownDocument } from "./MarkdownPreviewTypes";

interface MarkdownPreviewHeaderProps {
  document: MarkdownDocument;
  content: string | undefined;
  copy: MarkdownCopy;
  onClose: () => void;
  closeButton: RefObject<HTMLButtonElement | null>;
}

export function MarkdownPreviewHeader({
  document,
  content,
  copy,
  onClose,
  closeButton,
}: MarkdownPreviewHeaderProps) {
  const { t } = useTranslation();
  return (
    <header>
      <div>
        <h2 id="markdown-preview-title">
          {document.title || t("markdownPreview.untitled")}
        </h2>
        <code>{document.value}</code>
      </div>
      <div className="markdown-actions">
        <button
          className="secondary"
          disabled={content === undefined}
          onClick={() => void copy(content ?? "", "content")}
        >
          {t("markdownPreview.copyContent")}
        </button>
        <button
          className="secondary"
          onClick={() => void copy(document.value, "path")}
        >
          {t("markdownPreview.copyPath")}
        </button>
        <button
          className="icon-button"
          aria-label={t("markdownPreview.close")}
          onClick={onClose}
          ref={closeButton}
        >
          ×
        </button>
      </div>
    </header>
  );
}
