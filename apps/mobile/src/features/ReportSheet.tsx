import React, { useState } from "react";
import { Modal, ScrollView, StyleSheet, View } from "react-native";
import { useMutation } from "@tanstack/react-query";
import { safetyApi } from "../api/endpoints";
import type { ReportReason } from "../api/types";
import { describeError } from "../lib/validation";
import { radii, useTheme } from "../theme/theme";
import { Button, Chip, Input, Text, useToast } from "../ui";

const REASONS: { value: ReportReason; label: string }[] = [
  { value: "fake", label: "Faux profil" },
  { value: "harassment", label: "Harcèlement" },
  { value: "inappropriate", label: "Contenu inapproprié" },
  { value: "spam", label: "Spam / arnaque" },
  { value: "underage", label: "Personne mineure" },
  { value: "other", label: "Autre" }
];

type Props = { visible: boolean; userId: string; name: string; onClose: () => void; onDone: () => void };

export function ReportSheet({ visible, userId, name, onClose, onDone }: Props) {
  const { colors } = useTheme();
  const toast = useToast();
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");
  const [block, setBlock] = useState(true);

  const submit = useMutation({
    mutationFn: () => safetyApi.report(userId, reason as ReportReason, details.trim(), block),
    onSuccess: () => {
      toast.show(block ? `${name} a été signalé(e) et bloqué(e).` : `Merci, ${name} a été signalé(e).`);
      setReason(null);
      setDetails("");
      onDone();
    },
    onError: (e) => toast.show(describeError(e), "error")
  });

  return (
    <Modal visible={visible} transparent animationType="slide" onRequestClose={onClose}>
      <View style={[styles.backdrop, { backgroundColor: colors.overlay }]}>
        <View style={[styles.sheet, { backgroundColor: colors.background }]}>
          <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled">
            <Text variant="heading">Signaler {name}</Text>
            <Text muted>Votre signalement est confidentiel.</Text>
            <View style={styles.wrap}>
              {REASONS.map((r) => (
                <Chip key={r.value} label={r.label} selected={reason === r.value} onPress={() => setReason(r.value)} />
              ))}
            </View>
            <Input label="Détails (facultatif)" value={details} onChangeText={setDetails} multiline maxLength={1000} />
            <Chip
              label={block ? "✓ Bloquer aussi cette personne" : "Bloquer aussi cette personne"}
              selected={block}
              onPress={() => setBlock((b) => !b)}
            />
            <Button
              title="Envoyer le signalement"
              onPress={() => submit.mutate()}
              disabled={!reason}
              loading={submit.isPending}
              variant="danger"
            />
            <Button title="Annuler" variant="ghost" onPress={onClose} />
          </ScrollView>
        </View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, justifyContent: "flex-end" },
  sheet: { maxHeight: "90%", borderTopLeftRadius: radii.lg, borderTopRightRadius: radii.lg },
  content: { padding: 20, gap: 14 },
  wrap: { flexDirection: "row", flexWrap: "wrap", gap: 8 }
});
