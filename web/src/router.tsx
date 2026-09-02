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
]);

export const router = createRouter({ routeTree });

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}
