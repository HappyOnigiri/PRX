import { useEffect, useRef, type KeyboardEvent, type ReactNode } from "react";

interface TabDescriptor<Id extends string> {
  id: Id;
  label: string;
  // ラベルの前にアイコンを置くストリップもあれば、テキストだけのものもある。
  icon?: ReactNode;
}

interface TabListProps<Id extends string> {
  tabs: readonly TabDescriptor<Id>[];
  active: Id;
  onSelect: (id: Id) => void;
  // DOM id は `<idPrefix>-tab-<id>` と `<idPrefix>-panel-<id>` で、TabPanel が
  // 同じ 2 つの値から組み立て直す。
  idPrefix: string;
  className: string;
  tabClassName: string;
  label?: string;
  // タブを開いた状態で表示するダイアログはそこへフォーカスを移すが、ページ上の
  // ストリップは移さない。読み手が見出しから外れてしまうため。
  focusOnMount?: boolean;
}

// 設定ダイアログ・参照ダイアログ・feature リストのタブストリップは文言と見た目
// しか違わないので、role と roving tabindex、矢印キーはここに 1 度だけ置く。
export function TabList<Id extends string>({
  tabs,
  active,
  onSelect,
  idPrefix,
  className,
  tabClassName,
  label,
  focusOnMount,
}: TabListProps<Id>) {
  const tabRefs = useRef<(HTMLButtonElement | null)[]>([]);

  useEffect(() => {
    if (focusOnMount) tabRefs.current[0]?.focus();
  }, [focusOnMount]);

  function moveTo(index: number) {
    const tab = tabs[index];
    if (!tab) return;
    onSelect(tab.id);
    tabRefs.current[index]?.focus();
  }

  function handleKey(event: KeyboardEvent<HTMLButtonElement>) {
    const current = tabs.findIndex((tab) => tab.id === active);
    let next: number | undefined;
    if (event.key === "ArrowRight") next = (current + 1) % tabs.length;
    if (event.key === "ArrowLeft")
      next = (current - 1 + tabs.length) % tabs.length;
    if (event.key === "Home") next = 0;
    if (event.key === "End") next = tabs.length - 1;
    if (next === undefined) return;
    event.preventDefault();
    moveTo(next);
  }

  return (
    <div className={className} role="tablist" aria-label={label}>
      {tabs.map((tab, index) => (
        <button
          aria-controls={`${idPrefix}-panel-${tab.id}`}
          aria-selected={tab.id === active}
          className={tabClassName}
          id={`${idPrefix}-tab-${tab.id}`}
          key={tab.id}
          onClick={() => {
            onSelect(tab.id);
          }}
          onKeyDown={handleKey}
          ref={(element) => {
            tabRefs.current[index] = element;
          }}
          role="tab"
          tabIndex={tab.id === active ? 0 : -1}
          type="button"
        >
          {tab.icon}
          {tab.label}
        </button>
      ))}
    </div>
  );
}

// パネルはタブへの結線だけを担う。子をマウントするかどうかは呼び出し元に残る。
// これにより、あるダイアログは未保存のフォームを生かし、別のダイアログは開いて
// いる間だけパネルをマウントできる。
export function TabPanel({
  children,
  active,
  className,
  idPrefix,
  tab,
}: {
  children: ReactNode;
  active: boolean;
  className: string;
  idPrefix: string;
  tab: string;
}) {
  return (
    <div
      aria-labelledby={`${idPrefix}-tab-${tab}`}
      className={className}
      hidden={!active}
      id={`${idPrefix}-panel-${tab}`}
      role="tabpanel"
      tabIndex={0}
    >
      {children}
    </div>
  );
}
