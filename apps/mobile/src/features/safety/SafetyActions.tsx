import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "../../api/queryKeys";
import { safetyApi } from "../../api/safety";
import type { ReportReason } from "../../api/types";
import { errorMessage } from "../../lib/errors";
import { radii, spacing, useTheme } from "../../theme";
import { Button, ConfirmSheet, Icon, InlineError, Sheet, Text, TextField, toast } from "../../ui";

export const reportReasons: { value: ReportReason; label: string }[] = [
  { value: "fake_profile", label: "Faux profil ou usurpation" },
  { value: "inappropriate_content", label: "Photos ou propos inappropriés" },
  { value: "harassment", label: "Harcèlement ou menaces" },
  { value: "spam", label: "Arnaque, spam ou publicité" },
  { value: "underage", label: "Personne mineure" },
  { value: "other", label: "Autre chose" }
];

function useInvalidateSocial() {
  const queryClient = useQueryClient();
  return () => {
    for (const key of [queryKeys.conversations, queryKeys.discovery, queryKeys.likes, queryKeys.badges, queryKeys.blocks]) {
      void queryClient.invalidateQueries({ queryKey: key });
    }
  };
}

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
  onDone: () => void;
}) {
  const { colors } = useTheme();
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");
  const invalidate = useInvalidateSocial();
  const report = useMutation({
    mutationFn: () => safetyApi.report(userId, reason as ReportReason, details.trim()),
    onSuccess: () => {
      invalidate();
      toast(`Merci. ${name} a été signalé·e et bloqué·e.`, "success");
      setReason(null);
      setDetails("");
      onDone();
    }
  });

  return (
    <Sheet visible={visible} onClose={onClose} title={`Signaler ${name}`}>
      <Text variant="caption" color="textMuted" style={styles.intro}>
        Votre signalement est confidentiel. La personne sera aussi bloquée : vous ne vous verrez plus.
      </Text>
      <View style={styles.reasons}>
        {reportReasons.map((r) => (
          <Pressable
            key={r.value}
            accessibilityRole="radio"
            accessibilityState={{ selected: reason === r.value }}
            onPress={() => setReason(r.value)}
            testID={`report-${r.value}`}
            style={[
              styles.reason,
              { borderColor: reason === r.value ? colors.primary : colors.border, backgroundColor: colors.surfaceAlt }
            ]}
          >
            <Text variant="label" style={styles.flex}>
              {r.label}
            </Text>
            {reason === r.value ? <Icon name="checkmark-circle" size={20} color="primary" /> : null}
          </Pressable>
        ))}
      </View>
      {reason ? <TextField label="Détails (facultatif)" value={details} onChangeText={setDetails} multiline maxLength={1000} /> : null}
      <View style={styles.actions}>
        <InlineError message={report.error ? errorMessage(report.error) : null} />
        <Button
          label="Envoyer le signalement"
          variant="danger"
          disabled={!reason}
          loading={report.isPending}
          onPress={() => report.mutate()}
          testID="report-submit"
        />
      </View>
    </Sheet>
  );
}

export function BlockConfirm({
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
  onDone: () => void;
}) {
  const invalidate = useInvalidateSocial();
  const block = useMutation({
    mutationFn: () => safetyApi.block(userId),
    onSuccess: () => {
      invalidate();
      toast(`${name} est bloqué·e.`, "success");
      onDone();
    },
    onError: (e) => toast(errorMessage(e), "error")
  });
  return (
    <ConfirmSheet
      visible={visible}
      title={`Bloquer ${name} ?`}
      message="Vous ne verrez plus son profil et votre match ainsi que vos messages seront supprimés. Cette personne ne sera pas prévenue."
      confirmLabel="Bloquer"
      destructive
      loading={block.isPending}
      onConfirm={() => block.mutate()}
      onClose={onClose}
    />
  );
}

const styles = StyleSheet.create({
  intro: { marginBottom: spacing.md },
  reasons: { gap: spacing.xs, marginBottom: spacing.md },
  reason: { flexDirection: "row", alignItems: "center", padding: spacing.md, borderRadius: radii.md, borderWidth: 1.5 },
  flex: { flex: 1 },
  actions: { gap: spacing.sm, marginTop: spacing.md }
});
