import React, { useState } from "react";
import { Modal, Pressable, ScrollView, StyleSheet, View } from "react-native";
import { REPORT_REASONS, type ReportReason } from "../api/types";
import { Button, Chip, Input, Notice, Text, radii, spacing, useTheme } from "../shared/ui";

type ReportSheetProps = {
  visible: boolean;
  name: string;
  busy?: boolean;
  error?: string | null;
  onSubmit: (reason: ReportReason, details: string, block: boolean) => void;
  onClose: () => void;
};

export function ReportSheet({ visible, name, busy, error, onSubmit, onClose }: ReportSheetProps) {
  const { colors } = useTheme();
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");
  const [block, setBlock] = useState(true);

  return (
    <Modal animationType="slide" onRequestClose={onClose} transparent visible={visible}>
      <Pressable accessibilityLabel="Fermer" onPress={onClose} style={[styles.backdrop, { backgroundColor: colors.overlay }]}>
        <Pressable style={[styles.sheet, { backgroundColor: colors.surface }]}>
          <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
            <Text variant="heading">Signaler {name}</Text>
            <Text tone="muted">Votre signalement est confidentiel. Notre équipe l&apos;examinera.</Text>
            <View style={styles.reasons}>
              {REPORT_REASONS.map((r) => (
                <Chip key={r.value} label={r.label} onPress={() => setReason(r.value)} selected={reason === r.value} />
              ))}
            </View>
            <Input
              accessibilityLabel="Détails (facultatif)"
              maxLength={1000}
              multiline
              onChangeText={setDetails}
              placeholder="Détails (facultatif)"
              value={details}
            />
            <Chip
              label={block ? "✓ Je bloque aussi cette personne" : "Bloquer aussi cette personne"}
              onPress={() => setBlock(!block)}
              selected={block}
            />
            {error ? <Notice message={error} tone="danger" /> : null}
            <Button
              disabled={!reason}
              label="Envoyer le signalement"
              loading={busy}
              onPress={() => reason && onSubmit(reason, details.trim(), block)}
              variant="destructive"
            />
            <Button label="Annuler" onPress={onClose} variant="ghost" />
          </ScrollView>
        </Pressable>
      </Pressable>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, justifyContent: "flex-end" },
  sheet: { maxHeight: "90%", borderTopLeftRadius: radii.xl, borderTopRightRadius: radii.xl },
  content: { padding: spacing.lg, gap: spacing.md, paddingBottom: spacing.xxl },
  reasons: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }
});
