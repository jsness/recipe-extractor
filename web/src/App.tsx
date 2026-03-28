import { useEffect, useState } from "react";
import { LandingPage } from "./components/LandingPage";
import { RecipeApp } from "./RecipeApp";

const APP_PATH_PREFIX = "/app";

export const App = () => {
  const [pathname, setPathname] = useState(window.location.pathname);

  useEffect(() => {
    const handlePopState = () => setPathname(window.location.pathname);
    window.addEventListener("popstate", handlePopState);
    return () => window.removeEventListener("popstate", handlePopState);
  }, []);

  if (pathname === APP_PATH_PREFIX || pathname.startsWith(`${APP_PATH_PREFIX}/`)) {
    return <RecipeApp />;
  }

  return <LandingPage onLaunchApp={() => window.location.assign(APP_PATH_PREFIX)} />;
};
