import { Header } from "./components/Header.js";
import { OutputPreview } from "./components/OutputPreview.js";
import {
  DevicesPanel,
  MixerPanel,
  SettingsPanel,
} from "./components/SettingsPanel.js";
import { StreamGrid } from "./components/StreamGrid.js";
import { SessionStoreProvider } from "./stores/session-store.js";

export default function App() {
  return (
    <SessionStoreProvider>
      <div class="grid h-screen grid-cols-[1fr_minmax(160px,38vw)] grid-rows-[auto_minmax(0,1fr)_auto_minmax(0,1fr)] gap-2 overflow-hidden p-2 lg:grid-cols-[1fr_360px] lg:grid-rows-[40px_minmax(0,1fr)_260px]">
        <Header class="col-span-2 lg:col-span-2" />
        <OutputPreview class="col-span-2 min-h-0" />
        <DevicesPanel class="col-start-1 row-start-3 min-h-0 overflow-y-auto lg:hidden" />
        <MixerPanel class="col-start-2 row-span-2 row-start-3 min-h-0 self-stretch overflow-y-auto lg:hidden" />
        <SettingsPanel class="hidden min-h-0 lg:col-start-2 lg:row-start-2 lg:flex" />
        <StreamGrid class="col-start-1 row-start-4 min-h-0 overflow-y-auto lg:col-span-2 lg:row-start-3" />
      </div>
    </SessionStoreProvider>
  );
}
