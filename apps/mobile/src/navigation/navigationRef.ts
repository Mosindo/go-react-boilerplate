import { createNavigationContainerRef } from "@react-navigation/native";
import type { AppStackParamList } from "./types";

export const navigationRef = createNavigationContainerRef<AppStackParamList>();

export function openChat(conversationId: string) {
  if (navigationRef.isReady()) {
    navigationRef.navigate("Chat", { conversationId });
  }
}
