import React, { useState } from "react";
import { ScrollView, StyleSheet, View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { reportUser } from "../api/platform";
import type { ReportReason } from "../api/types";
import { describeError } from "../lib/errors";
import type { RootStackParamList } from "../navigation/types";
import { showToast } from "../shared/feedback";
import { Badge, Button, Input, Notice, Text, spacing } from "../shared/ui";

type Props = NativeStackScreenProps<RootStackParamList, "Report">;

const REASONS: { value: ReportReason; label: string }[] = [
  { value: "fake_profile", label: "Faux profil" },
  { value: "inappropriate_content", label: "Contenu inapproprié" },
  { value: "harassment", label: "Harcèlement" },
  { value: "spam", label: "Spam ou arnaque" },
  { value: "underage", label: "Personne mineure" },
  { value: "other", label: "Autre" }
];

export default function ReportScreen({ navigation, route }: Props) {
  const { userId, name } = route.params;
  const queryClient = useQueryClient();
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");
  const [alsoBlock, setAlsoBlock] = useState(true);

  const submit = useMutation({
    mutationFn: () => {
      if (!reason) {
        throw new Error("reason required");
      }
      return reportUser({ userId, reason, details: details.trim(), alsoBlock });
    },
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: ["matches"] });
      showToast("Merci, votre signalement a été transmis.", "success");
      navigation.popToTop();
    }
  });

  return (
    <ScrollView contentContainerStyle={styles.content} keyboardShouldPersistTaps="handled" testID="report-screen">
      <Text tone="muted">Pourquoi signalez-vous {name} ? La personne ne sera pas prévenue.</Text>
      <View style={styles.chips}>
        {REASONS.map((option) => (
          <Badge key={option.value} label={option.label} onPress={() => setReason(option.value)} selected={reason === option.value} />
        ))}
      </View>
      <Input
        helperText="Facultatif"
        label="Détails"
        maxLength={1000}
        multiline
        onChangeText={setDetails}
        placeholder="Décrivez ce qui s'est passé…"
        value={details}
      />
      <Badge
        label={alsoBlock ? "✓ Bloquer aussi cette personne" : "Bloquer aussi cette personne"}
        onPress={() => setAlsoBlock((v) => !v)}
        selected={alsoBlock}
      />
      {submit.isError ? <Notice message={describeError(submit.error)} tone="danger" /> : null}
      <Button
        disabled={!reason}
        label="Envoyer le signalement"
        loading={submit.isPending}
        onPress={() => submit.mutate()}
        testID="report-submit"
      />
    </ScrollView>
  );
}

const styles = StyleSheet.create({
  content: { padding: spacing.xl, gap: spacing.lg },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }
});
