import { createNavigationContainerRef } from "@react-navigation/native";
import type { RootStackParamList } from "./types";

/** Lets code outside screens (toasts, realtime events) navigate. */
export const navigationRef = createNavigationContainerRef<RootStackParamList>();
