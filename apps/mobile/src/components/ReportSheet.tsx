import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { safetyApi } from "../api/endpoints";
import type { ReportReason } from "../api/types";
import { errorMessage, keys } from "../hooks/queries";
import { Button, Chip, Input, Sheet, SheetBody, Text, spacing, useToast } from "../ui";

const REASONS: { value: ReportReason; label: string }[] = [
  { value: "fake_profile", label: "Faux profil" },
  { value: "harassment", label: "Harcèlement" },
  { value: "inappropriate_content", label: "Contenu inapproprié" },
  { value: "spam", label: "Spam ou arnaque" },
  { value: "underage", label: "Personne mineure" },
  { value: "other", label: "Autre" }
];

type Props = { visible: boolean; userId: string; name: string; onClose: () => void; onDone?: () => void };

export function ReportSheet({ visible, userId, name, onClose, onDone }: Props) {
  const toast = useToast();
  const queryClient = useQueryClient();
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");
  const [alsoBlock, setAlsoBlock] = useState(true);
  const [busy, setBusy] = useState(false);

  const submit = async () => {
    if (!reason) return;
    setBusy(true);
    try {
      await safetyApi.report(userId, reason, details.trim(), alsoBlock);
      void queryClient.invalidateQueries({ queryKey: keys.conversations });
      void queryClient.invalidateQueries({ queryKey: keys.blocked });
      toast.show("Merci, votre signalement a été transmis.");
      setReason(null);
      setDetails("");
      onClose();
      onDone?.();
    } catch (error) {
      toast.show(errorMessage(error), "error");
    } finally {
      setBusy(false);
    }
  };

  return (
    <Sheet onClose={onClose} title={`Signaler ${name}`} visible={visible}>
      <SheetBody>
        <Text tone="muted">Quel est le problème ? Votre signalement reste confidentiel.</Text>
        <View style={styles.chips}>
          {REASONS.map((r) => (
            <Chip
              key={r.value}
              label={r.label}
              onPress={() => setReason(r.value)}
              selected={reason === r.value}
              testID={`report-${r.value}`}
            />
          ))}
        </View>
        <Input label="Détails (facultatif)" maxLength={1000} multiline onChangeText={setDetails} value={details} />
        <Chip
          label={alsoBlock ? "✓ Bloquer aussi cette personne" : "Bloquer aussi cette personne"}
          onPress={() => setAlsoBlock(!alsoBlock)}
          selected={alsoBlock}
        />
        <Button
          disabled={!reason}
          label="Envoyer le signalement"
          loading={busy}
          onPress={submit}
          testID="report-submit"
        />
      </SheetBody>
    </Sheet>
  );
}

const styles = StyleSheet.create({
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }
});
