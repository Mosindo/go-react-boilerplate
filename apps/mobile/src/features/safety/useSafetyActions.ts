import { useCallback } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { safetyApi } from "../../api/endpoints";
import { REPORT_REASONS, type ReportReason } from "../../api/types";
import { errorMessage } from "../../lib/validation";
import { invalidateChat, queryKeys } from "../../realtime/cache";
import { useDialog, useToast } from "../../ui";
import { emitHandled } from "../discover/deckBus";

/** Block and report flows shared by the profile and chat screens. `onDone` runs after a successful block. */
export function useSafetyActions(userId: string, name: string, onDone?: () => void) {
  const dialog = useDialog();
  const toast = useToast();
  const queryClient = useQueryClient();

  const afterBlock = useCallback(() => {
    emitHandled(userId);
    invalidateChat(queryClient);
    void queryClient.invalidateQueries({ queryKey: queryKeys.blocked });
    onDone?.();
  }, [userId, queryClient, onDone]);

  const block = useCallback(async () => {
    const ok = await dialog.confirm({
      title: `Bloquer ${name} ?`,
      message:
        "Vous ne verrez plus cette personne et elle ne pourra plus vous contacter. Le match est supprimé.",
      confirmLabel: "Bloquer",
      destructive: true,
    });
    if (!ok) return;
    try {
      await safetyApi.block(userId);
      toast.show({ message: `${name} est bloqué(e).`, kind: "success" });
      afterBlock();
    } catch (e) {
      toast.show({ message: errorMessage(e), kind: "error" });
    }
  }, [dialog, toast, userId, name, afterBlock]);

  const report = useCallback(async () => {
    const reason = await dialog.choose({
      title: `Signaler ${name}`,
      message: "Pourquoi signalez-vous ce profil ? Notre équipe examinera votre signalement.",
      actions: [
        ...REPORT_REASONS.map((r) => ({
          label: r.label,
          value: r.value,
          variant: "secondary" as const,
        })),
        { label: "Annuler", value: "cancel", variant: "ghost" as const },
      ],
    });
    if (!reason || reason === "cancel") return;
    const alsoBlock = await dialog.confirm({
      title: `Bloquer aussi ${name} ?`,
      message: "Recommandé : vous ne vous verrez plus.",
      confirmLabel: "Signaler et bloquer",
      destructive: true,
    });
    try {
      await safetyApi.report(userId, reason as ReportReason, "", alsoBlock);
      toast.show({ message: "Merci, votre signalement a été envoyé.", kind: "success" });
      if (alsoBlock) afterBlock();
    } catch (e) {
      toast.show({ message: errorMessage(e), kind: "error" });
    }
  }, [dialog, toast, userId, name, afterBlock]);

  return { block, report };
}
