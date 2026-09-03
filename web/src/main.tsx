import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { RouterProvider } from "@tanstack/react-router";
import "@xyflow/react/dist/style.css";
import React from "react";
import ReactDOM from "react-dom/client";
import "./i18n";
import { router } from "./router";
import "./styles/base.css";
import "./styles/components.css";
import "./styles/dashboard.css";
import "./styles/graph.css";
import "./styles/markdown-preview.css";
import "./styles/project.css";
import "./styles/shell.css";
import "./styles/task-inspector.css";
import "./styles/task-search.css";
import "./styles/theme.css";
import "./styles/workspace.css";
import "./theme";

const queryClient = new QueryClient({
  defaultOptions: { queries: { staleTime: 5_000, retry: 1 } },
});

const root = document.getElementById("root");
if (!root) throw new Error("The application root element is missing.");

ReactDOM.createRoot(root).render(
  <React.StrictMode>
    <QueryClientProvider client={queryClient}>
      <RouterProvider router={router} />
    </QueryClientProvider>
  </React.StrictMode>,
);
