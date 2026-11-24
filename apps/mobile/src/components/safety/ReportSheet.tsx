import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { safetyApi } from "../../api/endpoints";
import type { ReportReason } from "../../api/types";
import { removeFromDeck } from "../../hooks/useSwipe";
import { queryKeys } from "../../queryClient";
import { colors, spacing } from "../../theme/tokens";
import { errorMessage } from "../../utils/errors";
import { reportReasonLabels } from "../../utils/format";
import { AppText } from "../AppText";
import { Button } from "../Button";
import { Chip } from "../Chip";
import { Sheet } from "../Sheet";
import { TextField } from "../TextField";
import { showToast } from "../Toast";

const reasons = Object.keys(reportReasonLabels) as ReportReason[];

type Props = { visible: boolean; userId: string; name: string; onClose: () => void; onDone: () => void };

/** Report flow. Reporting always blocks the member as well. */
export function ReportSheet({ visible, userId, name, onClose, onDone }: Props) {
  const client = useQueryClient();
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const submit = async () => {
    if (!reason) {
      setError("Choisis un motif.");
      return;
    }
    setBusy(true);
    setError(null);
    try {
      await safetyApi.report(userId, reason, details.trim());
      removeFromDeck(client, userId);
      void client.invalidateQueries({ queryKey: queryKeys.conversations });
      void client.invalidateQueries({ queryKey: queryKeys.blocks });
      showToast(`Merci. ${name} a été signalé·e et bloqué·e.`, "success");
      onDone();
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Sheet visible={visible} title={`Signaler ${name}`} onClose={onClose}>
      <AppText color={colors.textMuted}>Ton signalement est confidentiel. {name} sera aussi bloqué·e : vous ne vous verrez plus.</AppText>
      <View style={styles.chips}>
        {reasons.map((r) => (
          <Chip key={r} label={reportReasonLabels[r]} selected={reason === r} onPress={() => setReason(r)} testID={`report-reason-${r}`} />
        ))}
      </View>
      <TextField label="Détails (facultatif)" value={details} onChangeText={setDetails} multiline maxLength={1000} counter={1000} />
      {error ? <AppText color={colors.danger}>{error}</AppText> : null}
      <Button label="Envoyer le signalement" variant="danger" onPress={submit} loading={busy} testID="report-submit" />
    </Sheet>
  );
}

const styles = StyleSheet.create({
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs }
});
