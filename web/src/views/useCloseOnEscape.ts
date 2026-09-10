import { useEffect, useRef } from "react";

interface EscapeTarget {
  close: () => void;
  closable: boolean;
}

// モーダルは重なるので、Escape は最後に開いた 1 つだけが受ける。閉じられない間も
// 受けたまま捨て、下のモーダルへ渡さない。docs/design/webui.md を参照。
const stack: EscapeTarget[] = [];

function closeTopmost(event: KeyboardEvent) {
  if (event.key !== "Escape") return;
  const target = stack.at(-1);
  if (!target) return;
  event.preventDefault();
  if (target.closable) target.close();
}

// 焦点がモーダルの中にあるとは限らないので、要素ではなく window で受ける。
export function useCloseOnEscape(onClose: () => void, closable = true) {
  const target = useRef<EscapeTarget>({ close: onClose, closable });
  useEffect(() => {
    target.current.close = onClose;
    target.current.closable = closable;
  });
  useEffect(() => {
    const entry = target.current;
    stack.push(entry);
    if (stack.length === 1) window.addEventListener("keydown", closeTopmost);
    return () => {
      const index = stack.lastIndexOf(entry);
      if (index !== -1) stack.splice(index, 1);
      if (stack.length === 0)
        window.removeEventListener("keydown", closeTopmost);
    };
  }, []);
}
