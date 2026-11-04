import React, { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { errorMessage } from "../../api/client";
import { safetyApi } from "../../api/endpoints";
import type { ReportReason } from "../../api/types";
import { Button, Chip, Input, Sheet, Text, spacing, useToast } from "../../shared/ui";
import { View } from "react-native";
import { REPORT_REASONS } from "../../utils/labels";

type Props = { userId: string; visible: boolean; onClose: () => void; onReported: () => void };

export function ReportSheet({ userId, visible, onClose, onReported }: Props) {
  const toast = useToast();
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");
  const [alsoBlock, setAlsoBlock] = useState(true);

  const send = useMutation({
    mutationFn: () => safetyApi.report(userId, reason ?? "other", details.trim(), alsoBlock),
    onSuccess: () => {
      toast.show("Signalement envoyé. Merci.", "success");
      setReason(null);
      setDetails("");
      onReported();
    },
    onError: (err) => toast.show(errorMessage(err), "error"),
  });

  return (
    <Sheet visible={visible} title="Signaler ce profil" onClose={onClose} testID="report-sheet">
      <Text tone="muted">Votre signalement est confidentiel.</Text>
      <View style={{ flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }}>
        {REPORT_REASONS.map((r) => <Chip key={r.value} label={r.label} selected={reason === r.value} onPress={() => setReason(r.value)} />)}
      </View>
      <Input label="Détails (facultatif)" value={details} onChangeText={setDetails} multiline multilineHeight={80} maxLength={1000} />
      <Chip label="Bloquer aussi cette personne" selected={alsoBlock} onPress={() => setAlsoBlock((v) => !v)} />
      <Button title="Envoyer le signalement" onPress={() => send.mutate()} disabled={!reason} loading={send.isPending} testID="report-send" />
    </Sheet>
  );
}
