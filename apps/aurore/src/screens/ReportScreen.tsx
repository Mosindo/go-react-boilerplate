import React, { useState } from "react";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { moderationApi } from "../api/endpoints";
import { keys } from "../api/keys";
import type { ReportReason } from "../api/types";
import { useFeedback } from "../components/Feedback";
import { errorMessage } from "../components/forms";
import { Button, Chip, ErrorText, Row, Screen, Text, TextField } from "../components/ui";
import { REPORT_REASONS } from "../lib/format";
import type { RootStackParamList } from "../navigation/types";
import { spacing } from "../theme/theme";

type Props = NativeStackScreenProps<RootStackParamList, "Report">;

export function ReportScreen({ navigation, route }: Props) {
  const { userId, name } = route.params;
  const qc = useQueryClient();
  const { toast } = useFeedback();
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");
  const [alsoBlock, setAlsoBlock] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const report = useMutation({
    mutationFn: () => moderationApi.report(userId, reason as ReportReason, details.trim(), alsoBlock),
    onSuccess: () => {
      toast("Merci, votre signalement a été envoyé.", "success");
      void qc.invalidateQueries({ queryKey: keys.matches });
      void qc.invalidateQueries({ queryKey: keys.conversations });
      void qc.invalidateQueries({ queryKey: ["discover"] });
      navigation.popToTop();
    },
    onError: (e) => toast(errorMessage(e), "error")
  });

  const submit = () => {
    if (!reason) return setError("Choisissez un motif.");
    setError(null);
    report.mutate();
  };

  return (
    <Screen scroll edges={["left", "right", "bottom"]}>
      <Text variant="title" style={{ marginTop: spacing.lg, marginBottom: spacing.sm }}>Signaler {name}</Text>
      <Text muted style={{ marginBottom: spacing.xl }}>Votre signalement est confidentiel : la personne ne saura pas que c&apos;est vous.</Text>
      <Row style={{ marginBottom: spacing.sm }}>
        {REPORT_REASONS.map((r) => (
          <Chip key={r.value} label={r.label} selected={reason === r.value} onPress={() => setReason(r.value)} testID={`reason-${r.value}`} />
        ))}
      </Row>
      {error ? <ErrorText style={{ marginBottom: spacing.md }}>{error}</ErrorText> : null}
      <TextField label="Détails (facultatif)" value={details} onChangeText={setDetails} multiline maxLength={1000} hint={`${details.length}/1000`} testID="report-details" />
      <Chip label={alsoBlock ? "✓ Bloquer aussi cette personne" : "Bloquer aussi cette personne"} selected={alsoBlock} onPress={() => setAlsoBlock((b) => !b)} />
      <Button label="Envoyer le signalement" onPress={submit} loading={report.isPending} style={{ marginTop: spacing.xl }} testID="report-submit" />
    </Screen>
  );
}
