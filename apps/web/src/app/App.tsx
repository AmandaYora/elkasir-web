import { AppProvider } from "./providers/AppProvider";
import { RouterProvider } from "./providers/RouterProvider";

export function App() {
  return (
    <AppProvider>
      <RouterProvider />
    </AppProvider>
  );
}
