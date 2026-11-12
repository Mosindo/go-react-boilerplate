import React, { useState } from "react";
import { Modal, Pressable, ScrollView, StyleSheet, View } from "react-native";
import { errorMessage } from "../api/client";
import { matchingApi, safetyApi } from "../api/endpoints";
import type { ReportReason } from "../api/types";
import { useInvalidateRelationships } from "../hooks/queries";
import { radii, spacing, useTheme } from "../theme";
import { Button, Chip, InlineError, ListRow, Text, TextField, showToast } from "../ui";
import { reportReasonLabels } from "../utils/format";

type Mode = "menu" | "report" | "block";

/** Bottom sheet grouping safety actions: unmatch, block, report. */
export function SafetySheet({
  visible,
  onClose,
  userId,
  firstName,
  matchId,
  onDone
}: {
  visible: boolean;
  onClose: () => void;
  userId: string;
  firstName: string;
  matchId?: string;
  onDone: () => void;
}) {
  const { colors } = useTheme();
  const invalidate = useInvalidateRelationships();
  const [mode, setMode] = useState<Mode>("menu");
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const close = () => {
    setMode("menu");
    setReason(null);
    setDetails("");
    setError(null);
    onClose();
  };

  const run = async (action: () => Promise<unknown>, message: string) => {
    setBusy(true);
    setError(null);
    try {
      await action();
      invalidate();
      showToast(message, "success");
      close();
      onDone();
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Modal visible={visible} transparent animationType="slide" onRequestClose={close}>
      <Pressable style={[styles.backdrop, { backgroundColor: colors.overlay }]} onPress={close} accessibilityLabel="Fermer" />
      <View style={[styles.sheet, { backgroundColor: colors.surface }]} accessibilityViewIsModal>
        <View style={[styles.grabber, { backgroundColor: colors.borderStrong }]} />
        <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
          {mode === "menu" ? (
            <>
              <Text variant="heading">{firstName}</Text>
              <View style={[styles.group, { borderColor: colors.border }]}>
                {matchId ? (
                  <ListRow
                    icon="heart-dislike-outline"
                    title="Annuler le match"
                    subtitle="La conversation sera supprimée pour vous deux."
                    onPress={() => void run(() => matchingApi.unmatch(matchId), "Match annulé")}
                  />
                ) : null}
                <ListRow icon="ban-outline" title="Bloquer" subtitle="Vous ne vous verrez plus, nulle part." onPress={() => setMode("block")} />
                <ListRow icon="flag-outline" title="Signaler" subtitle="Notre équipe examinera le profil." destructive onPress={() => setMode("report")} last />
              </View>
              <InlineError message={error} />
              <Button label="Fermer" variant="secondary" onPress={close} />
            </>
          ) : null}
          {mode === "block" ? (
            <>
              <Text variant="heading">Bloquer {firstName} ?</Text>
              <Text tone="muted">
                {firstName} ne pourra plus voir votre profil ni vous écrire, et votre éventuelle conversation sera supprimée. {firstName} ne sera pas prévenu·e.
              </Text>
              <InlineError message={error} />
              <Button
                label="Bloquer"
                variant="danger"
                loading={busy}
                onPress={() => void run(() => safetyApi.block(userId), `${firstName} est bloqué·e`)}
                testID="safety-block-confirm"
              />
              <Button label="Annuler" variant="ghost" onPress={() => setMode("menu")} />
            </>
          ) : null}
          {mode === "report" ? (
            <>
              <Text variant="heading">Signaler {firstName}</Text>
              <Text tone="muted">Le signalement est anonyme. {firstName} sera aussi bloqué·e automatiquement.</Text>
              <View style={styles.chips}>
                {(Object.keys(reportReasonLabels) as ReportReason[]).map((r) => (
                  <Chip key={r} label={reportReasonLabels[r]} selected={reason === r} onPress={() => setReason(r)} />
                ))}
              </View>
              <TextField label="Détails (facultatif)" value={details} onChangeText={setDetails} multiline maxLength={1000} counter />
              <InlineError message={error} />
              <Button
                label="Envoyer le signalement"
                variant="danger"
                disabled={!reason}
                loading={busy}
                onPress={() => reason && void run(() => safetyApi.report(userId, reason, details), "Merci, signalement envoyé")}
                testID="safety-report-confirm"
              />
              <Button label="Annuler" variant="ghost" onPress={() => setMode("menu")} />
            </>
          ) : null}
        </ScrollView>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1 },
  sheet: { borderTopLeftRadius: radii.xl, borderTopRightRadius: radii.xl, maxHeight: "85%", paddingBottom: spacing.xl },
  grabber: { width: 40, height: 4, borderRadius: 2, alignSelf: "center", marginTop: spacing.xs },
  content: { padding: spacing.lg, gap: spacing.md },
  group: { borderWidth: 1, borderRadius: radii.lg, overflow: "hidden" },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs }
});
