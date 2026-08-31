import {
  createRootRoute,
  createRoute,
  createRouter,
  Outlet,
} from "@tanstack/react-router";
import { AppShell } from "./shell";
import { ArchivedFeatures } from "./views/ArchivedFeatures";
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
const archivedRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/archived",
  component: ArchivedFeatures,
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
  archivedRoute,
  taskSearchRoute,
  featureRoute,
]);

export const router = createRouter({ routeTree });

declare module "@tanstack/react-router" {
  interface Register {
    router: typeof router;
  }
}
