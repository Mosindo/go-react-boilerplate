import { useMutation, useQueryClient } from "@tanstack/react-query";
import { describeError } from "../api/client";
import { keys } from "../api/keys";
import * as api from "../api/platform";
import { showToast } from "../shared/feedback";
import { confirm } from "../utils/confirm";

/** Block / unmatch flows shared by the profile and chat screens. Each resolves true when it changed something. */
export function useUserActions() {
  const client = useQueryClient();

  const refresh = () => {
    for (const key of [keys.matches, keys.conversations, keys.unread, keys.notifications, keys.notificationsUnread, keys.discover]) {
      void client.invalidateQueries({ queryKey: key });
    }
  };

  const block = useMutation({
    mutationFn: (userId: string) => api.blockUser(userId),
    onSuccess: refresh
  });
  const unmatch = useMutation({
    mutationFn: (matchId: string) => api.unmatch(matchId),
    onSuccess: refresh
  });
  const hide = useMutation({
    mutationFn: (conversationId: string) => api.hideConversation(conversationId),
    onSuccess: refresh
  });

  return {
    async blockUser(userId: string, firstName: string): Promise<boolean> {
      const ok = await confirm({
        title: `Bloquer ${firstName} ?`,
        message: "Cette personne ne pourra plus vous voir ni vous écrire, et votre conversation sera supprimée.",
        confirmLabel: "Bloquer",
        destructive: true
      });
      if (!ok) return false;
      try {
        await block.mutateAsync(userId);
        showToast(`${firstName} est bloqué(e).`, "success");
        return true;
      } catch (error) {
        showToast(describeError(error), "error");
        return false;
      }
    },
    async unmatchUser(matchId: string, firstName: string): Promise<boolean> {
      const ok = await confirm({
        title: `Annuler le match avec ${firstName} ?`,
        message: "La conversation sera supprimée pour vous deux. Vous ne vous reverrez plus dans la découverte.",
        confirmLabel: "Annuler le match",
        destructive: true
      });
      if (!ok) return false;
      try {
        await unmatch.mutateAsync(matchId);
        showToast("Match annulé.", "success");
        return true;
      } catch (error) {
        showToast(describeError(error), "error");
        return false;
      }
    },
    async hideConversation(conversationId: string): Promise<boolean> {
      const ok = await confirm({
        title: "Supprimer cette conversation ?",
        message: "Elle disparaît de votre liste. L'autre personne la conserve, et elle réapparaîtra si elle vous écrit.",
        confirmLabel: "Supprimer",
        destructive: true
      });
      if (!ok) return false;
      try {
        await hide.mutateAsync(conversationId);
        return true;
      } catch (error) {
        showToast(describeError(error), "error");
        return false;
      }
    }
  };
}
