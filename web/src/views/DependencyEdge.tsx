import { BaseEdge, EdgeToolbar, type EdgeProps } from "@xyflow/react";
import { X } from "lucide-react";
import { useContext, useState } from "react";
import {
  HoveredEdgeContext,
  type DependencyFlowEdge,
  type GraphPoint,
} from "./dependencyGraph";
import { IconButton } from "./IconButton";

function pathForPoints(points: GraphPoint[]) {
  return points
    .map(
      (point, index) =>
        `${index === 0 ? "M" : "L"}${String(point.x)} ${String(point.y)}`,
    )
    .join(" ");
}

function pointHalfwayAlong(points: GraphPoint[]) {
  const first = points[0];
  if (!first) return { x: 0, y: 0 };
  if (points.length === 1) return first;

  let total = 0;
  let previous = first;
  for (const point of points.slice(1)) {
    total += Math.hypot(point.x - previous.x, point.y - previous.y);
    previous = point;
  }
  const halfway = total / 2;
  let travelled = 0;

  previous = first;
  for (const point of points.slice(1)) {
    const segment = Math.hypot(point.x - previous.x, point.y - previous.y);
    if (travelled + segment >= halfway) {
      const ratio = segment === 0 ? 0 : (halfway - travelled) / segment;
      return {
        x: previous.x + (point.x - previous.x) * ratio,
        y: previous.y + (point.y - previous.y) * ratio,
      };
    }
    travelled += segment;
    previous = point;
  }

  return previous;
}

export function DependencyEdge({
  data,
  id,
  interactionWidth,
  markerEnd,
  markerStart,
  selected,
  sourceX,
  sourceY,
  style,
  targetX,
  targetY,
}: EdgeProps<DependencyFlowEdge>) {
  const hoveredEdgeId = useContext(HoveredEdgeContext);
  // ツールバーはエッジの外へ描かれるので、線からボタンへポインタを移すと線の
  // hover は外れる。ツールバー自身の hover を足して途中で消えないようにする。
  const [toolbarHovered, setToolbarHovered] = useState(false);
  const points = data?.route?.points.length
    ? data.route.points
    : [
        { x: sourceX, y: sourceY },
        { x: targetX, y: targetY },
      ];
  const path = pathForPoints(points);
  const toolbarPosition = pointHalfwayAlong(points);

  return (
    <>
      <BaseEdge
        id={id}
        interactionWidth={interactionWidth ?? 20}
        {...(markerEnd ? { markerEnd } : {})}
        {...(markerStart ? { markerStart } : {})}
        path={path}
        style={style ?? {}}
      />
      {data && !data.readOnly && (
        <EdgeToolbar
          // React Flow はツールバーを画面上で一定の大きさに保つので、線の中点を
          // 渡せばどの倍率でもそこに乗る。CSS のオフセットだとビューポートに
          // 合わせて拡大縮小してしまう。
          className="dependency-edge-toolbar nodrag nopan"
          edgeId={id}
          isVisible={
            selected === true || hoveredEdgeId === id || toolbarHovered
          }
          onMouseEnter={() => {
            setToolbarHovered(true);
          }}
          onMouseLeave={() => {
            setToolbarHovered(false);
          }}
          onPointerDown={(event) => {
            event.stopPropagation();
          }}
          // React Flow は選択中のエッジを z-index 1000 へ持ち上げる。線の中点
          // に重ねるボタンは、その当たり判定より上に置かないと押せなくなる。
          style={{ zIndex: 1001 }}
          x={toolbarPosition.x}
          y={toolbarPosition.y}
        >
          <IconButton
            className="dependency-edge-remove"
            disabled={data.disabled}
            icon={X}
            iconOnly
            label={data.removeLabel}
            onClick={(event) => {
              event.stopPropagation();
              data.onRemove();
            }}
            size="compact"
            variant="danger"
          />
        </EdgeToolbar>
      )}
    </>
  );
}
