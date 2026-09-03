import {
  createRootRoute,
  createRoute,
  createRouter,
  Outlet,
} from "@tanstack/react-router";
import { AppShell } from "./shell";
import { ActiveFeatures } from "./views/ActiveFeatures";
import { ArchivedFeatures } from "./views/ArchivedFeatures";
import { CompletedFeatures } from "./views/CompletedFeatures";
import { Dashboard } from "./views/Dashboard";
import { FeatureWorkspace } from "./views/FeatureWorkspace";
import { ProjectListPage } from "./views/ProjectListPage";
import { ProjectWorkspace } from "./views/ProjectWorkspace";
import { TaskSearch } from "./views/TaskSearch";

const rootRoute = createRootRoute({
  component: () => (
    <AppShell>
      <Outlet />
    </AppShell>
  ),
});
const indexRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/",
  component: Dashboard,
});
const featureRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/features/$featureId",
  component: FeatureWorkspace,
});
const activeRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/active",
  component: ActiveFeatures,
});
const archivedRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/archived",
  component: ArchivedFeatures,
});
const completedRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/completed",
  component: CompletedFeatures,
});
const projectsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/projects",
  // The archive toggle stays in the URL so reload, history, and a shared link
  // reproduce the view instead of depending on browser-local state.
  validateSearch: (search: Record<string, unknown>) => ({
    archived: search["archived"] === true || search["archived"] === "true",
  }),
  component: ProjectListPage,
});
const projectRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/projects/$projectId",
  component: ProjectWorkspace,
});
const taskSearchRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/tasks",
  validateSearch: (search: Record<string, unknown>) => ({
    q: typeof search["q"] === "string" ? search["q"] : "",
  }),
  component: TaskSearch,
});
const routeTree = rootRoute.addChildren([
  indexRoute,
  activeRoute,
  archivedRoute,
  completedRoute,
  taskSearchRoute,
  featureRoute,
  projectsRoute,
  projectRoute,
]);

export const router = createRouter({ routeTree });

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}
