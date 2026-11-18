import React, { useState } from "react";
import { View } from "react-native";
import { errorMessage } from "../api/errors";
import type { ReportReason } from "../api/types";
import { afterModalDismiss, Sheet, useFeedback } from "../feedback";
import { useBlock, useReport, useUnmatch } from "../hooks/queries";
import { spacing } from "../theme";
import { Button, ListRow, Text, TextField } from "../ui";
import { reportReasonLabels, reportReasons } from "../utils/labels";

type Props = {
  visible: boolean;
  userId: string;
  firstName: string;
  isMatch: boolean;
  onClose: () => void;
  /** Called after the member disappeared from the current context. */
  onRemoved: () => void;
};

/** Unmatch, block and report: always two taps away from any profile or chat. */
export function SafetySheet({ visible, userId, firstName, isMatch, onClose, onRemoved }: Props) {
  const { toast, confirm } = useFeedback();
  const unmatch = useUnmatch();
  const block = useBlock();
  const report = useReport();
  const [mode, setMode] = useState<"menu" | "report">("menu");
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");

  const close = () => {
    setMode("menu");
    setReason(null);
    setDetails("");
    onClose();
  };

  const doUnmatch = async () => {
    close();
    await afterModalDismiss();
    const ok = await confirm({
      title: `Annuler le match avec ${firstName} ?`,
      message: "La conversation sera supprimée pour vous deux.",
      confirmLabel: "Annuler le match",
      destructive: true
    });
    if (!ok) {
      return;
    }
    try {
      await unmatch.mutateAsync(userId);
      toast("Match annulé", "success");
      onRemoved();
    } catch (e) {
      toast(errorMessage(e), "error");
    }
  };

  const doBlock = async () => {
    close();
    await afterModalDismiss();
    const ok = await confirm({
      title: `Bloquer ${firstName} ?`,
      message: "Vous ne vous verrez plus et ne pourrez plus vous écrire. Cette personne ne sera pas prévenue.",
      confirmLabel: "Bloquer",
      destructive: true
    });
    if (!ok) {
      return;
    }
    try {
      await block.mutateAsync(userId);
      toast(`${firstName} est bloqué·e`, "success");
      onRemoved();
    } catch (e) {
      toast(errorMessage(e), "error");
    }
  };

  const doReport = async () => {
    if (!reason) {
      return;
    }
    try {
      await report.mutateAsync({ userId, reason, details: details.trim() });
      close();
      toast("Merci. Notre équipe va examiner ce signalement.", "success");
      onRemoved();
    } catch (e) {
      toast(errorMessage(e), "error");
    }
  };

  return (
    <Sheet onClose={close} testID="safety-sheet" title={mode === "menu" ? firstName : `Signaler ${firstName}`} visible={visible}>
      {mode === "menu" ? (
        <>
          {isMatch ? <ListRow icon="heart-dislike-outline" onPress={doUnmatch} testID="safety-unmatch" title="Annuler le match" /> : null}
          <ListRow
            icon="ban-outline"
            onPress={doBlock}
            subtitle="Discret : la personne n’est pas prévenue"
            testID="safety-block"
            title="Bloquer"
            tone="danger"
          />
          <ListRow
            icon="flag-outline"
            onPress={() => setMode("report")}
            subtitle="Bloque aussi la personne"
            testID="safety-report"
            title="Signaler"
            tone="danger"
          />
        </>
      ) : (
        <View style={{ gap: spacing.sm }}>
          <Text color="textMuted">Votre signalement est confidentiel. Que se passe-t-il ?</Text>
          {reportReasons.map((r) => (
            <ListRow
              key={r}
              icon={reason === r ? "radio-button-on" : "radio-button-off"}
              onPress={() => setReason(r)}
              testID={`report-reason-${r}`}
              title={reportReasonLabels[r]}
              tone={reason === r ? "accent" : "text"}
            />
          ))}
          <TextField label="Détails (facultatif)" maxLength={1000} multiline onChangeText={setDetails} value={details} />
          <Button disabled={!reason} label="Envoyer le signalement" loading={report.isPending} onPress={doReport} testID="report-submit" />
          <Button label="Retour" onPress={() => setMode("menu")} variant="ghost" />
        </View>
      )}
    </Sheet>
  );
}
