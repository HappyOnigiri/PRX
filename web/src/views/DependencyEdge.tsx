import { BaseEdge, EdgeToolbar, type EdgeProps } from "@xyflow/react";
import { Unlink } from "lucide-react";
import type { DependencyFlowEdge, GraphPoint } from "./dependencyGraph";
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
      {data && (
        <EdgeToolbar
          className="dependency-edge-toolbar nodrag nopan"
          edgeId={id}
          isVisible={selected === true}
          onPointerDown={(event) => {
            event.stopPropagation();
          }}
          x={toolbarPosition.x}
          y={toolbarPosition.y}
        >
          <span title={data.label}>{data.label}</span>
          {!data.readOnly && (
            <IconButton
              className="dependency-edge-remove"
              disabled={data.disabled}
              icon={Unlink}
              iconOnly
              label={data.removeLabel}
              onClick={(event) => {
                event.stopPropagation();
                data.onRemove();
              }}
              size="compact"
              variant="danger"
            />
          )}
        </EdgeToolbar>
      )}
    </>
  );
}
