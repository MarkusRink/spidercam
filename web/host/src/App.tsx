import { Header } from "./components/Header.js";
import { OutputPreview } from "./components/OutputPreview.js";
import { SettingsPanel } from "./components/SettingsPanel.js";
import { StreamGrid } from "./components/StreamGrid.js";
import { SessionStoreProvider } from "./stores/session-store.js";

export default function App() {
  return (
    <SessionStoreProvider>
      <div class="grid h-screen grid-rows-[40px_minmax(0,1fr)_260px] grid-cols-[1fr_360px] gap-2 overflow-hidden p-2">
        <Header class="col-span-2" />
        <OutputPreview />
        <SettingsPanel />
        <StreamGrid class="col-span-2 min-h-0 overflow-y-auto" />
      </div>
    </SessionStoreProvider>
  );
}
