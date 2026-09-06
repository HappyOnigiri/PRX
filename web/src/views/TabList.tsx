import { useEffect, useRef, type KeyboardEvent, type ReactNode } from "react";

interface TabDescriptor<Id extends string> {
  id: Id;
  label: string;
  // Some strips prefix the label with an icon; the rest render text alone.
  icon?: ReactNode;
}

interface TabListProps<Id extends string> {
  tabs: readonly TabDescriptor<Id>[];
  active: Id;
  onSelect: (id: Id) => void;
  // The DOM ids are `<idPrefix>-tab-<id>` and `<idPrefix>-panel-<id>`, which
  // TabPanel rebuilds from the same two values.
  idPrefix: string;
  className: string;
  tabClassName: string;
  label?: string;
  // Dialogs that open onto their tabs move focus there; a strip on a page does
  // not, because taking focus on load would move the reader off the heading.
  focusOnMount?: boolean;
}

// The tab strips of the settings dialog, the reference dialog, and the feature
// lists differ only in wording and styling, so the roles, the roving tabindex,
// and the arrow keys live here once.
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

// The panel only carries the wiring back to its tab. Whether its children are
// mounted at all stays with the caller, which is what lets one dialog keep an
// unsaved form alive and another mount its panel only while it is open.
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
