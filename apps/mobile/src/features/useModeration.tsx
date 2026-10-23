import { useMutation, useQueryClient } from "@tanstack/react-query";
import { blockUser, hideMatch, report, unmatch } from "../api";
import type { MatchesPage, ReportReason } from "../api/types";
import { errorMessage } from "../i18n/errorMessage";
import { t } from "../i18n/fr";
import { applyMatchRemoved, type Paged } from "../realtime/cache";
import { queryKeys } from "../session/queryKeys";
import { useDialog, useToast } from "../ui";

export type ModerationTarget = { matchId?: string; userId: string; firstName: string };

const REASONS: ReportReason[] = [
  "harassment",
  "fake_profile",
  "inappropriate_content",
  "spam",
  "underage",
  "other"
];

/** Hide / unmatch / block / report flows shared by the conversation list, chat and profile screens. */
export function useModeration(onRemoved?: () => void) {
  const dialog = useDialog();
  const toast = useToast();
  const queryClient = useQueryClient();

  const dropMatch = (matchId: string) => {
    queryClient.setQueryData<Paged<MatchesPage>>(queryKeys.matches, (data) =>
      applyMatchRemoved(data, matchId)
    );
    queryClient.removeQueries({ queryKey: queryKeys.messages(matchId) });
    void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
    onRemoved?.();
  };

  const hide = useMutation({
    mutationFn: (target: ModerationTarget) => hideMatch(target.matchId as string),
    onSuccess: (_r, target) => {
      toast.success(t.moderation.hidden);
      dropMatch(target.matchId as string);
    },
    onError: (error) => toast.error(errorMessage(error))
  });

  const remove = useMutation({
    mutationFn: (target: ModerationTarget) => unmatch(target.matchId as string),
    onSuccess: (_r, target) => {
      toast.success(t.moderation.unmatched);
      dropMatch(target.matchId as string);
    },
    onError: (error) => toast.error(errorMessage(error))
  });

  const block = useMutation({
    mutationFn: (target: ModerationTarget) => blockUser(target.userId),
    onSuccess: (_r, target) => {
      toast.success(t.moderation.blocked);
      if (target.matchId) dropMatch(target.matchId);
      else onRemoved?.();
      void queryClient.invalidateQueries({ queryKey: queryKeys.blocks });
      void queryClient.invalidateQueries({ queryKey: queryKeys.discover });
    },
    onError: (error) => toast.error(errorMessage(error))
  });

  const sendReport = useMutation({
    mutationFn: ({ target, reason }: { target: ModerationTarget; reason: ReportReason }) =>
      report({ userId: target.userId, reason, matchId: target.matchId }),
    onSuccess: () => toast.success(t.moderation.reported),
    onError: (error) => toast.error(errorMessage(error))
  });

  const askReport = (target: ModerationTarget) =>
    dialog.showSheet({
      title: t.moderation.reportTitle(target.firstName),
      message: t.moderation.reportText,
      actions: REASONS.map((reason) => ({
        label: t.moderation.reasons[reason],
        onPress: () => sendReport.mutate({ target, reason })
      }))
    });

  const askBlock = (target: ModerationTarget) =>
    dialog.confirm({
      title: t.moderation.blockConfirmTitle(target.firstName),
      message: t.moderation.blockConfirmText,
      confirmLabel: t.moderation.block,
      destructive: true,
      onConfirm: () => block.mutate(target)
    });

  const askUnmatch = (target: ModerationTarget) =>
    dialog.confirm({
      title: t.moderation.unmatchConfirmTitle,
      message: t.moderation.unmatchConfirmText,
      confirmLabel: t.moderation.unmatch,
      destructive: true,
      onConfirm: () => remove.mutate(target)
    });

  const askHide = (target: ModerationTarget) =>
    dialog.confirm({
      title: t.moderation.hideConfirmTitle,
      message: t.moderation.hideConfirmText,
      confirmLabel: t.moderation.hide,
      onConfirm: () => hide.mutate(target)
    });

  /** Action sheet with every applicable action for the target. */
  const openMenu = (
    target: ModerationTarget,
    options: { title?: string; onViewProfile?: () => void; withHide?: boolean } = {}
  ) => {
    const actions = [];
    if (options.onViewProfile) {
      actions.push({ label: t.moderation.viewProfile, onPress: options.onViewProfile });
    }
    if (target.matchId && options.withHide) {
      actions.push({ label: t.moderation.hide, onPress: () => askHide(target) });
    }
    if (target.matchId) {
      actions.push({
        label: t.moderation.unmatch,
        onPress: () => askUnmatch(target),
        destructive: true
      });
    }
    actions.push({
      label: t.moderation.report,
      onPress: () => askReport(target),
      destructive: true
    });
    actions.push({ label: t.moderation.block, onPress: () => askBlock(target), destructive: true });
    dialog.showSheet({ title: options.title ?? target.firstName, actions });
  };

  return { openMenu, askBlock, askReport, askUnmatch, askHide };
}
