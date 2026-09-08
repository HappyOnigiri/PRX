import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from "react";

// 設定ダイアログの保存は 1 つなので、各タブは自分の下書きの状態と保存手続きを
// ここへ預ける。フッタのボタンは預かったものをまとめて実行する。
export interface SettingsSection {
  dirty: boolean;
  invalid: boolean;
  save: () => Promise<void>;
}

interface SectionStore {
  publish: (id: string, section: SettingsSection) => void;
  withdraw: (id: string) => void;
}

const SectionStoreContext = createContext<SectionStore | undefined>(undefined);

export interface SettingsSectionsController {
  dirty: boolean;
  invalid: boolean;
  saveAll: () => Promise<void>;
  provider: (children: ReactNode) => ReactNode;
}

export function useSettingsSections(): SettingsSectionsController {
  const sections = useRef(new Map<string, SettingsSection>());
  const [summary, setSummary] = useState({ dirty: false, invalid: false });

  const refresh = useCallback(() => {
    const published = [...sections.current.values()];
    setSummary({
      dirty: published.some((section) => section.dirty),
      // 編集していないタブが持つ値は、サーバーに既にあるものなので保存を
      // 妨げない。止めるのはこれから書き込む下書きが不正なときだけである。
      invalid: published.some((section) => section.dirty && section.invalid),
    });
  }, []);

  const store = useMemo<SectionStore>(
    () => ({
      publish(id, section) {
        sections.current.set(id, section);
        refresh();
      },
      withdraw(id) {
        sections.current.delete(id);
        refresh();
      },
    }),
    [refresh],
  );

  // 保存は順番に走らせる。設定ファイルは 1 つで、並行に書くと後から書いた側が
  // 相手の変更を落とすため。
  const saveAll = useCallback(async () => {
    for (const section of sections.current.values()) {
      if (section.dirty) await section.save();
    }
  }, []);

  const provider = useCallback(
    (children: ReactNode) => (
      <SectionStoreContext.Provider value={store}>
        {children}
      </SectionStoreContext.Provider>
    ),
    [store],
  );

  return { ...summary, saveAll, provider };
}

export function useRegisterSettingsSection(
  id: string,
  section: SettingsSection,
) {
  const store = useContext(SectionStoreContext);
  const latest = useRef(section);
  useEffect(() => {
    latest.current = section;
  }, [section]);
  useEffect(() => {
    store?.publish(id, {
      dirty: section.dirty,
      invalid: section.invalid,
      // 保存の中身は毎レンダーで変わるので、実行時に最新のものを読む。
      save: () => latest.current.save(),
    });
  }, [store, id, section.dirty, section.invalid]);
  useEffect(() => () => store?.withdraw(id), [store, id]);
}
