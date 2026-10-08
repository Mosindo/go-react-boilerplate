import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { describeError } from "../api/client";
import { keys } from "../api/keys";
import { reportUser } from "../api/platform";
import type { ReportReason } from "../api/types";
import type { MainStackParamList } from "../navigation/types";
import { showToast } from "../shared/feedback";
import { ScreenContainer } from "../shared/layout";
import { Button, Chip, FormField, Notice, Text, spacing } from "../shared/ui";
import { useTheme } from "../theme";

const REASONS: { value: ReportReason; label: string }[] = [
  { value: "fake_profile", label: "Faux profil" },
  { value: "harassment", label: "Harcèlement ou comportement déplacé" },
  { value: "inappropriate_content", label: "Contenu inapproprié" },
  { value: "spam", label: "Spam ou arnaque" },
  { value: "underage", label: "Personne mineure" },
  { value: "other", label: "Autre" }
];

export default function ReportScreen({ navigation, route }: NativeStackScreenProps<MainStackParamList, "Report">) {
  const { userId, firstName } = route.params;
  const { colors } = useTheme();
  const client = useQueryClient();
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");
  const [alsoBlock, setAlsoBlock] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const submit = useMutation({
    mutationFn: () => reportUser(userId, reason as ReportReason, details, alsoBlock),
    onSuccess: () => {
      for (const key of [keys.matches, keys.conversations, keys.unread, keys.notifications]) void client.invalidateQueries({ queryKey: key });
      showToast("Merci, votre signalement a été transmis à notre équipe.", "success");
      navigation.popToTop();
    },
    onError: (e) => setError(describeError(e))
  });

  return (
    <ScreenContainer
      footer={
        <View style={styles.footer}>
          <Button disabled={!reason} fullWidth loading={submit.isPending} onPress={() => submit.mutate()} size="lg">
            Envoyer le signalement
          </Button>
        </View>
      }
    >
      <Text variant="heading">Signaler {firstName}</Text>
      <Text tone="muted">Votre signalement est confidentiel : {firstName} ne saura pas que c'est vous.</Text>
      {error ? <Notice message={error} tone="error" /> : null}
      <View style={styles.reasons}>
        {REASONS.map((item) => (
          <Chip key={item.value} label={item.label} onPress={() => setReason(item.value)} selected={reason === item.value} />
        ))}
      </View>
      <FormField counter={1000} label="Précisions (facultatif)" maxLength={1000} multiline onChangeText={setDetails} value={details} />
      <Pressable accessibilityRole="checkbox" accessibilityState={{ checked: alsoBlock }} onPress={() => setAlsoBlock((v) => !v)} style={styles.check}>
        <Ionicons color={alsoBlock ? colors.primary : colors.textSubtle} name={alsoBlock ? "checkbox" : "square-outline"} size={24} />
        <Text style={styles.flex}>Bloquer aussi {firstName}</Text>
      </Pressable>
    </ScreenContainer>
  );
}

const styles = StyleSheet.create({
  footer: { width: "100%", maxWidth: 520 },
  reasons: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  check: { flexDirection: "row", alignItems: "center", gap: spacing.sm, minHeight: 44 },
  flex: { flex: 1 }
});
