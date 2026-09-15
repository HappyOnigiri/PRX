import { create } from "@bufbuild/protobuf";
import type { TFunction } from "i18next";
import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { configMutations } from "../api";
import type {
  TaskLabelAppearances,
  TaskLabelOverrideUpdate,
  TaskLabelOverrides,
  TaskLabelOverridesUpdate,
} from "../gen/prx/v1/prx_pb";
import {
  TaskLabelOverrideUpdateSchema,
  TaskLabelOverridesUpdateSchema,
} from "../gen/prx/v1/prx_pb";
import { useTaskLabelConfig, useTaskLabelConfigMutation } from "../hooks";
import { StatusBadge } from "./StatusBadge";
import { useRegisterSettingsSection } from "./settingsSections";

type Scope = "global" | "project" | "feature";

interface TaskLabelEditorProps {
  scope: Scope;
  keys: string[];
  maxTextCodepoints: number;
  builtIn?: TaskLabelAppearances | undefined;
  globalOverrides?: TaskLabelOverrides | undefined;
  parentOverrides?: TaskLabelOverrides | undefined;
  overrides?: TaskLabelOverrides | undefined;
  editable: boolean;
  onStateChange?: (
    update: TaskLabelOverridesUpdate,
    dirty: boolean,
    invalid: boolean,
  ) => void;
}

interface DraftValue {
  text: string;
  color: string;
}

const emptyUpdate = create(TaskLabelOverridesUpdateSchema);

// Labels タブは global/project/feature で同じ検証とプレビューを使う。保存対象
// だけを各親へ返し、継承元の値を子 scope が誤って複製しないようにする。
function TaskLabelEditor({
  scope,
  keys,
  maxTextCodepoints,
  builtIn,
  globalOverrides,
  parentOverrides,
  overrides,
  editable,
  onStateChange,
}: TaskLabelEditorProps) {
  const { t } = useTranslation();
  const saved = useMemo(() => draftOf(overrides, keys), [keys, overrides]);
  const [draft, setDraft] = useState<Record<string, DraftValue>>(saved);
  const savedFingerprint = fingerprint(saved);
  const onStateChangeRef = useRef(onStateChange);
  const notifiedStateRef = useRef("");

  useEffect(() => {
    onStateChangeRef.current = onStateChange;
  }, [onStateChange]);

  // 保存後の query 更新や、別 scope の切替で受け取った値を下書きへ反映する。
  useEffect(() => {
    // eslint-disable-next-line react-hooks/set-state-in-effect -- 保存済み値が変わったときだけ、未編集の下書きを同期する。
    setDraft((current) =>
      fingerprint(current) === savedFingerprint ? saved : current,
    );
  }, [saved, savedFingerprint]);

  const validation = useMemo(
    () => validateDraft(draft, keys, maxTextCodepoints),
    [draft, keys, maxTextCodepoints],
  );
  const update = useMemo(
    () => changedUpdate(saved, draft, keys),
    [draft, keys, saved],
  );
  const dirty = Object.keys(update.values).length > 0;

  const updateFingerprint = useMemo(
    () => JSON.stringify(update.values),
    [update],
  );
  useEffect(() => {
    const stateFingerprint = `${dirty}:${validation.valid}:${updateFingerprint}`;
    if (notifiedStateRef.current === stateFingerprint) return;
    notifiedStateRef.current = stateFingerprint;
    onStateChangeRef.current?.(update, dirty, !validation.valid);
  }, [dirty, update, updateFingerprint, validation.valid]);

  const effective = useMemo(
    () =>
      resolveDraftAppearances({
        keys,
        builtIn,
        globalOverrides,
        parentOverrides,
        draft,
      }),
    [builtIn, draft, globalOverrides, keys, parentOverrides],
  );
  const statusKeys = keys.filter((key) => key.startsWith("status."));
  const blockKeys = keys.filter((key) => key.startsWith("block."));

  if (keys.length === 0)
    return <p className="settings-panel-state">{t("taskLabels.loading")}</p>;

  return (
    <div className="task-label-editor">
      <p className="dialog-lead">{t(`taskLabels.${scope}.description`)}</p>
      <LabelGroup
        title={t("taskLabels.statuses")}
        keys={statusKeys}
        draft={draft}
        saved={saved}
        effective={effective}
        editable={editable}
        maxTextCodepoints={maxTextCodepoints}
        onChange={setDraft}
      />
      <LabelGroup
        title={t("taskLabels.blocks")}
        keys={blockKeys}
        draft={draft}
        saved={saved}
        effective={effective}
        editable={editable}
        maxTextCodepoints={maxTextCodepoints}
        onChange={setDraft}
      />
      {!validation.valid && (
        <p className="form-error" role="alert">
          {validation.message}
        </p>
      )}
    </div>
  );
}

// eslint-disable-next-line max-lines-per-function -- status/block の同じ行レイアウトを保つため。
function LabelGroup({
  title,
  keys,
  draft,
  saved,
  effective,
  editable,
  maxTextCodepoints,
  onChange,
}: {
  title: string;
  keys: string[];
  draft: Record<string, DraftValue>;
  saved: Record<string, DraftValue>;
  effective: Map<string, EffectiveValue>;
  editable: boolean;
  maxTextCodepoints: number;
  onChange: (next: Record<string, DraftValue>) => void;
}) {
  const { t } = useTranslation();
  return (
    <section className="task-label-group">
      <h3>{title}</h3>
      <div className="task-label-list">
        {keys.map((key) => {
          const current = draft[key] ?? { text: "", color: "" };
          const previous = saved[key] ?? { text: "", color: "" };
          const value = effective.get(key) ?? {
            text: key,
            color: "",
            textSource: "built-in",
            colorSource: "built-in",
          };
          const textError = textValidationError(
            current.text,
            maxTextCodepoints,
          );
          const colorError = colorValidationError(current.color);
          return (
            <fieldset className="task-label-row" key={key}>
              <legend>
                <span>{taskLabelName(key, t)}</span>
                <code>{key}</code>
              </legend>
              <div className="task-label-fields">
                <label>
                  <span>{t("taskLabels.text")}</span>
                  <input
                    aria-label={`${taskLabelName(key, t)} ${t("taskLabels.text")}`}
                    disabled={!editable}
                    value={current.text}
                    onChange={(event) => {
                      onChange({
                        ...draft,
                        [key]: { ...current, text: event.target.value },
                      });
                    }}
                    placeholder={t("taskLabels.inherit")}
                  />
                  {textError && (
                    <small className="form-error">{textError}</small>
                  )}
                </label>
                <label>
                  <span>{t("taskLabels.color")}</span>
                  <input
                    aria-label={`${taskLabelName(key, t)} ${t("taskLabels.color")}`}
                    disabled={!editable}
                    value={current.color}
                    onChange={(event) => {
                      onChange({
                        ...draft,
                        [key]: { ...current, color: event.target.value },
                      });
                    }}
                    placeholder="#rrggbb"
                    inputMode="text"
                  />
                  {colorError && (
                    <small className="form-error">{colorError}</small>
                  )}
                </label>
                <div className="task-label-inheritance">
                  <span>{t("taskLabels.effective", { text: value.text })}</span>
                  <small>
                    {t("taskLabels.source", {
                      text: sourceName(value.textSource, t),
                      color: sourceName(value.colorSource, t),
                    })}
                  </small>
                </div>
                {editable && (
                  <div className="task-label-row-actions">
                    <button
                      type="button"
                      className="button secondary"
                      disabled={!current.text && !previous.text}
                      onClick={() => {
                        onChange({ ...draft, [key]: { ...current, text: "" } });
                      }}
                    >
                      {t("taskLabels.inheritText")}
                    </button>
                    <button
                      type="button"
                      className="button secondary"
                      disabled={!current.color && !previous.color}
                      onClick={() => {
                        onChange({
                          ...draft,
                          [key]: { ...current, color: "" },
                        });
                      }}
                    >
                      {t("taskLabels.inheritColor")}
                    </button>
                  </div>
                )}
              </div>
              <LabelPreview value={value} />
            </fieldset>
          );
        })}
      </div>
    </section>
  );
}

function LabelPreview({ value }: { value: EffectiveValue }) {
  const { t } = useTranslation();
  const warning =
    value.colorSource !== "built-in" &&
    value.color !== "" &&
    contrastWarning(value.color);
  return (
    <div className="task-label-preview">
      <span>{t("taskLabels.preview")}</span>
      <StatusBadge label={value.text} color={value.color || undefined} />
      {value.color && (
        <>
          <span className="task-label-preview-light">
            <StatusBadge label={value.text} color={value.color} />
          </span>
          <span className="task-label-preview-dark">
            <StatusBadge label={value.text} color={value.color} />
          </span>
        </>
      )}
      {warning && (
        <small className="form-warning">
          {t("taskLabels.contrastWarning")}
        </small>
      )}
    </div>
  );
}

interface EffectiveValue {
  text: string;
  color: string;
  textSource: string;
  colorSource: string;
}

function resolveDraftAppearances({
  keys,
  builtIn,
  globalOverrides,
  parentOverrides,
  draft,
}: {
  keys: string[];
  builtIn?: TaskLabelAppearances | undefined;
  globalOverrides?: TaskLabelOverrides | undefined;
  parentOverrides?: TaskLabelOverrides | undefined;
  draft: Record<string, DraftValue>;
}): Map<string, EffectiveValue> {
  const result = new Map<string, EffectiveValue>();
  for (const key of keys) {
    const own = draft[key];
    const parent = parentOverrides?.values[key];
    const global = globalOverrides?.values[key];
    const built = builtIn?.values.find((item) => item.key === key);
    const text =
      firstNonEmpty(own?.text, parent?.text, global?.text, built?.text) ?? key;
    const color =
      firstNonEmpty(own?.color, parent?.color, global?.color, built?.color) ??
      "";
    result.set(key, {
      text,
      color,
      textSource: own?.text
        ? "scope"
        : parent?.text
          ? "project"
          : global?.text
            ? "global"
            : "built-in",
      colorSource: own?.color
        ? "scope"
        : parent?.color
          ? "project"
          : global?.color
            ? "global"
            : "built-in",
    });
  }
  return result;
}

function draftOf(
  overrides: TaskLabelOverrides | undefined,
  keys: string[],
): Record<string, DraftValue> {
  return Object.fromEntries(
    keys.map((key) => {
      const value = overrides?.values[key];
      return [key, { text: value?.text ?? "", color: value?.color ?? "" }];
    }),
  );
}

function changedUpdate(
  saved: Record<string, DraftValue>,
  draft: Record<string, DraftValue>,
  keys: string[],
): TaskLabelOverridesUpdate {
  const values: Record<string, TaskLabelOverrideUpdate> = {};
  for (const key of keys) {
    const before = saved[key] ?? { text: "", color: "" };
    const after = draft[key] ?? { text: "", color: "" };
    const value = create(TaskLabelOverrideUpdateSchema);
    if (before.text !== after.text) value.text = after.text;
    if (before.color !== after.color) value.color = after.color;
    if (value.text !== undefined || value.color !== undefined)
      values[key] = value;
  }
  return create(TaskLabelOverridesUpdateSchema, { values });
}

function fingerprint(values: Record<string, DraftValue>): string {
  return JSON.stringify(
    Object.keys(values)
      .sort()
      .map((key) => [key, values[key]?.text ?? "", values[key]?.color ?? ""]),
  );
}

function validateDraft(
  draft: Record<string, DraftValue>,
  keys: string[],
  maxTextCodepoints: number,
): { valid: boolean; message: string } {
  for (const key of keys) {
    const textError = textValidationError(
      draft[key]?.text ?? "",
      maxTextCodepoints,
    );
    if (textError) return { valid: false, message: textError };
    const colorError = colorValidationError(draft[key]?.color ?? "");
    if (colorError) return { valid: false, message: colorError };
  }
  return { valid: true, message: "" };
}

function textValidationError(text: string, max: number): string {
  const trimmed = text.trim();
  if (Array.from(trimmed).length > max)
    return `Text must be ${max} characters or fewer.`;
  for (const character of trimmed) {
    const codePoint = character.codePointAt(0) ?? 0;
    if (codePoint <= 0x1f || (codePoint >= 0x7f && codePoint <= 0x9f))
      return "Text cannot contain control characters or newlines.";
  }
  return "";
}

function colorValidationError(color: string): string {
  if (!color || /^#[0-9a-f]{6}$/iu.test(color.trim())) return "";
  return "Color must use #RRGGBB.";
}

function contrastWarning(color: string): boolean {
  const rgb = hexRgb(color);
  if (!rgb) return false;
  return (
    contrast(rgb, [245, 246, 248]) < 4.5 || contrast(rgb, [25, 27, 31]) < 4.5
  );
}

function hexRgb(value: string): [number, number, number] | undefined {
  const match = /^#([0-9a-f]{6})$/iu.exec(value.trim());
  if (!match) return undefined;
  const hex = match[1];
  if (!hex) return undefined;
  return [
    Number.parseInt(hex.slice(0, 2), 16),
    Number.parseInt(hex.slice(2, 4), 16),
    Number.parseInt(hex.slice(4, 6), 16),
  ];
}

function firstNonEmpty(...values: (string | undefined)[]): string | undefined {
  return values.find((value) => value !== undefined && value !== "");
}

function contrast(
  left: [number, number, number],
  right: [number, number, number],
): number {
  const luminance = (rgb: [number, number, number]) => {
    const coefficients = [0.2126, 0.7152, 0.0722];
    return rgb.reduce((sum, channel, index) => {
      const value = channel / 255;
      const linear =
        value <= 0.03928 ? value / 12.92 : ((value + 0.055) / 1.055) ** 2.4;
      return sum + linear * (coefficients[index] ?? 0);
    }, 0);
  };
  const a = luminance(left);
  const b = luminance(right);
  return (Math.max(a, b) + 0.05) / (Math.min(a, b) + 0.05);
}

function taskLabelName(key: string, t: TFunction): string {
  const statusNames: Record<string, string> = {
    "status.not_started": "displayState.notStarted",
    "status.designing": "displayState.designing",
    "status.designed": "displayState.designed",
    "status.in_progress": "displayState.inProgress",
    "status.implemented": "displayState.implemented",
    "status.in_review": "displayState.inReview",
    "status.approved": "displayState.approved",
    "status.completed": "displayState.completed",
    "status.merged": "displayState.merged",
    "status.closed": "displayState.closed",
    "status.unknown": "displayState.unknown",
  };
  const blockNames: Record<string, string> = {
    "block.dependency_unresolved": "blockLabel.dependencyUnresolved",
    "block.conflict": "blockLabel.conflict",
    "block.changes_requested": "blockLabel.changesRequested",
    "block.ci_failed": "blockLabel.ciFailed",
    "block.unknown": "blockLabel.unknown",
  };
  return String(t((statusNames[key] ?? blockNames[key] ?? key) as never));
}

function sourceName(source: string, t: TFunction): string {
  const key = source === "built-in" ? "builtIn" : source;
  return String(t(`taskLabels.sourceNames.${key}` as never));
}

export function TaskLabelSettingsPanel() {
  const { t } = useTranslation();
  const config = useTaskLabelConfig(true);
  const update = useTaskLabelConfigMutation(configMutations.updateTaskLabels);
  const [state, setState] = useState({
    update: emptyUpdate,
    dirty: false,
    invalid: false,
  });
  useRegisterSettingsSection("task-labels", {
    dirty: state.dirty,
    invalid: state.invalid,
    save: async () => {
      await update.mutateAsync(state.update);
    },
  });
  if (config.isPending)
    return <p className="settings-panel-state">{t("taskLabels.loading")}</p>;
  if (!config.data) return <p className="form-error">{config.error.message}</p>;
  return (
    <>
      <TaskLabelEditor
        scope="global"
        keys={config.data.keys}
        maxTextCodepoints={config.data.maxTextCodepoints}
        builtIn={config.data.builtIn}
        overrides={config.data.overrides}
        editable
        onStateChange={(next, dirty, invalid) => {
          setState({ update: next, dirty, invalid });
        }}
      />
      {update.error && <p className="form-error">{update.error.message}</p>}
    </>
  );
}

export function TaskLabelOverridesTabPanel({
  active,
  idPrefix,
  labelsMounted,
  scope,
  overrides,
  parentOverrides,
  editable,
  onStateChange,
}: {
  active: boolean;
  idPrefix: string;
  labelsMounted: boolean;
  scope: "project" | "feature";
  overrides?: TaskLabelOverrides | undefined;
  parentOverrides?: TaskLabelOverrides | undefined;
  editable: boolean;
  onStateChange: (
    update: TaskLabelOverridesUpdate,
    dirty: boolean,
    invalid: boolean,
  ) => void;
}) {
  return (
    <div
      className="entity-edit-tab-panel task-labels-tab-panel"
      hidden={!active}
      id={`${idPrefix}-panel-labels`}
      role="tabpanel"
      aria-labelledby={`${idPrefix}-tab-labels`}
      tabIndex={0}
    >
      {labelsMounted && (
        <TaskLabelOverridesPanelContent
          scope={scope}
          parentOverrides={parentOverrides}
          overrides={overrides}
          editable={editable}
          onStateChange={onStateChange}
        />
      )}
    </div>
  );
}

function TaskLabelOverridesPanelContent({
  scope,
  parentOverrides,
  overrides,
  editable,
  onStateChange,
}: {
  scope: "project" | "feature";
  parentOverrides?: TaskLabelOverrides | undefined;
  overrides?: TaskLabelOverrides | undefined;
  editable: boolean;
  onStateChange: (
    update: TaskLabelOverridesUpdate,
    dirty: boolean,
    invalid: boolean,
  ) => void;
}) {
  const config = useTaskLabelConfig(true);
  return (
    <TaskLabelEditor
      scope={scope}
      keys={config.data?.keys ?? []}
      maxTextCodepoints={config.data?.maxTextCodepoints ?? 32}
      builtIn={config.data?.builtIn}
      globalOverrides={config.data?.overrides}
      parentOverrides={parentOverrides}
      overrides={overrides}
      editable={editable}
      onStateChange={onStateChange}
    />
  );
}
