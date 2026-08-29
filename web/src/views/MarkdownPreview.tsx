import { useTranslation } from "react-i18next";
import { MarkdownPreviewBody } from "./MarkdownPreviewBody";
import { MarkdownPreviewHeader } from "./MarkdownPreviewHeader";
import type { MarkdownDocument } from "./MarkdownPreviewTypes";
import { useMarkdownPreview } from "./useMarkdownPreview";

export interface MarkdownPreviewProps {
  document: MarkdownDocument;
  onClose: () => void;
}

export function MarkdownPreview({ document, onClose }: MarkdownPreviewProps) {
  const { t } = useTranslation();
  const { content, error, copyStatus, closeButton, copy, retry } =
    useMarkdownPreview({ documentId: document.id, onClose });

  return (
    <div className="scrim markdown-scrim">
      <section
        className="markdown-dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="markdown-preview-title"
      >
        <MarkdownPreviewHeader
          document={document}
          content={content}
          copy={copy}
          onClose={onClose}
          closeButton={closeButton}
        />
        <MarkdownPreviewBody content={content} error={error} onRetry={retry} />
        <p className="copy-status" aria-live="polite">
          {copyStatus === "content" && t("markdownPreview.contentCopied")}
          {copyStatus === "path" && t("markdownPreview.pathCopied")}
          {copyStatus === "failed" && t("markdownPreview.copyFailed")}
        </p>
      </section>
    </div>
  );
}
