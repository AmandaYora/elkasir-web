import { createBrowserRouter, Navigate } from "react-router-dom";
import { publicRoutes } from "./public.routes";
import { protectedRoutes } from "./protected.routes";
import { platformRoutes } from "./platform.routes";
import { ROUTE_PATHS } from "./route-paths";

export const router = createBrowserRouter([
  ...publicRoutes,
  ...protectedRoutes,
  ...platformRoutes,
  { path: "*", element: <Navigate to={ROUTE_PATHS.dashboard} replace /> },
]);
