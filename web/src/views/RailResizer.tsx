import { useRef, type KeyboardEvent, type PointerEvent } from "react";
import { useTranslation } from "react-i18next";
import {
  clampRailWidth,
  maxRailWidth,
  minRailWidth,
  writeRailWidth,
} from "../i18n/settings";

const keyboardStep = 16;

interface Drag {
  pointerId: number;
  startX: number;
  startWidth: number;
}

// rail は position: fixed で左端に張り付いているので、幅はポインタの横移動の
// 差分だけで決まる。レイアウトを測る必要がなく、jsdom でもそのまま動く。
export function RailResizer({
  railId,
  width,
  onWidth,
  onResizing,
}: {
  railId: string;
  width: number;
  onWidth: (width: number) => void;
  onResizing: (resizing: boolean) => void;
}) {
  const { t } = useTranslation();
  const drag = useRef<Drag | undefined>(undefined);

  function commit(next: number) {
    onWidth(next);
    writeRailWidth(next);
  }

  function handlePointerDown(event: PointerEvent<HTMLDivElement>) {
    if (event.button !== 0) return;
    // テキスト選択とネイティブのドラッグ開始を止めないと、追従が途切れる。
    event.preventDefault();
    drag.current = {
      pointerId: event.pointerId,
      startX: event.clientX,
      startWidth: width,
    };
    event.currentTarget.setPointerCapture(event.pointerId);
    onResizing(true);
  }

  function handlePointerMove(event: PointerEvent<HTMLDivElement>) {
    const current = drag.current;
    if (current?.pointerId !== event.pointerId) return;
    // 書き込みは操作の終わりだけにする。move ごとに保存する必要はない。
    onWidth(
      clampRailWidth(current.startWidth + event.clientX - current.startX),
    );
  }

  // pointerup と pointercancel の両方をここへ通す。cancel を落とすと幅が保存
  // されず、ドラッグ中の印も残ったままになる。
  function handlePointerEnd(event: PointerEvent<HTMLDivElement>) {
    const current = drag.current;
    if (current?.pointerId !== event.pointerId) return;
    drag.current = undefined;
    event.currentTarget.releasePointerCapture(event.pointerId);
    onResizing(false);
    commit(clampRailWidth(current.startWidth + event.clientX - current.startX));
  }

  function handleKey(event: KeyboardEvent<HTMLDivElement>) {
    let next: number | undefined;
    if (event.key === "ArrowLeft") next = width - keyboardStep;
    if (event.key === "ArrowRight") next = width + keyboardStep;
    if (event.key === "Home") next = minRailWidth;
    if (event.key === "End") next = maxRailWidth;
    if (next === undefined) return;
    event.preventDefault();
    commit(clampRailWidth(next));
  }

  return (
    <div
      aria-controls={railId}
      aria-label={t("nav.resizeRail")}
      aria-orientation="vertical"
      aria-valuemax={maxRailWidth}
      aria-valuemin={minRailWidth}
      aria-valuenow={width}
      className="rail-resizer"
      onKeyDown={handleKey}
      onPointerCancel={handlePointerEnd}
      onPointerDown={handlePointerDown}
      onPointerMove={handlePointerMove}
      onPointerUp={handlePointerEnd}
      role="separator"
      tabIndex={0}
      title={t("nav.resizeRail")}
    />
  );
}
