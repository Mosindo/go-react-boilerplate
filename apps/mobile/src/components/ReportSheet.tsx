import React, { useState } from "react";
import { KeyboardAvoidingView, Modal, Platform, Pressable, ScrollView, StyleSheet, View } from "react-native";
import type { ReportReason } from "../api/types";
import { Button, Input, Text, colors, radii, spacing } from "../shared/ui";

const REASONS: readonly { value: ReportReason; label: string }[] = [
  { value: "fake_profile", label: "Faux profil" },
  { value: "harassment", label: "Harcèlement ou comportement abusif" },
  { value: "inappropriate_content", label: "Contenu inapproprié" },
  { value: "spam", label: "Spam ou arnaque" },
  { value: "underage", label: "Personne mineure" },
  { value: "other", label: "Autre" }
];

type ReportSheetProps = {
  visible: boolean;
  name: string;
  submitting?: boolean;
  onClose: () => void;
  onSubmit: (reason: ReportReason, details: string) => void;
};

export function ReportSheet({ visible, name, submitting = false, onClose, onSubmit }: ReportSheetProps) {
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");

  const close = () => {
    setReason(null);
    setDetails("");
    onClose();
  };

  return (
    <Modal animationType="slide" onRequestClose={close} transparent visible={visible}>
      <KeyboardAvoidingView behavior={Platform.OS === "ios" ? "padding" : undefined} style={styles.backdrop}>
        <View style={styles.sheet}>
          <Text variant="heading" weight="bold">
            Signaler {name}
          </Text>
          <Text tone="muted">Votre signalement est confidentiel. Choisissez la raison qui correspond le mieux.</Text>
          <ScrollView contentContainerStyle={styles.reasons} keyboardShouldPersistTaps="handled">
            {REASONS.map((item) => (
              <Pressable
                accessibilityRole="radio"
                accessibilityState={{ selected: reason === item.value }}
                key={item.value}
                onPress={() => setReason(item.value)}
                style={[styles.reason, reason === item.value ? styles.reasonSelected : null]}
                testID={`report-${item.value}`}
              >
                <Text weight={reason === item.value ? "bold" : "regular"}>{item.label}</Text>
              </Pressable>
            ))}
            <Input
              label="Précisions (facultatif)"
              maxLength={1000}
              multiline
              onChangeText={setDetails}
              placeholder="Ce qui s'est passé…"
              value={details}
            />
          </ScrollView>
          <View style={styles.buttons}>
            <Button label="Annuler" onPress={close} style={styles.flex} variant="outline" />
            <Button
              disabled={!reason}
              label="Envoyer"
              loading={submitting}
              onPress={() => reason && onSubmit(reason, details.trim())}
              style={styles.flex}
              testID="report-submit"
              variant="destructive"
            />
          </View>
        </View>
      </KeyboardAvoidingView>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, backgroundColor: colors.overlay, justifyContent: "flex-end" },
  sheet: {
    backgroundColor: colors.backgroundElevated,
    borderTopLeftRadius: radii.xl,
    borderTopRightRadius: radii.xl,
    padding: spacing.xl,
    gap: spacing.md,
    maxHeight: "90%"
  },
  reasons: { gap: spacing.sm, paddingVertical: spacing.sm },
  reason: { padding: spacing.md, borderRadius: radii.md, borderWidth: 1, borderColor: colors.border },
  reasonSelected: { borderColor: colors.primary, backgroundColor: colors.primarySoft },
  buttons: { flexDirection: "row", gap: spacing.md },
  flex: { flex: 1 }
});
