import React, { useState } from "react";
import { View } from "react-native";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { safetyApi } from "../../api/endpoints";
import type { ReportReason } from "../../api/types";
import { errorMessage, reportReasonLabels } from "../../lib/format";
import type { AppStackParamList } from "../../navigation/types";
import { keys } from "../../providers/queryClient";
import { useTheme } from "../../theme";
import { Button, Chip, Header, Screen, showToast, Text, TextField } from "../../ui";
import { swipeStore } from "../discovery/swipeStore";

type Props = NativeStackScreenProps<AppStackParamList, "Report">;

export function ReportScreen({ navigation, route }: Props) {
  const { userId, firstName } = route.params;
  const { spacing } = useTheme();
  const queryClient = useQueryClient();
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");
  const report = useMutation({
    mutationFn: () => safetyApi.report(userId, reason as ReportReason, details.trim()),
    onSuccess: () => {
      swipeStore.add(userId);
      void queryClient.invalidateQueries({ queryKey: keys.conversationsRoot });
      void queryClient.invalidateQueries({ queryKey: keys.likes });
      showToast("Merci. Ton signalement a été transmis et cette personne est bloquée.", "success");
      navigation.popToTop();
    }
  });

  return (
    <Screen keyboard scroll header={<Header back title={`Signaler ${firstName}`} />}>
      <Text tone="muted">
        Les signalements sont confidentiels : {firstName} ne saura pas que c’est toi. La personne sera aussi bloquée pour toi. En cas de danger immédiat,
        contacte le 17 (ou le 112).
      </Text>
      <View style={{ flexDirection: "row", flexWrap: "wrap", gap: 8 }}>
        {(Object.keys(reportReasonLabels) as ReportReason[]).map((r) => (
          <Chip key={r} label={reportReasonLabels[r]} onPress={() => setReason(r)} selected={reason === r} testID={`report-${r}`} />
        ))}
      </View>
      <TextField
        counter={1000}
        label="Détails (facultatif)"
        maxLength={1000}
        multiline
        onChangeText={setDetails}
        placeholder="Ce qui s’est passé…"
        value={details}
      />
      {report.isError ? <Text tone="danger">{errorMessage(report.error)}</Text> : null}
      <View style={{ gap: spacing.sm }}>
        <Button
          disabled={!reason}
          label="Envoyer le signalement"
          loading={report.isPending}
          onPress={() => report.mutate()}
          testID="report-submit"
          variant="danger"
        />
      </View>
    </Screen>
  );
}
