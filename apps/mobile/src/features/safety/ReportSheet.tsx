import React, { useState } from "react";
import { Modal, ScrollView, StyleSheet, View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { safetyApi } from "../../api/endpoints";
import { queryKeys } from "../../api/queryKeys";
import type { ReportReason } from "../../api/types";
import { useTheme } from "../../theme/ThemeProvider";
import { radii, spacing } from "../../theme/tokens";
import { Button, Text, TextField, useFeedback } from "../../ui";
import { ChoiceList } from "../profile/ChoiceList";

const reasons: { value: ReportReason; label: string }[] = [
  { value: "fake_profile", label: "Faux profil ou usurpation" },
  { value: "inappropriate_content", label: "Photos ou contenu inapproprié" },
  { value: "harassment", label: "Harcèlement ou propos offensants" },
  { value: "spam", label: "Spam, arnaque ou publicité" },
  { value: "underage", label: "Personne mineure" },
  { value: "other", label: "Autre chose" }
];

type Props = { userId: string; name: string; visible: boolean; onClose: () => void; onDone: () => void };

/** Reporting always blocks the person too, so the reporter is protected immediately. */
export function ReportSheet({ userId, name, visible, onClose, onDone }: Props) {
  const { colors } = useTheme();
  const { toast } = useFeedback();
  const client = useQueryClient();
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");

  const report = useMutation({
    mutationFn: () => safetyApi.report(userId, reason ?? "other", details),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: queryKeys.conversations });
      void client.invalidateQueries({ queryKey: queryKeys.blocks });
      toast(`Merci. ${name} a été signalé·e et bloqué·e.`, "success");
      onDone();
    }
  });

  return (
    <Modal visible={visible} animationType="slide" transparent onRequestClose={onClose}>
      <View style={[styles.backdrop, { backgroundColor: colors.overlay }]}>
        <View style={[styles.sheet, { backgroundColor: colors.surfaceRaised }]}>
          <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
            <Text variant="heading">Signaler {name}</Text>
            <Text tone="muted">
              Votre signalement est confidentiel. {name} sera aussi bloqué·e et ne pourra plus vous voir ni vous écrire.
            </Text>
            <ChoiceList options={reasons} selected={reason ? [reason] : []} onToggle={setReason} testIDPrefix="report-reason" />
            <TextField label="Détails (facultatif)" value={details} onChangeText={setDetails} multiline maxLength={1000} counter />
            {report.error ? <Text tone="danger">{errorMessage(report.error)}</Text> : null}
            <Button
              label="Envoyer le signalement"
              variant="danger"
              disabled={!reason}
              loading={report.isPending}
              onPress={() => report.mutate()}
              testID="report-submit"
            />
            <Button label="Annuler" variant="ghost" onPress={onClose} />
          </ScrollView>
        </View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, justifyContent: "flex-end" },
  sheet: {
    maxHeight: "90%",
    borderTopLeftRadius: radii.lg,
    borderTopRightRadius: radii.lg,
    width: "100%",
    maxWidth: 640,
    alignSelf: "center"
  },
  content: { padding: spacing.lg, gap: spacing.md }
});
