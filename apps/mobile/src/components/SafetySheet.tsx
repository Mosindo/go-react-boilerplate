import React, { useState } from "react";
import { View } from "react-native";
import type { ReportReason } from "../api/types";
import { useSafetyMutations } from "../hooks/queries";
import { spacing } from "../theme/tokens";
import { Button, ConfirmDialog, Sheet, Text, TextField, showToast } from "../ui";
import { errorMessage } from "../utils/errors";
import { reportReasonLabels } from "../utils/format";
import { OptionList } from "./OptionList";

type Props = {
  visible: boolean;
  userId: string;
  firstName: string;
  matchId?: string;
  onClose: () => void;
  /** Called after the person is no longer reachable (unmatch/block/report). */
  onDone: () => void;
};

type Mode = "menu" | "report" | "confirmBlock" | "confirmUnmatch";

const reasons = Object.entries(reportReasonLabels).map(([value, label]) => ({ value: value as ReportReason, label }));

export function SafetySheet({ visible, userId, firstName, matchId, onClose, onDone }: Props) {
  const [mode, setMode] = useState<Mode>("menu");
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");
  const { block, report, unmatch } = useSafetyMutations();

  const close = () => {
    setMode("menu");
    setReason(null);
    setDetails("");
    onClose();
  };

  const run = async (fn: () => Promise<unknown>, success: string) => {
    try {
      await fn();
      showToast(success, "success");
      close();
      onDone();
    } catch (err) {
      showToast(errorMessage(err), "error");
    }
  };

  if (mode === "confirmBlock") {
    return (
      <ConfirmDialog
        visible={visible}
        title={`Bloquer ${firstName} ?`}
        message="Vous ne vous verrez plus nulle part et la conversation sera fermée. Cette personne ne sera pas prévenue."
        confirmLabel="Bloquer"
        destructive
        loading={block.isPending}
        onCancel={() => setMode("menu")}
        onConfirm={() => run(() => block.mutateAsync(userId), `${firstName} a été bloqué·e`)}
      />
    );
  }

  if (mode === "confirmUnmatch" && matchId) {
    return (
      <ConfirmDialog
        visible={visible}
        title={`Annuler le match avec ${firstName} ?`}
        message="La conversation disparaîtra pour vous deux. Cette action est définitive."
        confirmLabel="Annuler le match"
        destructive
        loading={unmatch.isPending}
        onCancel={() => setMode("menu")}
        onConfirm={() => run(() => unmatch.mutateAsync(matchId), "Match annulé")}
      />
    );
  }

  if (mode === "report") {
    return (
      <Sheet visible={visible} title={`Signaler ${firstName}`} onClose={close} testID="report-sheet">
        <Text tone="muted">Ton signalement est confidentiel. {firstName} sera aussi bloqué·e pour te protéger.</Text>
        <OptionList options={reasons} selected={reason ? [reason] : []} onToggle={setReason} testIDPrefix="report-reason" />
        <TextField
          label="Détails (facultatif)"
          value={details}
          onChangeText={setDetails}
          multiline
          maxLength={1000}
          counter
          placeholder="Ce qui s'est passé…"
        />
        <Button
          label="Envoyer le signalement"
          variant="danger"
          disabled={!reason}
          loading={report.isPending}
          testID="report-submit"
          onPress={() => reason && run(() => report.mutateAsync({ userId, reason, details }), "Merci. Signalement envoyé et personne bloquée.")}
        />
      </Sheet>
    );
  }

  return (
    <Sheet visible={visible} title={firstName} onClose={close} testID="safety-sheet">
      <View style={{ gap: spacing.sm }}>
        {matchId ? (
          <Button label="Annuler le match" icon="heart-dislike-outline" variant="secondary" onPress={() => setMode("confirmUnmatch")} testID="safety-unmatch" />
        ) : null}
        <Button label="Bloquer" icon="ban-outline" variant="secondary" onPress={() => setMode("confirmBlock")} testID="safety-block" />
        <Button label="Signaler" icon="flag-outline" variant="danger" onPress={() => setMode("report")} testID="safety-report" />
        <Button label="Fermer" variant="ghost" onPress={close} />
      </View>
    </Sheet>
  );
}
