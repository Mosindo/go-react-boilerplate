import { useCallback } from "react";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQueryClient } from "@tanstack/react-query";
import { ApiError } from "../../api/client";
import { discoverApi } from "../../api/endpoints";
import type { SwipeAction } from "../../api/types";
import type { MainStackParamList } from "../../navigation/types";
import { invalidateChat } from "../../realtime/cache";
import { useToast } from "../../ui";
import { emitHandled } from "./deckBus";

/** Records a like/pass; opens the match celebration when the interest is mutual. */
export function useSwipe() {
  const navigation = useNavigation<NativeStackNavigationProp<MainStackParamList>>();
  const queryClient = useQueryClient();
  const toast = useToast();

  return useCallback(
    async (userId: string, action: SwipeAction): Promise<boolean> => {
      try {
        const result = await discoverApi.swipe(userId, action);
        emitHandled(userId);
        if (result.matched && result.match) {
          invalidateChat(queryClient);
          navigation.navigate("MatchFound", { match: result.match });
        }
        return true;
      } catch (e) {
        // 404/409 mean the profile is no longer available or was already handled: just drop it.
        if (e instanceof ApiError && (e.status === 404 || e.status === 409)) {
          emitHandled(userId);
          return true;
        }
        toast.show({
          message: e instanceof Error ? e.message : "Action impossible.",
          kind: "error",
        });
        return false;
      }
    },
    [navigation, queryClient, toast],
  );
}
