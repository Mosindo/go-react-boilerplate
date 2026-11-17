import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { safetyApi } from "../../api/endpoints";
import { queryKeys } from "../../api/queryKeys";
import type { ReportReason } from "../../api/types";
import type { RootScreenProps } from "../../navigation/types";
import { colors, radii, spacing } from "../../theme";
import { Button, Screen, Text, TextField, showToast } from "../../ui";

const REASONS: { value: ReportReason; label: string }[] = [
  { value: "fake_profile", label: "Faux profil ou usurpation" },
  { value: "inappropriate_content", label: "Photos ou propos inappropriés" },
  { value: "harassment", label: "Harcèlement ou menaces" },
  { value: "scam", label: "Arnaque, demande d'argent" },
  { value: "underage", label: "Personne mineure" },
  { value: "other", label: "Autre chose" }
];

export function ReportScreen({ navigation, route }: RootScreenProps<"Report">) {
  const queryClient = useQueryClient();
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");
  const report = useMutation({
    mutationFn: () => safetyApi.report(route.params.userId, reason as ReportReason, details.trim()),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      void queryClient.invalidateQueries({ queryKey: queryKeys.discover });
      void queryClient.invalidateQueries({ queryKey: queryKeys.blocks });
      showToast("Merci. Le signalement est enregistré et ce membre est bloqué.", "success");
      navigation.popToTop();
    }
  });

  return (
    <Screen
      scroll
      edges={["left", "right", "bottom"]}
      testID="report-screen"
      footer={
        <Button
          label="Envoyer le signalement"
          testID="report-submit"
          disabled={!reason}
          loading={report.isPending}
          onPress={() => report.mutate()}
        />
      }
    >
      <View style={styles.body}>
        <Text variant="title">Signaler {route.params.firstName}</Text>
        <Text tone="muted">Ton signalement est confidentiel. Nous bloquons aussi ce membre pour toi.</Text>
        <View style={styles.reasons} accessibilityRole="radiogroup">
          {REASONS.map((r) => {
            const selected = reason === r.value;
            return (
              <Pressable
                key={r.value}
                testID={`report-reason-${r.value}`}
                accessibilityRole="radio"
                aria-checked={selected}
                onPress={() => setReason(r.value)}
                style={[styles.reason, selected ? styles.reasonSelected : null]}
              >
                <Ionicons
                  name={selected ? "radio-button-on" : "radio-button-off"}
                  size={20}
                  color={selected ? colors.accent : colors.textMuted}
                />
                <Text>{r.label}</Text>
              </Pressable>
            );
          })}
        </View>
        <TextField
          label="Détails (facultatif)"
          value={details}
          onChangeText={setDetails}
          multiline
          maxLength={1000}
          counter={{ max: 1000 }}
          placeholder="Décris ce qui s'est passé"
        />
        {report.isError ? <Text tone="danger">{errorMessage(report.error)}</Text> : null}
        <Text variant="caption" tone="subtle">
          En cas de danger immédiat, contacte les services d’urgence (112).
        </Text>
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  body: {
    gap: spacing.lg,
    paddingTop: spacing.md
  },
  reasons: {
    gap: spacing.sm
  },
  reason: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    padding: spacing.md,
    borderRadius: radii.md,
    borderWidth: 1,
    borderColor: colors.border,
    backgroundColor: colors.surface
  },
  reasonSelected: {
    borderColor: colors.accent,
    backgroundColor: colors.accentSoft
  }
});
