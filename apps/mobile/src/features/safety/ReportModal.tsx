import React, { useEffect, useState } from "react";
import {
  KeyboardAvoidingView,
  Modal,
  Platform,
  Pressable,
  ScrollView,
  StyleSheet,
  TextInput,
  View
} from "react-native";
import { Button, Text, radii, spacing } from "../../shared/ui";
import { usePalette } from "../ui/palette";
import type { ReportReason } from "../types";
import { MAX_REPORT_DETAILS, REPORT_REASONS } from "./reasons";

type Props = {
  visible: boolean;
  name: string;
  busy: boolean;
  onClose: () => void;
  onSubmit: (reason: ReportReason, details: string) => void;
};

export function ReportModal({ visible, name, busy, onClose, onSubmit }: Props) {
  const p = usePalette();
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");

  useEffect(() => {
    if (!visible) {
      setReason(null);
      setDetails("");
    }
  }, [visible]);

  return (
    <Modal animationType="slide" onRequestClose={onClose} transparent visible={visible}>
      <KeyboardAvoidingView
        behavior={Platform.OS === "ios" ? "padding" : undefined}
        style={[styles.backdrop, { backgroundColor: p.scrim }]}
      >
        <View style={[styles.sheet, { backgroundColor: p.surface }]} testID="report-modal">
          <Text variant="heading" weight="bold">
            Report {name}
          </Text>
          <Text tone="muted">Tell us what's wrong. Reports are confidential.</Text>
          <ScrollView keyboardShouldPersistTaps="handled" style={styles.scroll}>
            {REPORT_REASONS.map((item) => {
              const selected = reason === item.value;
              return (
                <Pressable
                  accessibilityLabel={item.label}
                  accessibilityRole="radio"
                  accessibilityState={{ selected }}
                  key={item.value}
                  onPress={() => setReason(item.value)}
                  style={[
                    styles.option,
                    {
                      borderColor: selected ? p.primary : p.border,
                      backgroundColor: selected ? p.primarySoft : p.surface
                    }
                  ]}
                  testID={`report-reason-${item.value}`}
                >
                  <Text weight={selected ? "bold" : "regular"}>{item.label}</Text>
                </Pressable>
              );
            })}
            <TextInput
              accessibilityLabel="Additional details (optional)"
              maxLength={MAX_REPORT_DETAILS}
              multiline
              onChangeText={setDetails}
              placeholder="Add details (optional)"
              placeholderTextColor={p.textMuted}
              style={[styles.input, { borderColor: p.border, color: p.text }]}
              testID="report-details-input"
              value={details}
            />
            <Text tone="muted" variant="caption">
              {details.length}/{MAX_REPORT_DETAILS}
            </Text>
          </ScrollView>
          <View style={styles.buttons}>
            <Button label="Cancel" onPress={onClose} variant="ghost" />
            <Button
              disabled={reason === null}
              label="Send report"
              loading={busy}
              onPress={() => reason && onSubmit(reason, details)}
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
  backdrop: { flex: 1, justifyContent: "flex-end" },
  sheet: {
    borderTopLeftRadius: radii.xl,
    borderTopRightRadius: radii.xl,
    padding: spacing.xl,
    gap: spacing.md,
    maxHeight: "88%"
  },
  scroll: { flexGrow: 0 },
  option: { borderWidth: 1, borderRadius: radii.md, padding: spacing.md, marginBottom: spacing.sm },
  input: {
    borderWidth: 1,
    borderRadius: radii.md,
    padding: spacing.md,
    minHeight: 90,
    textAlignVertical: "top",
    marginTop: spacing.xs
  },
  buttons: { flexDirection: "row", justifyContent: "flex-end", gap: spacing.sm }
});
