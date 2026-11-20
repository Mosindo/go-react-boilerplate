import { useMutation, useQueryClient } from "@tanstack/react-query";
import { safetyApi } from "../../api/endpoints";
import { errorMessage } from "../../lib/format";
import { keys } from "../../providers/queryClient";
import { showToast } from "../../ui";
import { swipeStore } from "../discovery/swipeStore";

export function useBlock(onDone?: () => void) {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: safetyApi.block,
    onSuccess: (_, userId) => {
      swipeStore.add(userId);
      void queryClient.invalidateQueries({ queryKey: keys.conversationsRoot });
      void queryClient.invalidateQueries({ queryKey: keys.likes });
      void queryClient.invalidateQueries({ queryKey: keys.blocked });
      showToast("Personne bloquée. Vous ne vous verrez plus.", "success");
      onDone?.();
    },
    onError: (error) => showToast(errorMessage(error), "error")
  });
}
