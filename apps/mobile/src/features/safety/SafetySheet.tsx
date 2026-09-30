import React, { useState } from "react";
import { Modal, Pressable, ScrollView, StyleSheet, View } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { Button, Chip, Text, TextField } from "../../design/components";
import { useTheme } from "../../design/ThemeProvider";
import { errorMessage } from "../../lib/api/client";
import { safetyApi } from "../../lib/api/endpoints";
import type { ReportReason } from "../../lib/api/types";
import { reportReasonLabels } from "../../lib/format";
import { queryKeys } from "../../lib/queryClient";
import { showToast } from "../../lib/toast";

type Props = {
  visible: boolean;
  userId: string;
  name: string;
  onClose: () => void;
  /** Called after the person was blocked (reporting also blocks). */
  onBlocked: () => void;
};

/** Report and/or block a member. Reporting always blocks to protect the reporter. */
export function SafetySheet({ visible, userId, name, onClose, onBlocked }: Props) {
  const { colors, radii } = useTheme();
  const client = useQueryClient();
  const [mode, setMode] = useState<"menu" | "report">("menu");
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");
  const [busy, setBusy] = useState(false);

  const close = () => {
    setMode("menu");
    setReason(null);
    setDetails("");
    onClose();
  };

  const finish = (message: string) => {
    void client.invalidateQueries({ queryKey: queryKeys.conversations });
    void client.invalidateQueries({ queryKey: queryKeys.blocked });
    showToast(message, "success");
    close();
    onBlocked();
  };

  const block = async () => {
    setBusy(true);
    try {
      await safetyApi.block(userId);
      finish(`${name} est bloqué·e. Vous ne vous verrez plus.`);
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
      await safetyApi.report(userId, reason, details);
      finish("Merci. Votre signalement a été transmis et la personne est bloquée.");
    } catch (e) {
      showToast(errorMessage(e), "error");
    } finally {
      setBusy(false);
    }
  };

  return (
    <Modal animationType="slide" onRequestClose={close} transparent visible={visible}>
      <Pressable onPress={close} style={[styles.backdrop, { backgroundColor: colors.overlay }]}>
        <Pressable style={[styles.sheet, { backgroundColor: colors.surfaceRaised, borderTopLeftRadius: radii.xl, borderTopRightRadius: radii.xl }]}>
          <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
            {mode === "menu" ? (
              <>
                <Text variant="heading">Sécurité</Text>
                <Text tone="muted">Bloquer {name} supprime votre match et votre conversation. {name} n’en sera pas notifié·e.</Text>
                <Button icon="flag-outline" label={`Signaler ${name}`} onPress={() => setMode("report")} testID="safety-report" variant="danger" />
                <Button icon="ban-outline" label={`Bloquer ${name}`} loading={busy} onPress={block} testID="safety-block" variant="secondary" />
                <Button label="Annuler" onPress={close} variant="ghost" />
              </>
            ) : (
              <>
                <Text variant="heading">Pourquoi signalez-vous {name} ?</Text>
                <View style={styles.reasons}>
                  {(Object.keys(reportReasonLabels) as ReportReason[]).map((r) => (
                    <Chip key={r} label={reportReasonLabels[r]} onPress={() => setReason(r)} selected={reason === r} testID={`report-${r}`} />
                  ))}
                </View>
                <TextField label="Détails (facultatif)" maxLength={1000} multiline onChangeText={setDetails} value={details} />
                <Text tone="muted" variant="caption">
                  Votre signalement est confidentiel. La personne sera aussi bloquée.
                </Text>
                <Button disabled={!reason} label="Envoyer le signalement" loading={busy} onPress={report} testID="report-submit" variant="danger" />
                <Button label="Retour" onPress={() => setMode("menu")} variant="ghost" />
              </>
            )}
          </ScrollView>
        </Pressable>
      </Pressable>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, justifyContent: "flex-end" },
  sheet: { maxHeight: "85%", width: "100%", maxWidth: 640, alignSelf: "center" },
  content: { padding: 24, gap: 14 },
  reasons: { flexDirection: "row", flexWrap: "wrap", gap: 8 }
});
