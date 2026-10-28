import { Alert } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { keys } from "../api/keys";
import { blockUser } from "../api/social";
import type { PublicProfile } from "../api/types";
import { useToast } from "../components";
import { errorMessage } from "../lib/format";

export function useSafety() {
  const queryClient = useQueryClient();
  const toast = useToast();

  const confirmBlock = (userId: string, name: string, afterBlock?: () => void): void => {
    Alert.alert(`Block ${name}?`, "You will no longer see each other, and any conversation will disappear.", [
      { text: "Cancel", style: "cancel" },
      {
        text: "Block",
        style: "destructive",
        onPress: () => {
          blockUser(userId)
            .then(async () => {
              queryClient.setQueryData<PublicProfile[]>(keys.discover, (c) => c?.filter((p) => p.userId !== userId));
              await queryClient.invalidateQueries({ queryKey: keys.conversations });
              await queryClient.invalidateQueries({ queryKey: keys.blocks });
              toast.show(`${name} blocked`);
              afterBlock?.();
            })
            .catch((e: unknown) => toast.show(errorMessage(e), "error"));
        }
      }
    ]);
  };

  return { confirmBlock };
}
