import React, { useState } from "react";
import { Modal, StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { reportUser } from "../api/dating";
import type { ReportReason } from "../api/types";
import { Button, Input, Notice, Text, colors, radii, spacing } from "../shared/ui";
import { Chip } from "./Chip";

const REASONS: Array<{ value: ReportReason; label: string }> = [
  { value: "fake", label: "Faux profil" },
  { value: "inappropriate", label: "Contenu inapproprié" },
  { value: "harassment", label: "Harcèlement" },
  { value: "spam", label: "Spam" },
  { value: "underage", label: "Mineur" },
  { value: "other", label: "Autre" }
];

type ReportSheetProps = {
  userId: string | null;
  name: string;
  onClose: (reported: boolean) => void;
};

export function ReportSheet({ userId, name, onClose }: ReportSheetProps) {
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");
  const [alsoBlock, setAlsoBlock] = useState(true);

  const mutation = useMutation({
    mutationFn: () => {
      if (!userId || !reason) throw new Error("missing");
      return reportUser(userId, reason, details.trim(), alsoBlock);
    },
    onSuccess: () => {
      setReason(null);
      setDetails("");
      onClose(true);
    }
  });

  return (
    <Modal animationType="slide" onRequestClose={() => onClose(false)} transparent visible={userId !== null}>
      <View style={styles.backdrop}>
        <View style={styles.sheet}>
          <Text variant="heading" weight="bold">
            Signaler {name}
          </Text>
          <Text tone="muted">Votre signalement est confidentiel. Dites-nous ce qui pose problème.</Text>
          <View style={styles.row}>
            {REASONS.map((r) => (
              <Chip key={r.value} label={r.label} onPress={() => setReason(r.value)} selected={reason === r.value} />
            ))}
          </View>
          <Input label="Détails (facultatif)" maxLength={1000} multiline onChangeText={setDetails} value={details} />
          <View style={styles.row}>
            <Chip
              label={alsoBlock ? "Bloquer aussi : oui" : "Bloquer aussi : non"}
              onPress={() => setAlsoBlock((v) => !v)}
              selected={alsoBlock}
            />
          </View>
          {mutation.isError ? <Notice title="Le signalement n'a pas pu être envoyé. Réessayez." tone="danger" /> : null}
          <Button
            disabled={!reason}
            fullWidth
            label="Envoyer le signalement"
            loading={mutation.isPending}
            onPress={() => mutation.mutate()}
            variant="destructive"
          />
          <Button fullWidth label="Annuler" onPress={() => onClose(false)} variant="ghost" />
        </View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, backgroundColor: colors.overlay, justifyContent: "flex-end" },
  sheet: {
    backgroundColor: colors.background,
    borderTopLeftRadius: radii.xl,
    borderTopRightRadius: radii.xl,
    padding: spacing.xl,
    gap: spacing.md
  },
  row: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }
});
