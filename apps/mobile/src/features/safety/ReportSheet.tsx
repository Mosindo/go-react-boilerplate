import React, { useEffect, useState } from "react";
import { StyleSheet, View } from "react-native";
import { REPORT_REASONS, type ReportReason } from "../../api/types";
import { spacing } from "../../theme/palette";
import { Button, Chip, Sheet, Text, TextField, ToggleRow } from "../../ui";

type Props = {
  visible: boolean;
  name: string;
  loading?: boolean;
  onClose: () => void;
  onSubmit: (input: { reason: ReportReason; details: string; block: boolean }) => void;
};

export function ReportSheet({ visible, name, loading, onClose, onSubmit }: Props) {
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");
  const [block, setBlock] = useState(true);

  useEffect(() => {
    if (visible) {
      setReason(null);
      setDetails("");
      setBlock(true);
    }
  }, [visible]);

  return (
    <Sheet visible={visible} title={`Signaler ${name}`} onClose={onClose}>
      <Text tone="muted">Ton signalement est confidentiel : {name} ne saura pas qui l’a envoyé.</Text>
      <View style={styles.reasons}>
        {REPORT_REASONS.map((r) => (
          <Chip key={r.value} label={r.label} selected={reason === r.value} onPress={() => setReason(r.value)} testID={`report-${r.value}`} />
        ))}
      </View>
      <TextField label="Détails (facultatif)" value={details} onChangeText={setDetails} multiline maxLength={1000} showCounter />
      <ToggleRow title={`Bloquer ${name}`} subtitle="Recommandé : vous ne vous verrez plus." value={block} onChange={setBlock} />
      <Button
        label="Envoyer le signalement"
        variant="danger"
        size="lg"
        disabled={!reason}
        loading={loading}
        onPress={() => reason && onSubmit({ reason, details: details.trim(), block })}
        testID="report-submit"
      />
    </Sheet>
  );
}

const styles = StyleSheet.create({
  reasons: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs }
});
