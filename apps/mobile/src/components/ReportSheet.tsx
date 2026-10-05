import { useState } from "react";
import { Modal, Pressable, ScrollView, StyleSheet, View } from "react-native";
import { reportUser } from "../api/platform";
import type { ReportReason } from "../api/types";
import { errorMessage } from "../lib/errors";
import { REPORT_REASONS } from "../lib/labels";
import { showSuccess } from "../shared/feedback";
import { Button, Chip, Input, Notice, SwitchRow, Text } from "../shared/ui";
import { radii, spacing, useTheme } from "../theme";

/** Report a person (with an optional block). Reports are reviewed by moderators; the reported person is never told. */
export function ReportSheet({
  visible,
  userId,
  name,
  onClose,
  onDone
}: {
  visible: boolean;
  userId: string;
  name: string;
  onClose: () => void;
  onDone?: (blocked: boolean) => void;
}) {
  const { colors } = useTheme();
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");
  const [alsoBlock, setAlsoBlock] = useState(true);
  const [sending, setSending] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const submit = async () => {
    if (!reason) {
      setError("Choisissez un motif.");
      return;
    }
    setSending(true);
    setError(null);
    try {
      await reportUser(userId, reason, details.trim(), alsoBlock);
      showSuccess(alsoBlock ? `${name} a été signalé(e) et bloqué(e).` : `Merci, ${name} a été signalé(e).`);
      setReason(null);
      setDetails("");
      onClose();
      onDone?.(alsoBlock);
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setSending(false);
    }
  };

  return (
    <Modal visible={visible} transparent animationType="slide" onRequestClose={onClose} accessibilityViewIsModal>
      <View style={[styles.backdrop, { backgroundColor: colors.overlay }]}>
        <Pressable style={styles.dismiss} onPress={onClose} accessibilityLabel="Fermer" accessibilityRole="button" />
        <View style={[styles.sheet, { backgroundColor: colors.surface }]} testID="report-sheet">
          <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
            <Text variant="heading">Signaler {name}</Text>
            <Text tone="muted">Votre signalement est confidentiel. Notre équipe l'examinera.</Text>
            <View style={styles.reasons}>
              {REPORT_REASONS.map((r) => (
                <Chip key={r.value} label={r.label} selected={reason === r.value} onPress={() => setReason(r.value)} />
              ))}
            </View>
            <Input
              testID="report-details"
              label="Détails (facultatif)"
              value={details}
              onChangeText={setDetails}
              multiline
              maxLength={1000}
            />
            <SwitchRow
              label="Bloquer aussi cette personne"
              description="Vous ne vous verrez plus mutuellement."
              value={alsoBlock}
              onValueChange={setAlsoBlock}
            />
            {error ? <Notice kind="error" message={error} /> : null}
            <Button testID="report-submit" label="Envoyer le signalement" onPress={submit} loading={sending} />
            <Button label="Annuler" variant="ghost" onPress={onClose} />
          </ScrollView>
        </View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, justifyContent: "flex-end", alignItems: "center" },
  dismiss: { ...StyleSheet.absoluteFillObject },
  sheet: { width: "100%", maxWidth: 560, maxHeight: "88%", borderTopLeftRadius: radii.xl, borderTopRightRadius: radii.xl },
  content: { padding: spacing.xl, gap: spacing.lg },
  reasons: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }
});
