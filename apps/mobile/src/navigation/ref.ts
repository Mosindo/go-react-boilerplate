import { createNavigationContainerRef } from "@react-navigation/native";
import { getProfile } from "../api/platform";
import { friendlyError } from "../api/errors";
import { queryKeys } from "../hooks/queries";
import { queryClient } from "../hooks/useAuth";
import type { RootStackParamList } from "./types";
import { showAlert } from "../lib/alert";

export const navigationRef = createNavigationContainerRef<RootStackParamList>();

/** Opens a conversation from outside a screen (match modal, notifications). */
export async function openThread(matchId: string, userId: string): Promise<void> {
  try {
    const profile = await queryClient.fetchQuery({
      queryKey: queryKeys.profile(userId),
      queryFn: () => getProfile(userId),
      staleTime: 60_000
    });
    if (navigationRef.isReady()) {
      navigationRef.navigate("Thread", { matchId, profile });
    }
  } catch (error) {
    showAlert("Conversation indisponible", friendlyError(error));
  }
}
