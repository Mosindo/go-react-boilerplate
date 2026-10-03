import React, { useCallback, useState } from "react";
import { Modal, Pressable, ScrollView, StyleSheet, View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { blockUser, errorMessage, queryKeys, reportUser, type ReportReason } from "../api/platform";
import { showToast } from "../shared/feedback";
import { Button, Input, Text, colors, radii, spacing } from "../shared/ui";
import { showDialog } from "../shared/dialog";

const REASONS: { value: ReportReason; label: string }[] = [
  { value: "fake", label: "Fake profile or scam" },
  { value: "harassment", label: "Harassment or abuse" },
  { value: "inappropriate", label: "Inappropriate content" },
  { value: "spam", label: "Spam" },
  { value: "underage", label: "Appears to be under 18" },
  { value: "other", label: "Something else" }
];

type Options = { userId: string; name: string; onBlocked?: () => void };

/**
 * Block / report entry point shared by the discovery card, the profile page and the chat.
 * `open()` shows the action menu; render `element` once somewhere in the screen.
 */
export function useSafetyMenu({ name, onBlocked, userId }: Options) {
  const client = useQueryClient();
  const [reporting, setReporting] = useState(false);
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");

  const block = useMutation({
    mutationFn: () => blockUser(userId),
    onSuccess: () => {
      showToast(`${name} is blocked.`, { tone: "success" });
      void client.invalidateQueries({ queryKey: queryKeys.conversations });
      void client.invalidateQueries({ queryKey: queryKeys.discover });
      void client.invalidateQueries({ queryKey: queryKeys.blocks });
      onBlocked?.();
    },
    onError: (e) => showToast(errorMessage(e), { tone: "error" })
  });

  const report = useMutation({
    mutationFn: () => reportUser(userId, reason ?? "other", details.trim()),
    onSuccess: () => {
      setReporting(false);
      setReason(null);
      setDetails("");
      showToast("Thanks. Your report was sent to our moderators.", { tone: "success" });
    },
    onError: (e) => showToast(errorMessage(e), { tone: "error" })
  });

  const confirmBlock = useCallback(() => {
    showDialog(`Block ${name}?`, "You will no longer see each other, and any conversation disappears for both of you. They are not told.", [
      { text: "Cancel", style: "cancel" },
      { text: "Block", style: "destructive", onPress: () => block.mutate() }
    ]);
  }, [block, name]);

  const open = useCallback(() => {
    showDialog(name, undefined, [
      { text: "Report", onPress: () => setReporting(true) },
      { text: "Block", style: "destructive", onPress: confirmBlock },
      { text: "Cancel", style: "cancel" }
    ]);
  }, [confirmBlock, name]);

  const element = (
    <Modal animationType="slide" onRequestClose={() => setReporting(false)} transparent visible={reporting}>
      <View style={styles.scrim}>
        <View style={styles.sheet}>
          <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
            <Text variant="heading" weight="bold">
              Report {name}
            </Text>
            <Text tone="muted">Tell us what is wrong. Reports are confidential.</Text>
            {REASONS.map((r) => (
              <Pressable
                accessibilityRole="radio"
                accessibilityState={{ selected: reason === r.value }}
                key={r.value}
                onPress={() => setReason(r.value)}
                style={[styles.reason, reason === r.value ? styles.reasonSelected : null]}
                testID={`report-reason-${r.value}`}
              >
                <Text weight={reason === r.value ? "bold" : "medium"}>{r.label}</Text>
              </Pressable>
            ))}
            <Input label="Details (optional)" maxLength={1000} multiline onChangeText={setDetails} value={details} />
            <Button disabled={!reason} label="Send report" loading={report.isPending} onPress={() => report.mutate()} testID="report-submit" />
            <Button label="Cancel" onPress={() => setReporting(false)} variant="ghost" />
          </ScrollView>
        </View>
      </View>
    </Modal>
  );

  return { open, element };
}

const styles = StyleSheet.create({
  scrim: { flex: 1, justifyContent: "flex-end", backgroundColor: colors.scrim },
  sheet: { maxHeight: "88%", backgroundColor: colors.background, borderTopLeftRadius: radii.xl, borderTopRightRadius: radii.xl },
  content: { padding: spacing.xl, gap: spacing.md },
  reason: { padding: spacing.md, borderRadius: radii.md, borderWidth: 1, borderColor: colors.border, backgroundColor: colors.backgroundElevated },
  reasonSelected: { borderColor: colors.primary, backgroundColor: colors.primarySoft }
});
