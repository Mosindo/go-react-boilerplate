import { createNavigationContainerRef } from "@react-navigation/native";
import type { MainStackParamList } from "./types";

/** Lets code outside a screen (the match celebration) navigate. */
export const navigationRef = createNavigationContainerRef<MainStackParamList>();
