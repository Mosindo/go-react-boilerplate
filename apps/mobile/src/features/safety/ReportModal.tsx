import React, { useState } from "react";
import { Modal, StyleSheet, Text, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { api } from "../../api";
import { errorMessage } from "../../api/client";
import type { ReportReason } from "../../api/types";
import { radius, spacing, useTheme } from "../../theme";
import { Button, Chip, ChipRow, ErrorText, TextField } from "../../ui/components";

const REASONS: { value: ReportReason; label: string }[] = [
  { value: "inappropriate", label: "Contenu inapproprié" },
  { value: "harassment", label: "Harcèlement" },
  { value: "fake", label: "Faux profil" },
  { value: "spam", label: "Spam" },
  { value: "underage", label: "Mineur" },
  { value: "other", label: "Autre" }
];

type Props = { userId: string | null; name: string; onClose: (sent: boolean) => void };

export function ReportModal({ userId, name, onClose }: Props) {
  const theme = useTheme();
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");
  const submit = useMutation({
    mutationFn: () => api.report(userId as string, reason as ReportReason, details.trim()),
    onSuccess: () => {
      setReason(null);
      setDetails("");
      onClose(true);
    }
  });

  return (
    <Modal visible={userId !== null} transparent animationType="slide" onRequestClose={() => onClose(false)}>
      <View style={[styles.backdrop, { backgroundColor: theme.overlay }]}>
        <View style={[styles.sheet, { backgroundColor: theme.surface }]}>
          <Text style={[styles.title, { color: theme.text }]}>{`Signaler ${name}`}</Text>
          <Text style={{ color: theme.textMuted }}>Votre signalement est confidentiel. Il sera examiné par la modération.</Text>
          <ChipRow>
            {REASONS.map((r) => (
              <Chip
                key={r.value}
                label={r.label}
                selected={reason === r.value}
                onPress={() => setReason(r.value)}
                testID={`reason-${r.value}`}
              />
            ))}
          </ChipRow>
          <TextField label="Détails (facultatif)" value={details} onChangeText={setDetails} multiline maxLength={1000} />
          {submit.isError ? <ErrorText message={errorMessage(submit.error)} /> : null}
          <Button
            title="Envoyer le signalement"
            disabled={!reason}
            loading={submit.isPending}
            onPress={() => submit.mutate()}
            testID="report-submit"
          />
          <Button title="Annuler" variant="ghost" onPress={() => onClose(false)} />
        </View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, justifyContent: "flex-end" },
  sheet: { borderTopLeftRadius: radius.lg, borderTopRightRadius: radius.lg, padding: spacing.xl, gap: spacing.md },
  title: { fontSize: 20, fontWeight: "800" }
});
