import "@mantine/core/styles.css";
import { createRoot } from "react-dom/client";
import { createTheme, MantineProvider } from "@mantine/core";
import { App } from "./App";

const theme = createTheme({ fontFamily: "Open Sans, sans-serif", primaryColor: "cyan" });

createRoot(document.getElementById("root")!).render(
  <MantineProvider theme={theme} defaultColorScheme="dark">
    <App />
  </MantineProvider>
);
