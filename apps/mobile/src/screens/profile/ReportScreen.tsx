import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { safetyApi } from "../../api/dating";
import { reportReasonLabels, type ReportReason } from "../../api/types";
import { keys } from "../../hooks/queries";
import type { AppStackParamList } from "../../navigation/types";
import { showToast } from "../../shared/feedback";
import { Screen, ScreenHeader } from "../../shared/layout";
import { Button, Chip, Input, Text, spacing } from "../../shared/ui";
import { errorMessage } from "../../utils/errors";

type Props = NativeStackScreenProps<AppStackParamList, "Report">;

const REASONS = Object.keys(reportReasonLabels) as ReportReason[];

export default function ReportScreen({ navigation, route }: Props) {
  const { firstName, userId } = route.params;
  const client = useQueryClient();
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");
  const report = useMutation({
    mutationFn: () => safetyApi.report(userId, reason as ReportReason, details),
    onSuccess: () => {
      void client.invalidateQueries({ queryKey: keys.matches });
      void client.invalidateQueries({ queryKey: keys.deck });
      showToast("Merci. Notre équipe va examiner ce signalement.", "success");
      navigation.popToTop();
    }
  });

  return (
    <Screen testID="report-screen">
      <ScreenHeader
        subtitle={`Ton signalement est confidentiel. ${firstName} sera aussi bloqué·e et ne pourra plus te contacter.`}
        title={`Signaler ${firstName}`}
      />
      <View style={styles.reasons}>
        {REASONS.map((r) => (
          <Chip key={r} label={reportReasonLabels[r]} onPress={() => setReason(r)} selected={reason === r} testID={`report-${r}`} />
        ))}
      </View>
      <Input
        helperText={`${details.length}/1000`}
        label="Détails (facultatif)"
        maxLength={1000}
        multiline
        onChangeText={setDetails}
        placeholder="Ce qui s'est passé…"
        value={details}
      />
      <Text tone="subtle" variant="caption">
        En cas de danger immédiat, contacte les services d&apos;urgence (112).
      </Text>
      {report.isError ? <Text tone="danger">{errorMessage(report.error)}</Text> : null}
      <Button
        disabled={!reason}
        label="Envoyer le signalement"
        loading={report.isPending}
        onPress={() => report.mutate()}
        testID="report-submit"
        variant="danger"
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  reasons: {
    flexDirection: "row",
    flexWrap: "wrap",
    gap: spacing.sm
  }
});
