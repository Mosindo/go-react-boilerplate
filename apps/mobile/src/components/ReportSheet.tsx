import React, { useCallback, useRef, useState } from "react";
import { Pressable, ScrollView, StyleSheet, TextInput, View } from "react-native";
import type { ReportReason } from "../api/models";
import { reportUser } from "../api/safety";
import { useTheme } from "../shared/ui/theme";
import { BottomSheet } from "./BottomSheet";
import { errorMessage } from "./errors";
import { AppButton, Banner, MIN_TARGET, Txt } from "./kit";

export const REPORT_REASONS: readonly { value: ReportReason; label: string }[] = [
  { value: "spam", label: "Spam" },
  { value: "fake_profile", label: "Fake profile" },
  { value: "harassment", label: "Harassment" },
  { value: "inappropriate_content", label: "Inappropriate content" },
  { value: "underage", label: "Seems underage" },
  { value: "scam", label: "Scam or fraud" },
  { value: "other", label: "Something else" }
];

const MAX_DETAILS = 1000;

type Props = {
  visible: boolean;
  userId: string;
  firstName: string;
  onClose: () => void;
  /** Called after a successful report; `blocked` is true when "Also block" was checked. */
  onReported?: (result: { blocked: boolean }) => void;
};

export function ReportSheet({ visible, userId, firstName, onClose, onReported }: Props) {
  const sentRef = useRef<{ blocked: boolean } | null>(null);
  const handleSent = useCallback((result: { blocked: boolean }) => {
    sentRef.current = result;
  }, []);
  // Report side effects (e.g. leaving a blocked chat) wait until the confirmation is dismissed.
  const close = useCallback(() => {
    const result = sentRef.current;
    sentRef.current = null;
    onClose();
    if (result) onReported?.(result);
  }, [onClose, onReported]);
  return (
    <BottomSheet
      visible={visible}
      onClose={close}
      testID="report-sheet"
      accessibilityLabel={`Report ${firstName}`}
    >
      {/* Mounted only while visible, so the form state resets every time the sheet opens. */}
      <ReportForm userId={userId} firstName={firstName} onClose={close} onSent={handleSent} />
    </BottomSheet>
  );
}

function ReportForm({
  userId,
  firstName,
  onClose,
  onSent
}: Pick<Props, "userId" | "firstName" | "onClose"> & {
  onSent: (result: { blocked: boolean }) => void;
}) {
  const { colors, radii, spacing } = useTheme();
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");
  const [block, setBlock] = useState(false);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [sent, setSent] = useState<{ blocked: boolean } | null>(null);

  const submit = useCallback(async () => {
    if (!reason || submitting) return;
    setSubmitting(true);
    setError(null);
    try {
      const trimmed = details.trim();
      await reportUser({
        userId,
        reason,
        ...(trimmed ? { details: trimmed } : {}),
        ...(block ? { block: true } : {})
      });
      setSent({ blocked: block });
      onSent({ blocked: block });
    } catch (e) {
      setError(
        errorMessage(e, "We couldn't send your report. Check your connection and try again.")
      );
    } finally {
      setSubmitting(false);
    }
  }, [block, details, onSent, reason, submitting, userId]);

  return (
    <>
      {sent ? (
        <View
          style={[styles.padded, { padding: spacing.xl, gap: spacing.md }]}
          testID="report-sent"
        >
          <Txt variant="heading" accessibilityRole="header">
            Thanks for letting us know
          </Txt>
          <Txt tone="muted">
            {sent.blocked
              ? `We'll review your report. ${firstName} has been blocked and won't see you anymore.`
              : "We'll review your report. Your feedback keeps Alba kind and safe."}
          </Txt>
          <AppButton label="Done" onPress={onClose} testID="report-done" fullWidth />
        </View>
      ) : (
        <ScrollView
          keyboardShouldPersistTaps="handled"
          contentContainerStyle={{ padding: spacing.xl, gap: spacing.md }}
        >
          <Txt variant="heading" accessibilityRole="header">
            {`Report ${firstName}`}
          </Txt>
          <Txt tone="muted">Pick what fits best. Reports are confidential.</Txt>

          <View accessibilityRole="radiogroup" style={{ gap: spacing.xs }}>
            {REPORT_REASONS.map((item) => {
              const selected = reason === item.value;
              return (
                <Pressable
                  key={item.value}
                  accessibilityRole="radio"
                  accessibilityLabel={item.label}
                  accessibilityState={{ selected }}
                  onPress={() => setReason(item.value)}
                  testID={`report-reason-${item.value}`}
                  style={[
                    styles.row,
                    {
                      minHeight: MIN_TARGET,
                      borderRadius: radii.md,
                      paddingHorizontal: spacing.md,
                      gap: spacing.md,
                      backgroundColor: selected ? colors.primarySoft : colors.surface,
                      borderColor: selected ? colors.primary : colors.border
                    }
                  ]}
                >
                  <View
                    style={[
                      styles.radio,
                      {
                        borderColor: selected ? colors.primary : colors.borderStrong
                      }
                    ]}
                  >
                    {selected ? (
                      <View style={[styles.radioDot, { backgroundColor: colors.primary }]} />
                    ) : null}
                  </View>
                  <Txt weight={selected ? "semibold" : "regular"}>{item.label}</Txt>
                </Pressable>
              );
            })}
          </View>

          <TextInput
            value={details}
            onChangeText={setDetails}
            maxLength={MAX_DETAILS}
            multiline
            placeholder="Add details (optional)"
            placeholderTextColor={colors.textSubtle}
            accessibilityLabel="Report details, optional"
            testID="report-details"
            style={[
              styles.input,
              {
                color: colors.text,
                backgroundColor: colors.surface,
                borderColor: colors.border,
                borderRadius: radii.md,
                padding: spacing.md
              }
            ]}
          />
          <Txt variant="caption" tone="subtle" style={styles.right}>
            {`${details.length}/${MAX_DETAILS}`}
          </Txt>

          <Pressable
            accessibilityRole="checkbox"
            accessibilityLabel="Also block this person"
            accessibilityState={{ checked: block }}
            onPress={() => setBlock((v) => !v)}
            testID="report-block-checkbox"
            style={[styles.row, styles.plainRow, { minHeight: MIN_TARGET, gap: spacing.md }]}
          >
            <View
              style={[
                styles.checkbox,
                {
                  borderColor: block ? colors.primary : colors.borderStrong,
                  backgroundColor: block ? colors.primary : "transparent"
                }
              ]}
            >
              {block ? (
                <Txt variant="caption" weight="bold" style={{ color: colors.primaryForeground }}>
                  ✓
                </Txt>
              ) : null}
            </View>
            <Txt style={styles.flex}>Also block this person</Txt>
          </Pressable>

          {error ? <Banner tone="danger" message={error} testID="report-error" /> : null}

          <AppButton
            label="Send report"
            onPress={() => void submit()}
            disabled={!reason}
            loading={submitting}
            testID="report-submit"
            fullWidth
          />
          <AppButton label="Cancel" onPress={onClose} variant="ghost" fullWidth />
        </ScrollView>
      )}
    </>
  );
}

const styles = StyleSheet.create({
  padded: {},
  row: { flexDirection: "row", alignItems: "center", borderWidth: 1 },
  plainRow: { borderWidth: 0 },
  radio: {
    width: 22,
    height: 22,
    borderRadius: 11,
    borderWidth: 2,
    alignItems: "center",
    justifyContent: "center"
  },
  radioDot: { width: 10, height: 10, borderRadius: 5 },
  checkbox: {
    width: 24,
    height: 24,
    borderRadius: 6,
    borderWidth: 2,
    alignItems: "center",
    justifyContent: "center"
  },
  input: {
    minHeight: 96,
    maxHeight: 160,
    borderWidth: 1,
    textAlignVertical: "top",
    fontSize: 15
  },
  right: { textAlign: "right" },
  flex: { flex: 1 }
});
