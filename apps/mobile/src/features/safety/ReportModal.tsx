import React, { useState } from "react";
import { Modal, Pressable, ScrollView, StyleSheet, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { getErrorMessage } from "../../api/client";
import { REPORT_REASONS, type ReportReason } from "../../api/types";
import { useReportUser } from "../../hooks/useSafety";
import { REPORT_DETAILS_MAX } from "../../lib/validation";
import { MIN_TOUCH, radii, spacing, useColors } from "../../theme";
import { Button, Icon, Input, SwitchRow, Text, toast } from "../../ui";
import { REPORT_REASON_LABELS } from "../profile/labels";

type Props = {
  visible: boolean;
  userId: string;
  displayName: string;
  onClose: () => void;
  onDone?: (blocked: boolean) => void;
};

export function ReportModal({ visible, userId, displayName, onClose, onDone }: Props) {
  const colors = useColors();
  const report = useReportUser();
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");
  const [block, setBlock] = useState(true);
  const [error, setError] = useState<string | null>(null);

  function close() {
    setReason(null);
    setDetails("");
    setError(null);
    onClose();
  }

  async function submit() {
    if (!reason) {
      setError("Choose a reason so we can look into it.");
      return;
    }
    try {
      await report.mutateAsync({ userId, reason, details: details.trim() || undefined, block });
      toast.success(block ? `Thanks. ${displayName} was reported and blocked.` : "Thanks. Your report was sent.");
      close();
      onDone?.(block);
    } catch (e) {
      setError(getErrorMessage(e));
    }
  }

  return (
    <Modal animationType="slide" onRequestClose={close} visible={visible}>
      <SafeAreaView style={[styles.screen, { backgroundColor: colors.background }]}>
        <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
          <Text variant="heading">Report {displayName}</Text>
          <Text tone="muted">Reports are reviewed by people. The other person is never told who reported them.</Text>
          <View accessibilityRole="radiogroup" style={styles.reasons}>
            {REPORT_REASONS.map((value) => {
              const selected = reason === value;
              return (
                <Pressable
                  key={value}
                  accessibilityLabel={REPORT_REASON_LABELS[value]}
                  accessibilityRole="radio"
                  accessibilityState={{ selected }}
                  onPress={() => {
                    setReason(value);
                    setError(null);
                  }}
                  style={[
                    styles.reason,
                    {
                      borderColor: selected ? colors.primary : colors.border,
                      backgroundColor: selected ? colors.primarySoft : colors.surface
                    }
                  ]}
                >
                  <Icon
                    color={selected ? colors.primary : colors.textMuted}
                    name={selected ? "radio-button-on" : "radio-button-off"}
                    size={22}
                  />
                  <Text weight="medium">{REPORT_REASON_LABELS[value]}</Text>
                </Pressable>
              );
            })}
          </View>
          <Input
            hint={`${details.length}/${REPORT_DETAILS_MAX}`}
            label="Details (optional)"
            maxLength={REPORT_DETAILS_MAX}
            multiline
            onChangeText={setDetails}
            value={details}
          />
          <SwitchRow
            description="They won't see you or be able to contact you."
            label="Also block this person"
            onValueChange={setBlock}
            value={block}
          />
          {error ? (
            <Text accessibilityRole="alert" tone="danger">
              {error}
            </Text>
          ) : null}
          <Button label="Send report" loading={report.isPending} onPress={submit} />
          <Button disabled={report.isPending} label="Cancel" onPress={close} variant="ghost" />
        </ScrollView>
      </SafeAreaView>
    </Modal>
  );
}

const styles = StyleSheet.create({
  screen: { flex: 1 },
  content: { padding: spacing.md, gap: spacing.md, maxWidth: 560, width: "100%", alignSelf: "center" },
  reasons: { gap: spacing.xs },
  reason: {
    minHeight: MIN_TOUCH + 4,
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.sm,
    paddingHorizontal: spacing.md,
    borderRadius: radii.md,
    borderWidth: 1
  }
});
