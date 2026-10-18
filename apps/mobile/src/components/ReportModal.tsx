import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { reportUser } from "../api/safety";
import { errorMessage } from "../api/client";
import type { ReportReason } from "../api/types";
import { showToast } from "../shared/feedback";
import { Input, Text, spacing } from "../shared/ui";
import { Chip, ChipRow } from "./Chip";
import { ConfirmModal } from "./ConfirmModal";

const REASONS: { value: ReportReason; label: string }[] = [
  { value: "fake", label: "Faux profil" },
  { value: "harassment", label: "Harcèlement" },
  { value: "inappropriate", label: "Contenu inapproprié" },
  { value: "spam", label: "Spam ou arnaque" },
  { value: "underage", label: "Mineur" },
  { value: "other", label: "Autre" }
];

type ReportModalProps = {
  userId: string;
  firstName: string;
  visible: boolean;
  onClose: () => void;
  onDone: () => void;
};

export function ReportModal({ userId, firstName, visible, onClose, onDone }: ReportModalProps) {
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");

  const report = useMutation({
    mutationFn: () => reportUser(userId, reason ?? "other", details.trim(), true),
    onSuccess: () => {
      showToast(`${firstName} a été signalé(e) et bloqué(e). Merci.`, "success");
      setReason(null);
      setDetails("");
      onDone();
    },
    onError: (error) => showToast(errorMessage(error), "error")
  });

  return (
    <ConfirmModal
      confirmLabel="Signaler et bloquer"
      destructive
      loading={report.isPending}
      message={`${firstName} ne pourra plus vous voir ni vous écrire. Notre équipe examinera le signalement.`}
      onCancel={onClose}
      onConfirm={() => (reason ? report.mutate() : showToast("Choisissez un motif.", "error"))}
      title={`Signaler ${firstName}`}
      visible={visible}
    >
      <View style={styles.body}>
        <ChipRow>
          {REASONS.map((r) => (
            <Chip key={r.value} label={r.label} onPress={() => setReason(r.value)} selected={reason === r.value} testID={`reason-${r.value}`} />
          ))}
        </ChipRow>
        <Input maxLength={1000} multiline onChangeText={setDetails} placeholder="Détails (facultatif)" testID="report-details" value={details} />
        <Text tone="muted" variant="caption">
          En cas de danger immédiat, contactez les services d'urgence.
        </Text>
      </View>
    </ConfirmModal>
  );
}

const styles = StyleSheet.create({ body: { gap: spacing.md } });
