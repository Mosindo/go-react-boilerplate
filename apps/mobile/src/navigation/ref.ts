import { createNavigationContainerRef } from "@react-navigation/native";
import type { MainStackParamList } from "./types";

/** Lets code outside screens (match celebration, deep links) navigate in the signed-in stack. */
export const navigationRef = createNavigationContainerRef<MainStackParamList>();
