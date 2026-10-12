import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useQueryClient } from "@tanstack/react-query";
import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { errorMessage } from "../api/client";
import { datingApi } from "../api/dating";
import type { ReportReason } from "../api/types";
import { keys } from "../hooks/queries";
import type { MainStackParamList } from "../navigation/types";
import { spacing } from "../theme";
import { Button, Chip, Input, Screen, Text, useToast } from "../ui";

const REASONS: { value: ReportReason; label: string }[] = [
  { value: "inappropriate", label: "Contenu inapproprié" },
  { value: "harassment", label: "Harcèlement" },
  { value: "fake", label: "Faux profil" },
  { value: "spam", label: "Spam ou arnaque" },
  { value: "underage", label: "Personne mineure" },
  { value: "other", label: "Autre" }
];

export default function ReportScreen({ navigation, route }: NativeStackScreenProps<MainStackParamList, "Report">) {
  const { userId, name } = route.params;
  const toast = useToast();
  const client = useQueryClient();
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");
  const [alsoBlock, setAlsoBlock] = useState(true);
  const [busy, setBusy] = useState(false);

  const submit = async () => {
    if (!reason) {
      toast.show("Choisissez un motif.", "error");
      return;
    }
    setBusy(true);
    try {
      await datingApi.report(userId, reason, details.trim(), alsoBlock);
      void client.invalidateQueries({ queryKey: keys.matches });
      void client.invalidateQueries({ queryKey: keys.discover });
      toast.show("Merci, votre signalement a été transmis.", "success");
      navigation.popToTop();
    } catch (e) {
      toast.show(errorMessage(e), "error");
    } finally {
      setBusy(false);
    }
  };

  return (
    <Screen scroll top={false}>
      <Text variant="title">Signaler {name}</Text>
      <Text tone="muted">Votre signalement est confidentiel. Notre équipe l'examinera.</Text>
      <View style={styles.chips}>
        {REASONS.map((r) => (
          <Chip key={r.value} label={r.label} selected={reason === r.value} onPress={() => setReason(r.value)} />
        ))}
      </View>
      <Input
        label="Détails (facultatif)"
        value={details}
        onChangeText={setDetails}
        multiline
        maxLength={1000}
        hint={`${details.length}/1000`}
      />
      <View style={styles.chips}>
        <Chip
          label={alsoBlock ? "✓ Bloquer aussi cette personne" : "Bloquer aussi cette personne"}
          selected={alsoBlock}
          onPress={() => setAlsoBlock((v) => !v)}
        />
      </View>
      <Button label="Envoyer le signalement" variant="danger" onPress={() => void submit()} loading={busy} />
    </Screen>
  );
}

const styles = StyleSheet.create({ chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm } });
