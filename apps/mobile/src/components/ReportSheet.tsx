import React, { useState } from "react";
import { Modal, Pressable, ScrollView, StyleSheet, View } from "react-native";
import { REPORT_REASONS, type ReportReason } from "../api/types";
import { REPORT_REASON_LABEL } from "../domain/labels";
import { Button, Chip, Input, Text } from "../shared/ui";
import { radius, spacing, useTheme } from "../theme";

export type ReportSheetProps = {
  visible: boolean;
  name: string;
  submitting: boolean;
  onClose: () => void;
  onSubmit: (reason: ReportReason, details: string) => void;
};

export function ReportSheet({ visible, name, submitting, onClose, onSubmit }: ReportSheetProps) {
  const theme = useTheme();
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");
  return (
    <Modal animationType="slide" onRequestClose={onClose} transparent visible={visible}>
      <View style={[styles.backdrop, { backgroundColor: theme.overlay }]}>
        <Pressable
          accessibilityLabel="Close report form"
          accessibilityRole="button"
          onPress={onClose}
          style={styles.dismiss}
        />
        <View style={[styles.sheet, { backgroundColor: theme.background }]}>
          <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
            <Text variant="title">Report {name}</Text>
            <Text tone="muted">Reports are private. Tell us what is wrong and our team will review it.</Text>
            <View style={styles.reasons}>
              {REPORT_REASONS.map((item) => (
                <Chip
                  key={item}
                  label={REPORT_REASON_LABEL[item]}
                  onPress={() => setReason(item)}
                  selected={reason === item}
                />
              ))}
            </View>
            <Input
              hint={`${details.length}/1000`}
              label="Details (optional)"
              maxLength={1000}
              multiline
              onChangeText={setDetails}
              style={styles.details}
              value={details}
            />
            <Button
              disabled={!reason}
              label="Send report"
              loading={submitting}
              onPress={() => reason && onSubmit(reason, details.trim())}
            />
            <Button label="Cancel" onPress={onClose} variant="ghost" />
          </ScrollView>
        </View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, justifyContent: "flex-end" },
  dismiss: { flex: 1 },
  sheet: { maxHeight: "85%", borderTopLeftRadius: radius.lg, borderTopRightRadius: radius.lg },
  content: { padding: spacing.lg, gap: spacing.lg },
  reasons: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  details: { minHeight: 90, textAlignVertical: "top" }
});
