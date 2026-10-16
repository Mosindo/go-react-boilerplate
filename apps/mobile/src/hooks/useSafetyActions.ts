import { useCallback, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { friendlyError } from "../api/errors";
import { blockUser, clearConversation, reportUser, unmatch } from "../api/platform";
import type { ReportReason } from "../api/types";
import { markDecided } from "../lib/decisions";
import { queryKeys } from "./queries";
import { showAlert } from "../lib/alert";

type Options = {
  userId: string;
  name: string;
  matchId?: string;
  /** Called once the other person is gone from the user's world (blocked, unmatched). */
  onGone: () => void;
};

/** Block / report / unmatch / clear, with the confirmations and cache updates they need. */
export function useSafetyActions({ userId, name, matchId, onGone }: Options) {
  const queryClient = useQueryClient();
  const [reportOpen, setReportOpen] = useState(false);
  const [reporting, setReporting] = useState(false);

  const refreshLists = useCallback(() => {
    for (const key of [
      queryKeys.matches,
      queryKeys.conversations,
      queryKeys.likes,
      queryKeys.unread,
      queryKeys.blocked,
      queryKeys.notifications
    ]) {
      void queryClient.invalidateQueries({ queryKey: key });
    }
  }, [queryClient]);

  const fail = (error: unknown) => showAlert("Action impossible", friendlyError(error));

  const confirmBlock = useCallback(() => {
    showAlert(
      `Bloquer ${name} ?`,
      "Vous ne vous verrez plus, et votre conversation sera supprimée pour vous deux. Vous pourrez débloquer cette personne dans les réglages.",
      [
        { text: "Annuler", style: "cancel" },
        {
          text: "Bloquer",
          style: "destructive",
          onPress: () => {
            blockUser(userId)
              .then(() => {
                markDecided(userId);
                refreshLists();
                onGone();
              })
              .catch(fail);
          }
        }
      ]
    );
  }, [name, onGone, refreshLists, userId]);

  const submitReport = useCallback(
    (reason: ReportReason, details: string) => {
      setReporting(true);
      reportUser(userId, reason, details)
        .then(() => {
          setReportOpen(false);
          showAlert("Merci", "Votre signalement a bien été envoyé. Souhaitez-vous aussi bloquer cette personne ?", [
            { text: "Pas maintenant", style: "cancel" },
            { text: "Bloquer", style: "destructive", onPress: confirmBlock }
          ]);
        })
        .catch(fail)
        .finally(() => setReporting(false));
    },
    [confirmBlock, userId]
  );

  const confirmUnmatch = useCallback(() => {
    if (!matchId) {
      return;
    }
    showAlert(`Annuler le match avec ${name} ?`, "La conversation sera supprimée pour vous deux.", [
      { text: "Garder", style: "cancel" },
      {
        text: "Annuler le match",
        style: "destructive",
        onPress: () => {
          unmatch(matchId)
            .then(() => {
              refreshLists();
              onGone();
            })
            .catch(fail);
        }
      }
    ]);
  }, [matchId, name, onGone, refreshLists]);

  const confirmClear = useCallback(() => {
    if (!matchId) {
      return;
    }
    showAlert("Supprimer la conversation ?", "Les messages disparaissent de votre côté uniquement.", [
      { text: "Annuler", style: "cancel" },
      {
        text: "Supprimer",
        style: "destructive",
        onPress: () => {
          clearConversation(matchId)
            .then(() => {
              void queryClient.invalidateQueries({ queryKey: queryKeys.messages(matchId) });
              refreshLists();
              onGone();
            })
            .catch(fail);
        }
      }
    ]);
  }, [matchId, onGone, queryClient, refreshLists]);

  return {
    reportOpen,
    reporting,
    openReport: () => setReportOpen(true),
    closeReport: () => setReportOpen(false),
    submitReport,
    confirmBlock,
    confirmUnmatch,
    confirmClear
  };
}
