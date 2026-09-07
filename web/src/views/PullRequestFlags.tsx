import { CircleAlert, TriangleAlert } from "lucide-react";
import { useRef, useState } from "react";
import { createPortal } from "react-dom";
import { useTranslation } from "react-i18next";

interface PullRequestFlagsProps {
  stale: boolean;
  // syncError は本文をそのまま理由に載せるので、真偽ではなく文言で受ける。
  syncError: string;
  size?: number;
}

// pull request 自身の異常はカードでもノードでも 1 行に収めたいので、文ではなく
// アイコンで示し、理由は指した瞬間に出る tooltip と読み上げ用の名前が持つ。
// docs/design/webui.md を参照。
export function PullRequestFlags({
  stale,
  syncError,
  size = 13,
}: PullRequestFlagsProps) {
  const { t } = useTranslation();
  return (
    <>
      {stale && (
        <PullRequestFlag
          icon={TriangleAlert}
          tone="stale"
          size={size}
          text={t("pullRequestFlag.stale")}
        />
      )}
      {syncError && (
        <PullRequestFlag
          icon={CircleAlert}
          tone="sync-error"
          size={size}
          text={`${t("pullRequestFlag.syncError")}: ${syncError}`}
        />
      )}
    </>
  );
}

// 印はグラフノードのスクロールする資料一覧の中にも立つので、tooltip は body へ
// 出して切り取られないようにし、位置は指した時点の矩形から決める。
function PullRequestFlag({
  icon: Icon,
  tone,
  size,
  text,
}: {
  icon: typeof TriangleAlert;
  tone: "stale" | "sync-error";
  size: number;
  text: string;
}) {
  const anchor = useRef<HTMLButtonElement>(null);
  const [tip, setTip] = useState<{ left: number; top: number } | undefined>();
  const show = () => {
    const rect = anchor.current?.getBoundingClientRect();
    if (rect) setTip({ left: rect.left + rect.width / 2, top: rect.top - 6 });
  };
  const hide = () => {
    setTip(undefined);
  };
  return (
    // 理由はポインタだけのものにしない。キーボードでも辿り着けるよう、印自身を
    // focus を受けるコントロールにする。
    <button
      ref={anchor}
      type="button"
      className={`pr-flag is-${tone}`}
      aria-label={text}
      onMouseEnter={show}
      onMouseLeave={hide}
      onFocus={show}
      onBlur={hide}
    >
      <Icon aria-hidden="true" focusable="false" size={size} />
      {tip &&
        createPortal(
          <span className="pr-flag-tip" style={tip}>
            {text}
          </span>,
          document.body,
        )}
    </button>
  );
}
