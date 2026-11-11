import React, { useState } from "react";
import { View } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { safetyApi } from "../../api/endpoints";
import type { ReportReason } from "../../api/types";
import { keys } from "../../hooks/queries";
import type { AppStackParamList } from "../../navigation/types";
import { Button, Chip, Screen, Text, TextField, useFeedback } from "../../ui";
import { reportReasonLabels } from "../../utils/format";

type Props = NativeStackScreenProps<AppStackParamList, "Report">;

export function ReportScreen({ route, navigation }: Props) {
  const { userId, firstName } = route.params;
  const qc = useQueryClient();
  const { toast } = useFeedback();
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");

  const report = useMutation({
    mutationFn: () => safetyApi.report(userId, reason as ReportReason, details.trim()),
    onSuccess: () => {
      void qc.invalidateQueries({ queryKey: keys.conversations });
      void qc.invalidateQueries({ queryKey: keys.discover });
      toast("Merci. Le signalement a été transmis et le profil est bloqué.", "success");
      navigation.popToTop();
    }
  });

  return (
    <Screen
      edges={[]}
      testID="report-screen"
      footer={
        <Button
          testID="report-submit"
          label="Envoyer le signalement"
          variant="danger"
          disabled={!reason}
          loading={report.isPending}
          onPress={() => report.mutate()}
        />
      }
    >
      <View style={{ gap: 6 }}>
        <Text variant="title">Signaler {firstName}</Text>
        <Text tone="muted">
          Ton signalement est confidentiel. {firstName} sera bloqué·e automatiquement et ne pourra plus te contacter.
        </Text>
      </View>
      <View style={{ gap: 8 }}>
        {(Object.keys(reportReasonLabels) as ReportReason[]).map((r) => (
          <Chip
            key={r}
            testID={`report-reason-${r}`}
            label={reportReasonLabels[r]}
            selected={reason === r}
            onPress={() => setReason(r)}
          />
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
      {report.isError ? <Text tone="danger">{errorMessage(report.error)}</Text> : null}
    </Screen>
  );
}
