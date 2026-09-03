import { useMutation } from "@tanstack/react-query";
import { FileUp, Link, Plus, Text, X } from "lucide-react";
import {
  useEffect,
  useId,
  useRef,
  useState,
  type KeyboardEvent,
  type ReactNode,
  type RefObject,
  type SyntheticEvent,
} from "react";
import { useTranslation } from "react-i18next";
import { mutations, selectLocalFile } from "../api";
import type { DocumentParent } from "../document-parent";
import { DocumentKind } from "../gen/prx/v1/prx_pb";
import { useDomainMutation } from "../hooks";
import { IconButton } from "./IconButton";
import { MutationError } from "./MutationError";

const documentTabs = [
  { kind: DocumentKind.URL, key: "url", icon: Link },
  { kind: DocumentKind.LOCAL_FILE, key: "localFile", icon: FileUp },
  { kind: DocumentKind.MARKDOWN, key: "markdown", icon: Text },
] as const;

type AddDocumentDialogProps = DocumentParent & {
  trigger: HTMLElement | null;
  onClose: () => void;
};

function useDialogState(props: AddDocumentDialogProps) {
  const addDocument = useDomainMutation(mutations.addDocument);
  const filePicker = useMutation({ mutationFn: selectLocalFile });
  const [kind, setKind] = useState<DocumentKind>(DocumentKind.URL);
  const [title, setTitle] = useState("");
  const [values, setValues] = useState<Partial<Record<DocumentKind, string>>>({
    [DocumentKind.URL]: "",
    [DocumentKind.LOCAL_FILE]: "",
    [DocumentKind.MARKDOWN]: "",
  });
  const [implementationPlan, setImplementationPlan] = useState(false);
  const [pickerCanceled, setPickerCanceled] = useState(false);
  const titleId = useId();
  const idPrefix = useId();
  const busy = addDocument.isPending || filePicker.isPending;

  async function chooseFile() {
    setPickerCanceled(false);
    try {
      const result = await filePicker.mutateAsync();
      if (result.canceled) {
        setPickerCanceled(true);
        return;
      }
      setValues((current) => ({
        ...current,
        [DocumentKind.LOCAL_FILE]: result.path,
      }));
    } catch {
      return;
    }
  }

  async function submit(event: SyntheticEvent<HTMLFormElement>) {
    event.preventDefault();
    const input: Parameters<typeof mutations.addDocument>[0] = {
      title,
      kind,
      value: values[kind] ?? "",
    };
    if (props.projectId !== undefined) input.projectId = props.projectId;
    if (props.featureId !== undefined) input.featureId = props.featureId;
    if (props.taskId !== undefined) {
      input.taskId = props.taskId;
      input.isImplementationPlan = implementationPlan;
    }
    try {
      await addDocument.mutateAsync(input);
    } catch {
      return;
    }
    props.onClose();
  }

  return {
    ...props,
    addDocument,
    busy,
    chooseFile,
    filePicker,
    idPrefix,
    implementationPlan,
    kind,
    pickerCanceled,
    setImplementationPlan,
    setKind,
    setPickerCanceled,
    setTitle,
    setValues,
    submit,
    title,
    titleId,
    values,
  };
}

type DialogState = ReturnType<typeof useDialogState>;

export function AddDocumentDialog(props: AddDocumentDialogProps) {
  const { t } = useTranslation();
  const state = useDialogState(props);
  const dialogRef = useRef<HTMLFormElement>(null);
  const tabRefs = useRef<(HTMLButtonElement | null)[]>([]);

  useEffect(() => {
    tabRefs.current[0]?.focus();
    return () => {
      window.setTimeout(() => props.trigger?.focus());
    };
  }, [props.trigger]);

  function trapFocus(event: KeyboardEvent<HTMLDivElement>) {
    if (event.key === "Escape" && !state.busy) {
      props.onClose();
      return;
    }
    if (event.key !== "Tab") return;
    const focusable = Array.from(
      dialogRef.current?.querySelectorAll<HTMLElement>(
        "button:not(:disabled), input:not(:disabled), textarea:not(:disabled), a[href]",
      ) ?? [],
    ).filter((element) => !element.closest("[hidden]"));
    const first = focusable[0];
    const last = focusable.at(-1);
    if (!first || !last) {
      event.preventDefault();
      return;
    }
    if (event.shiftKey && document.activeElement === first) {
      event.preventDefault();
      last.focus();
    } else if (!event.shiftKey && document.activeElement === last) {
      event.preventDefault();
      first.focus();
    }
  }
  return (
    <div
      className="scrim document-dialog-scrim"
      role="presentation"
      onKeyDown={trapFocus}
    >
      <form
        ref={dialogRef}
        className="dialog document-dialog"
        role="dialog"
        aria-modal="true"
        aria-labelledby={state.titleId}
        onSubmit={state.submit}
      >
        <DialogHeader state={state} />
        <label className="document-dialog-title-field">
          {t("documentDialog.titleLabel")}
          <input
            name="title"
            placeholder={t("documentDialog.titlePlaceholder")}
            value={state.title}
            onChange={(event) => {
              state.setTitle(event.currentTarget.value);
            }}
          />
        </label>
        <DocumentSourceTabs state={state} tabRefs={tabRefs} />
        {state.taskId !== undefined && (
          <label className="document-plan-toggle">
            <input
              checked={state.implementationPlan}
              type="checkbox"
              onChange={(event) => {
                state.setImplementationPlan(event.currentTarget.checked);
              }}
            />
            {t("documentDialog.implementationPlan")}
          </label>
        )}
        <MutationError error={state.addDocument.error} />
        <DialogFooter state={state} />
      </form>
    </div>
  );
}

// The heading names the parent the document will belong to, so the three
// entry points do not each need their own dialog.
function documentDialogTitleKey(state: DialogState) {
  if (state.projectId !== undefined) return "documentDialog.projectTitle";
  if (state.featureId !== undefined) return "documentDialog.featureTitle";
  return "documentDialog.taskTitle";
}

function DialogHeader({ state }: { state: DialogState }) {
  const { t } = useTranslation();
  return (
    <header className="document-dialog-head">
      <div>
        <h2 id={state.titleId}>{t(documentDialogTitleKey(state))}</h2>
        <p className="dialog-lead">{t("documentDialog.description")}</p>
      </div>
      <IconButton
        icon={X}
        label={t("common.close")}
        variant="secondary"
        iconOnly
        disabled={state.busy}
        onClick={state.onClose}
      />
    </header>
  );
}

function DocumentSourceTabs({
  state,
  tabRefs,
}: {
  state: DialogState;
  tabRefs: RefObject<(HTMLButtonElement | null)[]>;
}) {
  const { t } = useTranslation();
  function selectTab(index: number) {
    const tab = documentTabs[index];
    if (!tab) return;
    state.setKind(tab.kind);
    tabRefs.current[index]?.focus();
  }
  function handleKey(event: KeyboardEvent<HTMLButtonElement>) {
    const current = documentTabs.findIndex((tab) => tab.kind === state.kind);
    let next: number | undefined;
    if (event.key === "ArrowRight") next = (current + 1) % documentTabs.length;
    if (event.key === "ArrowLeft")
      next = (current - 1 + documentTabs.length) % documentTabs.length;
    if (event.key === "Home") next = 0;
    if (event.key === "End") next = documentTabs.length - 1;
    if (next === undefined) return;
    event.preventDefault();
    selectTab(next);
  }
  return (
    <>
      <div className="document-tabs" role="tablist">
        {documentTabs.map((tab, index) => {
          const TabIcon = tab.icon;
          const active = tab.kind === state.kind;
          return (
            <button
              aria-controls={`${state.idPrefix}-panel-${tab.key}`}
              aria-selected={active}
              className="document-tab"
              id={`${state.idPrefix}-tab-${tab.key}`}
              key={tab.key}
              onClick={() => {
                state.setKind(tab.kind);
              }}
              onKeyDown={handleKey}
              ref={(element) => {
                tabRefs.current[index] = element;
              }}
              role="tab"
              tabIndex={active ? 0 : -1}
              type="button"
            >
              <TabIcon aria-hidden="true" focusable="false" size={16} />
              {t(`documentDialog.tabs.${tab.key}`)}
            </button>
          );
        })}
      </div>
      <URLPanel state={state} />
      <LocalFilePanel state={state} />
      <MarkdownPanel state={state} />
    </>
  );
}

function TabPanel({
  children,
  state,
  tab,
}: {
  children: ReactNode;
  state: DialogState;
  tab: (typeof documentTabs)[number];
}) {
  return (
    <div
      aria-labelledby={`${state.idPrefix}-tab-${tab.key}`}
      className={`document-tab-panel document-tab-panel-${tab.key}`}
      hidden={state.kind !== tab.kind}
      id={`${state.idPrefix}-panel-${tab.key}`}
      role="tabpanel"
      tabIndex={0}
    >
      {children}
    </div>
  );
}

function setSourceValue(state: DialogState, kind: DocumentKind, value: string) {
  state.setValues((current) => ({ ...current, [kind]: value }));
}

function URLPanel({ state }: { state: DialogState }) {
  const { t } = useTranslation();
  const tab = documentTabs[0];
  return (
    <TabPanel state={state} tab={tab}>
      <label>
        {t("documentDialog.urlLabel")}
        <input
          disabled={state.kind !== tab.kind}
          required
          type="url"
          placeholder={t("documentDialog.urlPlaceholder")}
          value={state.values[tab.kind] ?? ""}
          onChange={(event) => {
            setSourceValue(state, tab.kind, event.currentTarget.value);
          }}
        />
      </label>
    </TabPanel>
  );
}

function LocalFilePanel({ state }: { state: DialogState }) {
  const { t } = useTranslation();
  const tab = documentTabs[1];
  return (
    <TabPanel state={state} tab={tab}>
      <label>
        {t("documentDialog.pathLabel")}
        <input
          disabled={state.kind !== tab.kind}
          required
          type="text"
          placeholder={t("documentDialog.pathPlaceholder")}
          value={state.values[tab.kind] ?? ""}
          onChange={(event) => {
            state.setPickerCanceled(false);
            setSourceValue(state, tab.kind, event.currentTarget.value);
          }}
        />
      </label>
      <div className="document-file-actions">
        <IconButton
          icon={FileUp}
          label={t(
            state.filePicker.isPending
              ? "documentDialog.choosingFile"
              : "documentDialog.chooseFile",
          )}
          variant="secondary"
          disabled={state.filePicker.isPending || state.addDocument.isPending}
          onClick={() => void state.chooseFile()}
        />
        {state.pickerCanceled && (
          <span className="document-picker-status" role="status">
            {t("documentDialog.chooseCanceled")}
          </span>
        )}
      </div>
      <MutationError error={state.filePicker.error} />
    </TabPanel>
  );
}

function MarkdownPanel({ state }: { state: DialogState }) {
  const { t } = useTranslation();
  const tab = documentTabs[2];
  return (
    <TabPanel state={state} tab={tab}>
      <label className="document-markdown-field">
        {t("documentDialog.markdownLabel")}
        <textarea
          disabled={state.kind !== tab.kind}
          required
          placeholder={t("documentDialog.markdownPlaceholder")}
          value={state.values[tab.kind] ?? ""}
          onChange={(event) => {
            setSourceValue(state, tab.kind, event.currentTarget.value);
          }}
        />
      </label>
    </TabPanel>
  );
}

function DialogFooter({ state }: { state: DialogState }) {
  const { t } = useTranslation();
  return (
    <footer>
      <IconButton
        icon={X}
        label={t("common.cancel")}
        variant="secondary"
        disabled={state.busy}
        onClick={state.onClose}
      />
      <IconButton
        icon={Plus}
        label={t(
          state.addDocument.isPending
            ? "documentDialog.submitting"
            : "documentDialog.submit",
        )}
        variant="primary"
        type="submit"
        disabled={state.busy}
      />
    </footer>
  );
}
