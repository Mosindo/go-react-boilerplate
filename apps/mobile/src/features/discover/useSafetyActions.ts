import { useMutation, useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { safetyApi } from "../../api/endpoints";
import { keys } from "../../hooks/queries";
import { confirm } from "../../shared/feedback/confirm";
import { useToast } from "../../shared/ui";

/** Block confirmation + cache cleanup shared by the profile and chat screens. */
export function useBlock(userId: string, name: string, onDone: () => void) {
  const qc = useQueryClient();
  const toast = useToast();
  const block = useMutation({
    mutationFn: () => safetyApi.block(userId),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: keys.conversations });
      void qc.invalidateQueries({ queryKey: keys.likes });
      void qc.invalidateQueries({ queryKey: keys.blocked });
      toast.show(`${name} est bloqué(e).`, "success");
      onDone();
    },
    onError: (err) => toast.show(errorMessage(err), "error"),
  });
  return () => confirm(`Bloquer ${name} ?`, "Vous ne verrez plus cette personne et elle ne pourra plus vous écrire. Vous pourrez la débloquer dans vos réglages.", "Bloquer", () => block.mutate());
}
