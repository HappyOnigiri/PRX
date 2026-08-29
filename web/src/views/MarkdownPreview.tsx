import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import ReactMarkdown from "react-markdown";
import { readMarkdownDocument } from "../api";
import { formatError } from "../i18n/domain";

interface MarkdownDocument {
  id: string;
  title: string;
  value: string;
}

export function MarkdownPreview({
  document,
  onClose,
}: {
  document: MarkdownDocument;
  onClose: () => void;
}) {
  const { t } = useTranslation();
  const [content, setContent] = useState<string>();
  const [error, setError] = useState<Error>();
  const [attempt, setAttempt] = useState(0);
  const [copyStatus, setCopyStatus] = useState<"content" | "path" | "failed">();
  const closeButton = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    let current = true;
    readMarkdownDocument(document.id).then(
      (value) => {
        if (current) setContent(value);
      },
      (reason: unknown) => {
        if (current)
          setError(
            reason instanceof Error ? reason : new Error(String(reason)),
          );
      },
    );
    return () => {
      current = false;
    };
  }, [attempt, document.id]);

  useEffect(() => closeButton.current?.focus(), []);

  useEffect(() => {
    function closeOnEscape(event: KeyboardEvent) {
      if (event.key === "Escape") onClose();
    }
    window.addEventListener("keydown", closeOnEscape);
    return () => {
      window.removeEventListener("keydown", closeOnEscape);
    };
  }, [onClose]);

  async function copy(value: string, kind: "content" | "path") {
    try {
      await navigator.clipboard.writeText(value);
      setCopyStatus(kind);
    } catch {
      setCopyStatus("failed");
    }
    window.setTimeout(() => {
      setCopyStatus(undefined);
    }, 1600);
  }

  function retry() {
    setContent(undefined);
    setError(undefined);
    setAttempt((value) => value + 1);
  }

  return (
    <div className="scrim markdown-scrim">
      <section
        className="markdown-dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby="markdown-preview-title"
      >
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
        <MarkdownBody content={content} error={error} onRetry={retry} />
        <p className="copy-status" aria-live="polite">
          {copyStatus === "content" && t("markdownPreview.contentCopied")}
          {copyStatus === "path" && t("markdownPreview.pathCopied")}
          {copyStatus === "failed" && t("markdownPreview.copyFailed")}
        </p>
      </section>
    </div>
  );
}

function MarkdownBody({
  content,
  error,
  onRetry,
}: {
  content: string | undefined;
  error: Error | undefined;
  onRetry: () => void;
}) {
  const { t } = useTranslation();
  return (
    <div className="markdown-preview-body">
      {content === undefined && !error && (
        <div className="preview-state" role="status">
          <div className="spinner" />
          <p>{t("markdownPreview.loading")}</p>
        </div>
      )}
      {error && (
        <div className="preview-state" role="alert">
          <p>{formatError(error, t)}</p>
          <button onClick={onRetry}>{t("common.retry")}</button>
        </div>
      )}
      {content !== undefined && (
        <article className="markdown-content">
          <MarkdownContent content={content} />
        </article>
      )}
    </div>
  );
}

export function MarkdownContent({ content }: { content: string }) {
  return (
    <ReactMarkdown
      components={{
        a: ({ children, ...props }) => (
          <a {...props} target="_blank" rel="noreferrer">
            {children}
          </a>
        ),
      }}
    >
      {content}
    </ReactMarkdown>
  );
}
