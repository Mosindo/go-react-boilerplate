import React, { useState } from "react";
import { Alert } from "react-native";
import { useBlockUser, useReportUser } from "./hooks";
import { ReportModal } from "./ReportModal";

type Config = {
  userId: string;
  firstName: string;
  conversationId?: string;
  /** Called after a successful block, to leave the current screen. */
  onBlocked: () => void;
};

/** Block (with confirmation) and report (with reason picker) actions for one person, shared by chat and profile. */
export function usePersonSafety({ userId, firstName, conversationId, onBlocked }: Config) {
  const [reporting, setReporting] = useState(false);
  const block = useBlockUser(onBlocked);
  const report = useReportUser(() => setReporting(false));

  const confirmBlock = () => {
    Alert.alert(
      `Bloquer ${firstName} ?`,
      "Vous ne verrez plus cette personne et elle ne pourra plus vous voir. Votre conversation éventuelle sera supprimée.",
      [
        { text: "Annuler", style: "cancel" },
        { text: "Bloquer", style: "destructive", onPress: () => block.mutate({ userId, conversationId }) }
      ]
    );
  };

  const reportModal = (
    <ReportModal
      firstName={firstName}
      onClose={() => setReporting(false)}
      onSubmit={(reason, details) => report.mutate({ userId, reason, details: details || undefined })}
      submitting={report.isPending}
      visible={reporting}
    />
  );

  return { confirmBlock, openReport: () => setReporting(true), reportModal };
}
