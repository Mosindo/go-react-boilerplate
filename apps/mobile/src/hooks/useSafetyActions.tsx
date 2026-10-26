import React, { useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { discoverApi, safetyApi, type PublicProfile, type ReportReason } from "../api";
import { useFeedback } from "../components/Feedback";
import { ReportSheet } from "../components/Sheets";
import { queryKeys } from "../lib/queryKeys";

/**
 * Block / report / unmatch flows shared by the chat and profile screens.
 * `onLeave` runs once the relationship no longer exists (the screen should close).
 */
export function useSafetyActions(profile: PublicProfile, matchId: string | undefined, onLeave: () => void) {
  const queryClient = useQueryClient();
  const { confirm, toast } = useFeedback();
  const [reportOpen, setReportOpen] = useState(false);

  const refreshLists = () => {
    void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
    void queryClient.invalidateQueries({ queryKey: queryKeys.notifications });
    void queryClient.invalidateQueries({ queryKey: queryKeys.blocks });
  };

  const block = async () => {
    const ok = await confirm({
      title: `Bloquer ${profile.firstName} ?`,
      message:
        "Vous ne vous verrez plus et la conversation sera fermée. Vous pourrez débloquer cette personne dans les réglages.",
      confirmLabel: "Bloquer",
      destructive: true
    });
    if (!ok) return;
    try {
      await safetyApi.block(profile.userId);
      refreshLists();
      toast(`${profile.firstName} est bloqué(e).`, "success");
      onLeave();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Blocage impossible.", "error");
    }
  };

  const unmatch = async () => {
    if (!matchId) return;
    const ok = await confirm({
      title: `Retirer ${profile.firstName} de vos matchs ?`,
      message: "La conversation sera supprimée pour vous deux.",
      confirmLabel: "Retirer",
      destructive: true
    });
    if (!ok) return;
    try {
      await discoverApi.unmatch(matchId);
      refreshLists();
      onLeave();
    } catch (err) {
      toast(err instanceof Error ? err.message : "Action impossible.", "error");
    }
  };

  const submitReport = async (reason: ReportReason, details: string) => {
    await safetyApi.report(profile.userId, reason, details);
    toast("Merci, votre signalement a été transmis.", "success");
  };

  const reportSheet = (
    <ReportSheet
      visible={reportOpen}
      name={profile.firstName}
      onClose={() => setReportOpen(false)}
      onSubmit={submitReport}
    />
  );

  return { block, unmatch, openReport: () => setReportOpen(true), reportSheet };
}
