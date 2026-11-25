import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../api/client";
import { matchingApi } from "../api/matching";
import { safetyApi } from "../api/safety";
import type { ReportReason } from "../api/types";
import { queryKeys } from "../hooks/queryClient";
import { spacing } from "../theme";
import { Button, Chip, Dialog, Sheet, Text, TextField, showToast } from "../ui";
import { reportReasonLabels, reportReasons } from "../utils/labels";

type SafetyActionsProps = {
  visible: boolean;
  onClose: () => void;
  userId: string;
  firstName: string;
  matchId?: string;
  /** Adds a "clear conversation" entry (chat screen only). */
  onClearConversation?: () => void;
  /** Called after the person was blocked, reported or unmatched. */
  onDone: () => void;
};

/** Report / block / unmatch actions for another member. */
export function SafetyActions({ visible, onClose, userId, firstName, matchId, onClearConversation, onDone }: SafetyActionsProps) {
  const qc = useQueryClient();
  const [mode, setMode] = useState<"menu" | "report" | "block" | "unmatch">("menu");
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");

  const refresh = () => {
    void qc.invalidateQueries({ queryKey: queryKeys.conversations });
    void qc.invalidateQueries({ queryKey: queryKeys.discover });
    void qc.invalidateQueries({ queryKey: queryKeys.blocks });
  };

  const close = () => {
    setMode("menu");
    setReason(null);
    setDetails("");
    onClose();
  };

  const action = useMutation({
    mutationFn: async (kind: "report" | "block" | "unmatch") => {
      if (kind === "report" && reason) {
        await safetyApi.report(userId, reason, details.trim(), true);
      } else if (kind === "block") {
        await safetyApi.block(userId);
      } else if (kind === "unmatch" && matchId) {
        await matchingApi.unmatch(matchId);
      }
      return kind;
    },
    onSuccess: (kind) => {
      refresh();
      close();
      showToast(
        kind === "report"
          ? "Merci. Le signalement a été envoyé et la personne est bloquée."
          : kind === "block"
            ? `${firstName} est bloqué·e. Vous ne vous verrez plus.`
            : "Le match a été annulé.",
        "success"
      );
      onDone();
    },
    onError: (error) => showToast(errorMessage(error), "error")
  });

  return (
    <>
      <Sheet visible={visible && mode === "menu"} title={firstName} onClose={close}>
        <View style={styles.list}>
          {onClearConversation ? (
            <Button
              label="Effacer la conversation (de mon côté)"
              variant="secondary"
              icon="trash-outline"
              onPress={onClearConversation}
              testID="chat-clear"
            />
          ) : null}
          {matchId ? (
            <Button
              label="Annuler le match"
              variant="secondary"
              icon="heart-dislike-outline"
              onPress={() => setMode("unmatch")}
              testID="safety-unmatch"
            />
          ) : null}
          <Button
            label={`Bloquer ${firstName}`}
            variant="secondary"
            icon="ban-outline"
            onPress={() => setMode("block")}
            testID="safety-block"
          />
          <Button label="Signaler" variant="danger" icon="flag-outline" onPress={() => setMode("report")} testID="safety-report" />
        </View>
      </Sheet>

      <Sheet visible={visible && mode === "report"} title={`Signaler ${firstName}`} onClose={close}>
        <Text variant="small" color="textMuted">
          Votre signalement est confidentiel. La personne sera aussi bloquée.
        </Text>
        <View style={styles.chips}>
          {reportReasons.map((r) => (
            <Chip key={r} label={reportReasonLabels[r]} selected={reason === r} onPress={() => setReason(r)} testID={`report-${r}`} />
          ))}
        </View>
        <TextField label="Détails (optionnel)" value={details} onChangeText={setDetails} multiline maxLength={1000} showCounter />
        <Button
          label="Envoyer le signalement"
          variant="danger"
          disabled={!reason}
          loading={action.isPending}
          onPress={() => action.mutate("report")}
          testID="report-submit"
        />
      </Sheet>

      <Dialog
        visible={visible && mode === "block"}
        title={`Bloquer ${firstName} ?`}
        message="Vous ne vous verrez plus et votre conversation sera fermée. Vous pourrez débloquer cette personne dans Confidentialité."
        confirmLabel="Bloquer"
        destructive
        loading={action.isPending}
        onConfirm={() => action.mutate("block")}
        onCancel={close}
        testID="block-dialog"
      />
      <Dialog
        visible={visible && mode === "unmatch"}
        title="Annuler ce match ?"
        message={`La conversation avec ${firstName} sera fermée pour vous deux.`}
        confirmLabel="Annuler le match"
        destructive
        loading={action.isPending}
        onConfirm={() => action.mutate("unmatch")}
        onCancel={close}
      />
    </>
  );
}

const styles = StyleSheet.create({
  list: {
    gap: spacing.sm
  },
  chips: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: spacing.xs
  }
});
