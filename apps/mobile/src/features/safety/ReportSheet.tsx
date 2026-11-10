import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { errorMessage } from "../../api/client";
import { safetyApi } from "../../api/endpoints";
import type { ReportReason } from "../../api/types";
import { Button, Chip, Sheet, Text, TextField, showToast } from "../../ui";
import { reportReasonLabels } from "../../utils/format";

type Props = {
  userId: string;
  name: string;
  visible: boolean;
  onClose: () => void;
  onDone: () => void;
};

const REASONS = Object.keys(reportReasonLabels) as ReportReason[];

export function ReportSheet({ userId, name, visible, onClose, onDone }: Props) {
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");
  const [loading, setLoading] = useState(false);

  const submit = async () => {
    if (!reason) return;
    setLoading(true);
    try {
      await safetyApi.report(userId, reason, details.trim());
      showToast(`Merci. ${name} est bloqué·e et notre équipe va examiner le signalement.`, "success");
      setReason(null);
      setDetails("");
      onDone();
    } catch (e) {
      showToast(errorMessage(e), "error");
    } finally {
      setLoading(false);
    }
  };

  return (
    <Sheet onClose={onClose} title={`Signaler ${name}`} visible={visible}>
      <View style={styles.body}>
        <Text tone="muted" variant="small">
          Votre signalement est confidentiel. {name} sera aussi bloqué·e : vous ne vous verrez plus.
        </Text>
        <View style={styles.chips}>
          {REASONS.map((r) => (
            <Chip key={r} label={reportReasonLabels[r]} onPress={() => setReason(r)} selected={reason === r} testID={`report-${r}`} />
          ))}
        </View>
        <TextField counter label="Détails (facultatif)" maxLength={1000} multiline onChangeText={setDetails} value={details} />
        <Button disabled={!reason} label="Envoyer le signalement" loading={loading} onPress={submit} testID="report-submit" variant="danger" />
      </View>
    </Sheet>
  );
}

const styles = StyleSheet.create({
  body: { gap: 16, paddingBottom: 8 },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: 8 }
});
