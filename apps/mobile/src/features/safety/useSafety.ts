import { useMutation, useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { safetyApi } from "../../api/endpoints";
import { queryKeys } from "../../api/queryKeys";
import { showToast } from "../../ui";
import { confirmAction } from "../../utils/confirm";

export function useBlockUser(onBlocked?: () => void) {
  const queryClient = useQueryClient();
  const mutation = useMutation({
    mutationFn: (userId: string) => safetyApi.block(userId),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      void queryClient.invalidateQueries({ queryKey: queryKeys.blocks });
      void queryClient.invalidateQueries({ queryKey: queryKeys.discover });
      showToast("Membre bloqué. Vous ne vous verrez plus.", "success");
      onBlocked?.();
    },
    onError: (error) => showToast(errorMessage(error), "error")
  });
  return {
    isPending: mutation.isPending,
    confirm: async (userId: string, firstName: string) => {
      const ok = await confirmAction(
        `Bloquer ${firstName} ?`,
        "Vous ne pourrez plus vous voir ni vous écrire. Votre match et votre conversation seront fermés.",
        "Bloquer"
      );
      if (ok) mutation.mutate(userId);
    }
  };
}
