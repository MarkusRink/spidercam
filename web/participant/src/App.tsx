import { onMount } from "solid-js";
import { LiveParticipantPeer } from "./adapters/live-peer.js";
import { LiveParticipantSignaling } from "./adapters/live-participant-signaling.js";
import { ParticipantShell } from "./components/ParticipantShell.js";
import { createParticipantStore } from "./stores/participant-store.js";
import { StoreProvider } from "./stores/store-context.js";

const signaling = new LiveParticipantSignaling();
const peer = new LiveParticipantPeer(signaling);
const store = createParticipantStore({ signaling, peer });

export default function App() {
  onMount(() => {
    void store.init();
  });

  return (
    <StoreProvider store={store}>
      <ParticipantShell />
    </StoreProvider>
  );
}
