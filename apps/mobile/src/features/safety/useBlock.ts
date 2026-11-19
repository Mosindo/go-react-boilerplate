import { useMutation, useQueryClient } from "@tanstack/react-query";
import { safetyApi } from "../../api/endpoints";
import { queryKeys } from "../../api/queryKeys";
import { useFeedback } from "../../ui";
import { errorMessage } from "../../api/client";

export function useBlock(onDone: () => void) {
  const client = useQueryClient();
  const { toast, confirm } = useFeedback();
  const mutation = useMutation({
    mutationFn: (userId: string) => safetyApi.block(userId),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: queryKeys.conversations });
      void client.invalidateQueries({ queryKey: queryKeys.blocks });
      void client.invalidateQueries({ queryKey: queryKeys.likesReceived });
      toast("Personne bloquée", "success");
      onDone();
    },
    onError: (e) => toast(errorMessage(e), "error")
  });
  return async (userId: string, name: string) => {
    const ok = await confirm({
      title: `Bloquer ${name} ?`,
      message: "Vous ne verrez plus son profil et votre conversation sera fermée. Cette personne n'en sera pas informée.",
      confirmLabel: "Bloquer",
      destructive: true
    });
    if (ok) mutation.mutate(userId);
  };
}
