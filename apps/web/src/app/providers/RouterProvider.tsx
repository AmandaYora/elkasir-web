import { RouterProvider as ReactRouterProvider } from "react-router-dom";
import { router } from "@/app/routes";

export function RouterProvider() {
  return <ReactRouterProvider router={router} />;
}
