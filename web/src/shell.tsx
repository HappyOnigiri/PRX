import { FormEvent, type ReactNode, useState } from "react";
import { Link, useNavigate } from "@tanstack/react-router";
import { mutations } from "./api";
import { useDomainMutation, useSnapshot } from "./hooks";

export function AppShell({ children }: { children: ReactNode }) {
  const snapshot = useSnapshot();
  const navigate = useNavigate();
  const [showCreate, setShowCreate] = useState(false);
  const createFeature = useDomainMutation(mutations.createFeature);
  async function submit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    const data = new FormData(event.currentTarget);
    const response = await createFeature.mutateAsync({
      slug: String(data.get("slug")),
      title: String(data.get("title")),
      description: String(data.get("description")),
    });
    setShowCreate(false);
    if (response.feature)
      await navigate({
        to: "/features/$featureId",
        params: { featureId: response.feature.id },
      });
  }
  return (
    <div className="app-shell">
      <aside className="rail">
        <Link to="/" className="brand" aria-label="PRX dashboard">
          <span className="brand-mark">
            P<span>R</span>X
          </span>
          <small>Dependency control</small>
        </Link>
        <nav aria-label="Features">
          <Link to="/" className="nav-link">
            Overview <span>{snapshot.data?.features.length ?? "—"}</span>
          </Link>
          <div className="nav-caption">Active circuits</div>
          {snapshot.data?.features
            .filter((f) => !f.archived)
            .map((feature) => (
              <Link
                key={feature.id}
                to="/features/$featureId"
                params={{ featureId: feature.id }}
                className="feature-link"
                activeProps={{ "data-active": true }}
              >
                <i
                  className={
                    feature.conflictCount
                      ? "pulse conflict"
                      : feature.readyCount
                        ? "pulse ready"
                        : "pulse"
                  }
                />
                <span>{feature.title}</span>
                <b>
                  {feature.mergedCount}/{feature.taskCount}
                </b>
              </Link>
            ))}
        </nav>
        <button className="rail-action" onClick={() => setShowCreate(true)}>
          ＋ New feature
        </button>
        <div className="rail-foot">
          <span className={snapshot.isError ? "health bad" : "health"} />
          {snapshot.isError ? "Server unavailable" : "Local database online"}
        </div>
      </aside>
      <main className="main-stage">{children}</main>
      {showCreate && (
        <div className="scrim" role="presentation">
          <form
            className="dialog"
            onSubmit={submit}
            aria-label="Create feature"
          >
            <header>
              <p>New circuit</p>
              <h2>Create feature</h2>
            </header>
            <label>
              Slug
              <input
                name="slug"
                required
                pattern="[a-z0-9]+(?:-[a-z0-9]+)*"
                placeholder="payments-rollout"
              />
            </label>
            <label>
              Title
              <input name="title" required placeholder="Payments rollout" />
            </label>
            <label>
              Description
              <textarea
                name="description"
                placeholder="What must this feature deliver?"
              />
            </label>
            {createFeature.error && (
              <p className="form-error">{createFeature.error.message}</p>
            )}
            <footer>
              <button
                type="button"
                className="secondary"
                onClick={() => setShowCreate(false)}
              >
                Cancel
              </button>
              <button disabled={createFeature.isPending}>Create feature</button>
            </footer>
          </form>
        </div>
      )}
    </div>
  );
}
