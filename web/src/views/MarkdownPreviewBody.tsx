import { useTranslation } from "react-i18next";
import ReactMarkdown from "react-markdown";
import { formatError } from "../i18n/domain";

interface MarkdownPreviewBodyProps {
  content: string | undefined;
  error: Error | undefined;
  onRetry: () => void;
}

export function MarkdownPreviewBody({
  content,
  error,
  onRetry,
}: MarkdownPreviewBodyProps) {
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
        </article>
      )}
    </div>
  );
}
