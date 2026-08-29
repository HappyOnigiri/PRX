export interface MarkdownDocument {
  id: string;
  title: string;
  value: string;
}

export type MarkdownCopyKind = "content" | "path";
export type MarkdownCopyStatus = MarkdownCopyKind | "failed";
export type MarkdownCopy = (
  value: string,
  kind: MarkdownCopyKind,
) => Promise<void>;
