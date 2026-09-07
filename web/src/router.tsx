import {
  createRootRoute,
  createRoute,
  createRouter,
  Outlet,
} from "@tanstack/react-router";
import { validateFeatureTabSearch } from "./feature-tabs";
import { AppShell } from "./shell";
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
const projectsRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/projects",
  // アーカイブ切り替えは URL に持たせる。ブラウザ側の状態に頼らず、リロード・
  // 履歴・共有リンクで同じ表示を再現するため。
  validateSearch: (search: Record<string, unknown>) => ({
    archived: search["archived"] === true || search["archived"] === "true",
  }),
  component: ProjectListPage,
});
const projectRoute = createRoute({
  getParentRoute: () => rootRoute,
  path: "/projects/$projectId",
  // ステータスタブも上のアーカイブ切り替えと同じ理由で URL に持たせる。
  validateSearch: validateFeatureTabSearch,
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
