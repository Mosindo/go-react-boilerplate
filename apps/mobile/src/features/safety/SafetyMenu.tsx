import React, { forwardRef, useImperativeHandle, useState } from "react";
import { useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { chatApi, discoveryApi, safetyApi } from "../../api/endpoints";
import { queryKeys } from "../../api/queryKeys";
import { ActionSheet, confirm, showToast, type SheetAction } from "../../ui";
import { ReportModal } from "./ReportModal";

export type SafetyMenuHandle = { open: () => void };

type Props = {
  userId: string;
  firstName: string;
  matched: boolean;
  /** Called after unmatch / block / report: the member should leave the screen. */
  onRemoved: () => void;
  extraActions?: SheetAction[];
  allowClear?: boolean;
};

export const SafetyMenu = forwardRef<SafetyMenuHandle, Props>(function SafetyMenu(
  { userId, firstName, matched, onRemoved, extraActions = [], allowClear },
  ref
) {
  const queryClient = useQueryClient();
  const [sheetOpen, setSheetOpen] = useState(false);
  const [reportOpen, setReportOpen] = useState(false);

  useImperativeHandle(ref, () => ({ open: () => setSheetOpen(true) }));

  const refreshAfterRemoval = () => {
    void queryClient.invalidateQueries({ queryKey: queryKeys.chats });
    void queryClient.invalidateQueries({ queryKey: queryKeys.blocks });
    queryClient.removeQueries({ queryKey: queryKeys.messages(userId) });
    onRemoved();
  };

  const actions: SheetAction[] = [...extraActions];
  if (allowClear) {
    actions.push({
      label: "Effacer la conversation",
      icon: "trash-bin-outline",
      onPress: async () => {
        if (
          !(await confirm({
            title: "Effacer la conversation ?",
            message: "Les messages seront effacés de votre côté uniquement.",
            confirmLabel: "Effacer",
            destructive: true
          }))
        )
          return;
        try {
          await chatApi.clear(userId);
          await queryClient.invalidateQueries({ queryKey: queryKeys.messages(userId) });
          void queryClient.invalidateQueries({ queryKey: queryKeys.chats });
        } catch (err) {
          showToast(errorMessage(err), "error");
        }
      }
    });
  }
  if (matched) {
    actions.push({
      label: "Annuler le match",
      icon: "heart-dislike-outline",
      testID: "safety-unmatch",
      onPress: async () => {
        if (
          !(await confirm({
            title: `Annuler le match avec ${firstName} ?`,
            message: "La conversation sera supprimée pour vous deux.",
            confirmLabel: "Annuler le match",
            destructive: true
          }))
        )
          return;
        try {
          await discoveryApi.unmatch(userId);
          showToast("Match annulé.", "success");
          refreshAfterRemoval();
        } catch (err) {
          showToast(errorMessage(err), "error");
        }
      }
    });
  }
  actions.push({
    label: `Bloquer ${firstName}`,
    icon: "ban-outline",
    destructive: true,
    testID: "safety-block",
    onPress: async () => {
      if (
        !(await confirm({
          title: `Bloquer ${firstName} ?`,
          message: "Vous ne vous verrez plus et la conversation sera supprimée. Cette personne n'en sera pas informée.",
          confirmLabel: "Bloquer",
          destructive: true
        }))
      )
        return;
      try {
        await safetyApi.block(userId);
        showToast(`${firstName} est bloqué·e.`, "success");
        refreshAfterRemoval();
      } catch (err) {
        showToast(errorMessage(err), "error");
      }
    }
  });
  actions.push({
    label: `Signaler ${firstName}`,
    icon: "flag-outline",
    destructive: true,
    testID: "safety-report",
    onPress: () => setReportOpen(true)
  });

  return (
    <>
      <ActionSheet visible={sheetOpen} title={firstName} actions={actions} onClose={() => setSheetOpen(false)} />
      <ReportModal
        userId={userId}
        firstName={firstName}
        visible={reportOpen}
        onClose={() => setReportOpen(false)}
        onReported={refreshAfterRemoval}
      />
    </>
  );
});
