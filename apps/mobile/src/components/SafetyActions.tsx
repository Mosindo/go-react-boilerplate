import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { errorMessage } from "../api/client";
import { blockUser, reportUser } from "../api/safety";
import type { ReportReason } from "../api/types";
import { queryClient, queryKeys } from "../hooks/queryClient";
import { spacing } from "../theme";
import { Button, Chip, ConfirmDialog, Dialog, Text, TextField, showToast } from "../ui";

const reasons: { value: ReportReason; label: string }[] = [
  { value: "fake_profile", label: "Faux profil" },
  { value: "inappropriate_content", label: "Contenu inapproprié" },
  { value: "harassment", label: "Harcèlement" },
  { value: "spam", label: "Spam ou arnaque" },
  { value: "underage", label: "Personne mineure" },
  { value: "other", label: "Autre" }
];

function refreshAfterSafetyAction() {
  for (const key of [queryKeys.matches, queryKeys.conversations, queryKeys.likes, queryKeys.blocked, queryKeys.unreadConversations]) {
    void queryClient.invalidateQueries({ queryKey: key });
  }
}

type Props = {
  userId: string;
  firstName: string;
  mode: "block" | "report" | null;
  onClose: () => void;
  onDone: () => void;
};

/** Block and report dialogs. Reporting always blocks too, so the reporter is never exposed again. */
export function SafetyDialogs({ userId, firstName, mode, onClose, onDone }: Props) {
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");
  const [busy, setBusy] = useState(false);

  const block = async () => {
    setBusy(true);
    try {
      await blockUser(userId);
      refreshAfterSafetyAction();
      showToast(`${firstName} a été bloqué·e.`, "success");
      onDone();
    } catch (e) {
      showToast(errorMessage(e), "error");
    } finally {
      setBusy(false);
    }
  };

  const report = async () => {
    if (!reason) return;
    setBusy(true);
    try {
      await reportUser(userId, reason, details);
      refreshAfterSafetyAction();
      showToast("Merci. Le signalement a été transmis et ce profil est bloqué.", "success");
      setReason(null);
      setDetails("");
      onDone();
    } catch (e) {
      showToast(errorMessage(e), "error");
    } finally {
      setBusy(false);
    }
  };

  return (
    <>
      <ConfirmDialog
        confirmLabel="Bloquer"
        destructive
        loading={busy}
        message={`${firstName} ne pourra plus voir votre profil ni vous écrire. Votre match et votre conversation seront supprimés.`}
        onCancel={onClose}
        onConfirm={block}
        title={`Bloquer ${firstName} ?`}
        visible={mode === "block"}
      />
      <Dialog
        message="Votre signalement est confidentiel. Cette personne sera aussi bloquée."
        onClose={onClose}
        testID="report-dialog"
        title={`Signaler ${firstName}`}
        visible={mode === "report"}
      >
        <View style={styles.reasons}>
          {reasons.map((r) => (
            <Chip
              key={r.value}
              label={r.label}
              onPress={() => setReason(r.value)}
              selected={reason === r.value}
              testID={`report-reason-${r.value}`}
            />
          ))}
        </View>
        <TextField label="Détails (facultatif)" maxLength={1000} multiline onChangeText={setDetails} value={details} />
        {reason === "underage" ? (
          <Text tone="muted" variant="caption">
            Merci : les profils de personnes mineures sont examinés en priorité.
          </Text>
        ) : null}
        <Button disabled={!reason} label="Envoyer le signalement" loading={busy} onPress={report} testID="report-submit" variant="danger" />
        <Button label="Annuler" onPress={onClose} variant="ghost" />
      </Dialog>
    </>
  );
}

const styles = StyleSheet.create({
  reasons: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs }
});
