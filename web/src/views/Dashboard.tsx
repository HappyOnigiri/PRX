import { Link } from "@tanstack/react-router";
import { useSnapshot } from "../hooks";

const queueNames = [
  ["readyTasks", "Ready now", "Dependencies cleared"],
  ["reviewWaitingTasks", "Review line", "Waiting on people"],
  ["conflictTasks", "Conflicts", "Needs intervention"],
  ["staleTasks", "Stale signal", "Refresh GitHub data"],
] as const;

export function Dashboard() {
  const { data, isPending, error, refetch } = useSnapshot();
  if (isPending)
    return (
      <StateMessage
        title="Mapping dependencies…"
        detail="Reading the local graph and latest GitHub state."
      />
    );
  if (error || !data)
    return (
      <StateMessage
        title="The roadmap could not be loaded"
        detail={error?.message ?? "No data returned."}
        action={() => refetch()}
      />
    );
  return (
    <div className="dashboard">
      <header className="page-head">
        <div>
          <p className="eyebrow">Operations / all repositories</p>
          <h1>
            What can move <em>now?</em>
          </h1>
          <p>
            Every queue is derived from the dependency graph—never manually
            marked ready.
          </p>
        </div>
        <div className="clock">
          <span>{data.tasks.length}</span>
          <small>nodes under control</small>
        </div>
      </header>
      <section className="queue-strip" aria-label="Roadmap status">
        {queueNames.map(([key, title, detail]) => (
          <article key={key} className={`queue-meter ${key}`}>
            <span>{data[key].length}</span>
            <div>
              <h2>{title}</h2>
              <p>{detail}</p>
            </div>
          </article>
        ))}
      </section>
      <div className="dashboard-grid">
        <section className="ready-board">
          <header>
            <p className="section-label">Execution queue</p>
            <h2>Ready to start</h2>
          </header>
          {data.readyTasks.length === 0 ? (
            <div className="empty">
              <span>◇</span>
              <h3>No task is ready yet</h3>
              <p>
                Create a feature and connect its tasks, or clear an upstream
                blocker.
              </p>
            </div>
          ) : (
            <ol>
              {data.readyTasks.map((task, index) => {
                const feature = data.features.find(
                  (f) => f.id === task.featureId,
                );
                return (
                  <li key={task.id}>
                    <span className="queue-index">
                      {String(index + 1).padStart(2, "0")}
                    </span>
                    <div>
                      <Link
                        to="/features/$featureId"
                        params={{ featureId: task.featureId }}
                      >
                        {task.title}
                      </Link>
                      <p>
                        {feature?.title} · {task.assignee || "Unassigned"}
                      </p>
                    </div>
                    <i>READY</i>
                  </li>
                );
              })}
            </ol>
          )}
        </section>
        <section className="feature-board">
          <header>
            <p className="section-label">Feature telemetry</p>
            <h2>Delivery lines</h2>
          </header>
          {data.features.length === 0 ? (
            <div className="empty compact">
              <h3>No features yet</h3>
              <p>Use “New feature” to draw the first delivery circuit.</p>
            </div>
          ) : (
            <div className="feature-table">
              {data.features.map((feature) => (
                <Link
                  key={feature.id}
                  to="/features/$featureId"
                  params={{ featureId: feature.id }}
                  className="feature-row"
                >
                  <div>
                    <b>{feature.title}</b>
                    <small>{feature.slug}</small>
                  </div>
                  <div className="progress-track">
                    <i
                      style={{
                        width: `${feature.taskCount ? (feature.mergedCount / feature.taskCount) * 100 : 0}%`,
                      }}
                    />
                  </div>
                  <span>
                    {feature.mergedCount}/{feature.taskCount}
                  </span>
                  <strong>
                    {feature.readyCount
                      ? `${feature.readyCount} ready`
                      : feature.conflictCount
                        ? `${feature.conflictCount} blocked`
                        : "steady"}
                  </strong>
                </Link>
              ))}
            </div>
          )}
        </section>
      </div>
    </div>
  );
}

function StateMessage({
  title,
  detail,
  action,
}: {
  title: string;
  detail: string;
  action?: () => void;
}) {
  return (
    <div className="state-message">
      <div className="spinner" />
      <h1>{title}</h1>
      <p>{detail}</p>
      {action && <button onClick={action}>Try again</button>}
    </div>
  );
}
