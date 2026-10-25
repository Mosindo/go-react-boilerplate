import React, { useState } from "react";
import { Modal, Pressable, StyleSheet, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { REPORT_REASONS, type ReportReason } from "../../api";
import { REPORT_REASON_LABELS } from "../../shared/labels";
import { Button, Chip, Text, TextField } from "../../shared/ui";
import { radius, spacing, useTheme } from "../../theme";

const MAX_DETAILS = 1000;

type Props = {
  visible: boolean;
  firstName: string;
  submitting: boolean;
  onClose: () => void;
  onSubmit: (reason: ReportReason, details: string) => void;
};

export function ReportModal({ visible, firstName, submitting, onClose, onSubmit }: Props) {
  const { colors } = useTheme();
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");

  return (
    <Modal animationType="slide" onRequestClose={onClose} transparent visible={visible}>
      <Pressable
        accessibilityLabel="Fermer"
        onPress={onClose}
        style={[styles.backdrop, { backgroundColor: colors.overlay }]}
      >
        <SafeAreaView edges={["bottom"]}>
          <Pressable accessible={false} style={[styles.sheet, { backgroundColor: colors.surface }]}>
            <Text variant="heading">{`Signaler ${firstName}`}</Text>
            <Text color="textMuted">Choisissez la raison. Votre signalement reste confidentiel.</Text>
            <View accessibilityRole="radiogroup" style={styles.reasons}>
              {REPORT_REASONS.map((item) => (
                <Chip
                  key={item}
                  label={REPORT_REASON_LABELS[item]}
                  onPress={() => setReason(item)}
                  selected={reason === item}
                  testID={`report-reason-${item}`}
                />
              ))}
            </View>
            <TextField
              label="Détails (facultatif)"
              maxLength={MAX_DETAILS}
              multiline
              onChangeText={setDetails}
              style={styles.details}
              value={details}
            />
            <Button
              disabled={!reason}
              label="Envoyer le signalement"
              loading={submitting}
              onPress={() => reason && onSubmit(reason, details.trim())}
              testID="report-submit"
            />
            <Button label="Annuler" onPress={onClose} variant="ghost" />
          </Pressable>
        </SafeAreaView>
      </Pressable>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, justifyContent: "flex-end" },
  sheet: { borderTopLeftRadius: radius.lg, borderTopRightRadius: radius.lg, padding: spacing.xl, gap: spacing.md },
  reasons: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  details: { minHeight: 64, textAlignVertical: "top" }
});
