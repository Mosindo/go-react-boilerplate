import { useMutation } from "@tanstack/react-query";
import { Alert, Platform } from "react-native";
import { errorMessage } from "../../api/client";
import { matchesApi, safetyApi } from "../../api/endpoints";
import type { ReportReason } from "../../api/types";
import { keys, queryClient } from "../../hooks/queryClient";
import { toast } from "../../ui";

/** Cross-platform confirmation dialog. */
export function confirm(title: string, message: string, confirmLabel: string): Promise<boolean> {
  if (Platform.OS === "web") {
    return Promise.resolve(typeof window !== "undefined" ? window.confirm(`${title}\n\n${message}`) : true);
  }
  return new Promise((resolve) => {
    Alert.alert(title, message, [
      { text: "Annuler", style: "cancel", onPress: () => resolve(false) },
      { text: confirmLabel, style: "destructive", onPress: () => resolve(true) }
    ]);
  });
}

function afterRelationshipChange(userId: string) {
  void queryClient.invalidateQueries({ queryKey: keys.matches });
  void queryClient.invalidateQueries({ queryKey: keys.blocks });
  queryClient.removeQueries({ queryKey: keys.profile(userId) });
}

export function useSafetyActions(onDone?: () => void) {
  const block = useMutation({
    mutationFn: (userId: string) => safetyApi.block(userId),
    onSuccess: (_, userId) => {
      afterRelationshipChange(userId);
      toast("Personne bloquée. Vous ne vous verrez plus.", "success");
      onDone?.();
    },
    onError: (e) => toast(errorMessage(e), "error")
  });

  const report = useMutation({
    mutationFn: (input: { userId: string; reason: ReportReason; details: string; block: boolean }) =>
      safetyApi.report(input.userId, input.reason, input.details, input.block),
    onSuccess: (result, input) => {
      afterRelationshipChange(input.userId);
      toast(result.blocked ? "Merci. Signalement envoyé et personne bloquée." : "Merci, notre équipe va examiner ce signalement.", "success");
      if (result.blocked) {
        onDone?.();
      }
    },
    onError: (e) => toast(errorMessage(e), "error")
  });

  const unmatch = useMutation({
    mutationFn: (matchId: string) => matchesApi.unmatch(matchId),
    onSuccess: (_, matchId) => {
      queryClient.removeQueries({ queryKey: keys.messages(matchId) });
      void queryClient.invalidateQueries({ queryKey: keys.matches });
      toast("Match annulé.", "info");
      onDone?.();
    },
    onError: (e) => toast(errorMessage(e), "error")
  });

  const confirmBlock = async (userId: string, name: string) => {
    if (await confirm(`Bloquer ${name} ?`, "Vous ne pourrez plus vous voir ni vous écrire. Votre match éventuel sera supprimé.", "Bloquer")) {
      block.mutate(userId);
    }
  };

  const confirmUnmatch = async (matchId: string, name: string) => {
    if (await confirm(`Annuler le match avec ${name} ?`, "La conversation sera supprimée pour vous deux.", "Annuler le match")) {
      unmatch.mutate(matchId);
    }
  };

  return { block, report, unmatch, confirmBlock, confirmUnmatch };
}
