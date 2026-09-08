import { type ReactNode } from "react";
import { useSettingsSections } from "../src/views/settingsSections";

// 設定のパネルは自前の保存ボタンを持たず、下書きの状態をレジストリへ預ける。
// パネル単体のテストは、ダイアログのフッタと同じ役目をこのボタンで代える。
export function SettingsSectionsHarness({ children }: { children: ReactNode }) {
  const sections = useSettingsSections();
  return (
    <>
      {sections.provider(children)}
      <button
        type="button"
        disabled={!sections.dirty || sections.invalid}
        onClick={() => {
          void sections.saveAll().catch(() => undefined);
        }}
      >
        Save all
      </button>
    </>
  );
}
