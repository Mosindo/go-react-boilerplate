import { Ionicons } from "@expo/vector-icons";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import React, { useState } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { safetyApi } from "../../api/safety";
import type { ReportReason } from "../../api/types";
import type { RootScreenProps } from "../../navigation/types";
import { queryKeys } from "../../state/queryClient";
import { useTheme } from "../../theme/ThemeProvider";
import { Button, Header, Screen, Text, TextField, useToast } from "../../ui";
import { describeError } from "../../utils/errors";
import { reportReasonLabels } from "../../utils/format";

export default function ReportScreen({ navigation, route }: RootScreenProps<"Report">) {
  const { userId, firstName } = route.params;
  const { colors, spacing, radii } = useTheme();
  const toast = useToast();
  const client = useQueryClient();
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");

  const report = useMutation({
    mutationFn: () => safetyApi.report(userId, reason as ReportReason, details.trim()),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: queryKeys.conversations });
      client.removeQueries({ queryKey: queryKeys.messages(userId) });
      toast.show("Merci. Le signalement est transmis et ce membre est bloqué.", "success");
      navigation.popToTop();
    }
  });

  return (
    <Screen
      scroll
      testID="report-screen"
      footer={
        <Button
          label="Envoyer le signalement"
          variant="danger"
          disabled={!reason}
          loading={report.isPending}
          onPress={() => report.mutate()}
          testID="report-submit"
        />
      }
    >
      <Header back title={`Signaler ${firstName}`} />
      <View style={{ gap: spacing.md }}>
        <Text tone="muted">
          Votre signalement est confidentiel : {firstName || "ce membre"} ne saura pas qu’il vient de vous. La personne
          sera aussi bloquée pour vous.
        </Text>
        <View style={{ gap: spacing.xs }} accessibilityRole="radiogroup">
          {(Object.keys(reportReasonLabels) as ReportReason[]).map((r) => (
            <Pressable
              key={r}
              accessibilityRole="radio"
              accessibilityState={{ checked: reason === r }}
              onPress={() => setReason(r)}
              style={[
                styles.option,
                {
                  borderColor: reason === r ? colors.primary : colors.border,
                  backgroundColor: colors.surface,
                  borderRadius: radii.md
                }
              ]}
              testID={`report-reason-${r}`}
            >
              <Ionicons
                name={reason === r ? "radio-button-on" : "radio-button-off"}
                size={22}
                color={reason === r ? colors.primary : colors.textMuted}
              />
              <Text style={styles.flex}>{reportReasonLabels[r]}</Text>
            </Pressable>
          ))}
        </View>
        <TextField
          label="Détails (facultatif)"
          value={details}
          onChangeText={setDetails}
          multiline
          maxLength={1000}
          hint={`${details.length}/1000`}
        />
        {reason === "underage" ? (
          <Text variant="label" tone="accent">
            Le profil sera immédiatement masqué en attendant la vérification.
          </Text>
        ) : null}
        {report.error ? <Text tone="danger">{describeError(report.error)}</Text> : null}
      </View>
    </Screen>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  option: { flexDirection: "row", alignItems: "center", gap: 12, padding: 14, borderWidth: 1.5 }
});
