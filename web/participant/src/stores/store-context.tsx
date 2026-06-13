import { createContext, useContext, type ParentProps } from "solid-js";
import type { ParticipantStore } from "../stores/participant-store.js";

const StoreContext = createContext<ParticipantStore>();

export function StoreProvider(props: ParentProps<{ store: ParticipantStore }>) {
  return (
    <StoreContext.Provider value={props.store}>
      {props.children}
    </StoreContext.Provider>
  );
}

export function useStore(): ParticipantStore {
  const store = useContext(StoreContext);
  if (!store) {
    throw new Error("StoreProvider required");
  }
  return store;
}
