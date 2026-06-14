import { Header } from "./components/Header.js";
import { OutputPreview } from "./components/OutputPreview.js";
import { SettingsPanel } from "./components/SettingsPanel.js";
import { StreamGrid } from "./components/StreamGrid.js";
import { SessionStoreProvider } from "./stores/session-store.js";

export default function App() {
  return (
    <SessionStoreProvider>
      <div class="grid h-screen grid-cols-1 grid-rows-[auto_minmax(0,1fr)_auto_minmax(160px,30vh)] gap-2 overflow-hidden p-2 lg:grid-cols-[1fr_360px] lg:grid-rows-[40px_minmax(0,1fr)_260px]">
        <Header class="lg:col-span-2" />
        <OutputPreview class="min-h-0" />
        <SettingsPanel class="max-h-48 lg:max-h-none" />
        <StreamGrid class="min-h-0 overflow-y-auto lg:col-span-2" />
      </div>
    </SessionStoreProvider>
  );
}
