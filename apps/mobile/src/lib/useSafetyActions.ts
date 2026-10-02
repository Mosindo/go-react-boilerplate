import { useQueryClient } from "@tanstack/react-query";
import { useCallback } from "react";

import { safetyApi } from "../api/endpoints";
import type { ReportReason } from "../api/types";
import { keys } from "../realtime/RealtimeProvider";
import { useFeedback } from "../ui/Feedback";
import { errorMessage } from "../ui/States";

const REASONS: { value: ReportReason; label: string }[] = [
  { value: "fake", label: "Faux profil / arnaque" },
  { value: "inappropriate", label: "Contenu inapproprié" },
  { value: "harassment", label: "Harcèlement" },
  { value: "spam", label: "Spam" },
  { value: "underage", label: "Personne mineure" },
  { value: "other", label: "Autre" },
];

/** Block and report flows, shared by the profile screen and the chat. */
export function useSafetyActions() {
  const qc = useQueryClient();
  const { toast, sheet, confirm } = useFeedback();

  const refreshAfterBlock = useCallback(() => {
    void qc.invalidateQueries({ queryKey: keys.matches });
    void qc.invalidateQueries({ queryKey: keys.discover });
    void qc.invalidateQueries({ queryKey: keys.summary });
    void qc.invalidateQueries({ queryKey: keys.blocked });
  }, [qc]);

  const block = useCallback(
    async (userId: string, name: string, onDone?: () => void) => {
      const ok = await confirm({
        title: `Bloquer ${name} ?`,
        message: "Vous ne vous verrez plus et la conversation éventuelle sera supprimée.",
        confirmLabel: "Bloquer",
        destructive: true,
      });
      if (!ok) return;
      try {
        await safetyApi.block(userId);
        refreshAfterBlock();
        toast(`${name} est bloqué(e).`, "success");
        onDone?.();
      } catch (err) {
        toast(errorMessage(err), "error");
      }
    },
    [confirm, refreshAfterBlock, toast],
  );

  const report = useCallback(
    (userId: string, name: string, onDone?: () => void) => {
      sheet(
        `Signaler ${name}`,
        REASONS.map((r) => ({
          label: r.label,
          onPress: async () => {
            try {
              await safetyApi.report(userId, r.value, "", false);
              toast("Merci, votre signalement a été envoyé.", "success");
              const alsoBlock = await confirm({
                title: `Bloquer ${name} aussi ?`,
                message: "Recommandé : vous ne vous verrez plus.",
                confirmLabel: "Bloquer",
                destructive: true,
              });
              if (alsoBlock) {
                await safetyApi.block(userId);
                refreshAfterBlock();
                onDone?.();
              }
            } catch (err) {
              toast(errorMessage(err), "error");
            }
          },
        })),
        "Pourquoi signalez-vous ce profil ? Notre équipe examinera votre signalement.",
      );
    },
    [sheet, confirm, toast, refreshAfterBlock],
  );

  return { block, report };
}
