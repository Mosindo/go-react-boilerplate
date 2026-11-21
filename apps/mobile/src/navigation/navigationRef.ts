import { createNavigationContainerRef } from "@react-navigation/native";
import type { RootStackParamList } from "./types";

export const navigationRef = createNavigationContainerRef<RootStackParamList>();

export function navigateToConversation(userId: string): void {
  if (navigationRef.isReady()) {
    navigationRef.navigate("Conversation", { userId });
  }
}
