import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { safetyApi } from "../../api/endpoints";
import type { ReportReason } from "../../api/types";
import { keys } from "../../state/queryClient";
import { radii, spacing } from "../../theme/palette";
import { useTheme } from "../../theme/ThemeProvider";
import { Button, Icon, InlineError, Sheet, Text, TextField, useToast } from "../../ui";
import { reportReasonLabels, reportReasons } from "../../utils/labels";

type Props = {
  visible: boolean;
  userId: string;
  firstName: string;
  onClose: () => void;
  onDone: () => void;
};

/** Report flow. Reporting also blocks the member for the reporter. */
export function ReportSheet({ visible, userId, firstName, onClose, onDone }: Props) {
  const { colors } = useTheme();
  const toast = useToast();
  const qc = useQueryClient();
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");
  const report = useMutation({
    mutationFn: () => safetyApi.report(userId, reason ?? "other", details.trim()),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: keys.conversations });
      void qc.invalidateQueries({ queryKey: keys.blocks });
      toast({ title: "Merci pour votre signalement", body: `${firstName} est bloqué·e et ne pourra plus vous contacter.`, tone: "success" });
      setReason(null);
      setDetails("");
      onDone();
    }
  });
  return (
    <Sheet onClose={onClose} title={`Signaler ${firstName}`} visible={visible}>
      <Text tone="muted">Votre signalement est confidentiel. La personne ne saura pas qu’il vient de vous.</Text>
      <View accessibilityRole="radiogroup" style={styles.list}>
        {reportReasons.map((r) => {
          const selected = reason === r;
          return (
            <Pressable
              accessibilityRole="radio"
              accessibilityState={{ selected }}
              key={r}
              onPress={() => setReason(r)}
              style={[styles.option, { borderColor: selected ? colors.primary : colors.border, backgroundColor: selected ? colors.primarySoft : colors.surface }]}
            >
              <Icon color={selected ? colors.primary : colors.textSubtle} name={selected ? "radio-button-on" : "radio-button-off"} size={20} />
              <Text style={styles.flex}>{reportReasonLabels[r]}</Text>
            </Pressable>
          );
        })}
      </View>
      <TextField
        hint={`${details.length}/1000`}
        label="Détails (facultatif)"
        maxLength={1000}
        multiline
        onChangeText={setDetails}
        placeholder="Ce qui s'est passé…"
        value={details}
      />
      <InlineError message={report.isError ? errorMessage(report.error) : null} />
      <Button disabled={!reason} label="Envoyer le signalement" loading={report.isPending} onPress={() => report.mutate()} variant="danger" />
    </Sheet>
  );
}

const styles = StyleSheet.create({
  list: { gap: spacing.xs },
  option: { flexDirection: "row", alignItems: "center", gap: spacing.sm, padding: spacing.sm, borderRadius: radii.md, borderWidth: 1 },
  flex: { flex: 1 }
});
