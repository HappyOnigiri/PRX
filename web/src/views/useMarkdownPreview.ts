import { useEffect, useRef, useState, type RefObject } from "react";
import { readMarkdownDocument } from "../api";
import type {
  MarkdownCopy,
  MarkdownCopyKind,
  MarkdownCopyStatus,
} from "./MarkdownPreviewTypes";

interface MarkdownPreviewOptions {
  documentId: string;
  onClose: () => void;
}

interface MarkdownPreviewState {
  content: string | undefined;
  error: Error | undefined;
  copyStatus: MarkdownCopyStatus | undefined;
  closeButton: RefObject<HTMLButtonElement | null>;
  copy: MarkdownCopy;
  retry: () => void;
}

export function useMarkdownPreview({
  documentId,
  onClose,
}: MarkdownPreviewOptions): MarkdownPreviewState {
  const [content, setContent] = useState<string>();
  const [error, setError] = useState<Error>();
  const [attempt, setAttempt] = useState(0);
  const [copyStatus, setCopyStatus] = useState<MarkdownCopyStatus>();
  const closeButton = useRef<HTMLButtonElement>(null);

  useEffect(() => {
    let current = true;
    readMarkdownDocument(documentId).then(
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
  }, [attempt, documentId]);

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

  const copy: MarkdownCopy = async (value: string, kind: MarkdownCopyKind) => {
    try {
      await navigator.clipboard.writeText(value);
      setCopyStatus(kind);
    } catch {
      setCopyStatus("failed");
    }
    window.setTimeout(() => {
      setCopyStatus(undefined);
    }, 1600);
  };

  function retry() {
    setContent(undefined);
    setError(undefined);
    setAttempt((value) => value + 1);
  }

  return { content, error, copyStatus, closeButton, copy, retry };
}
