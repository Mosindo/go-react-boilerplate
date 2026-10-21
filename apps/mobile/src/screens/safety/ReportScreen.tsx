import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useQueryClient } from "@tanstack/react-query";
import React, { useState } from "react";
import { StyleSheet, Switch, View } from "react-native";
import { reportUser } from "../../api/safety";
import { keys } from "../../hooks/queries";
import { REPORT_REASONS } from "../../lib/labels";
import type { RootStackParamList } from "../../navigation/types";
import { showToast } from "../../shared/feedback";
import { ScreenContainer } from "../../shared/layout";
import { Button, Chip, FormField, Input, Notice, Text } from "../../shared/ui";
import { spacing, useTheme } from "../../theme";

type Props = NativeStackScreenProps<RootStackParamList, "Report">;

export function ReportScreen({ navigation, route }: Props) {
  const { userId, name } = route.params;
  const theme = useTheme();
  const queryClient = useQueryClient();
  const [reason, setReason] = useState<string | null>(null);
  const [details, setDetails] = useState("");
  const [alsoBlock, setAlsoBlock] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);

  const submit = async () => {
    if (!reason) return setError("Choisissez un motif.");
    setBusy(true);
    setError(null);
    try {
      await reportUser(userId, reason, details.trim(), alsoBlock);
      void queryClient.invalidateQueries({ queryKey: keys.conversations });
      void queryClient.invalidateQueries({ queryKey: keys.discover });
      showToast("Merci, votre signalement a été transmis à notre équipe.", "success");
      navigation.popToTop();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Envoi impossible.");
      setBusy(false);
    }
  };

  return (
    <ScreenContainer scroll edges={["bottom", "left", "right"]} testID="report-screen">
      <Text tone="muted">Pourquoi signalez-vous {name} ? Le signalement est confidentiel.</Text>
      {error ? <Notice message={error} /> : null}
      <FormField label="Motif">
        <View style={styles.chips}>
          {REPORT_REASONS.map((r) => (
            <Chip
              key={r.value}
              testID={`reason-${r.value}`}
              label={r.label}
              selected={reason === r.value}
              onPress={() => setReason(r.value)}
            />
          ))}
        </View>
      </FormField>
      <FormField label="Détails (facultatif)" hint={`${details.length}/1000`}>
        <Input
          testID="report-details"
          value={details}
          onChangeText={setDetails}
          multiline
          maxLength={1000}
          placeholder="Que s'est-il passé ?"
        />
      </FormField>
      <View style={styles.row}>
        <Text style={styles.flex}>Bloquer aussi {name}</Text>
        <Switch
          testID="report-also-block"
          value={alsoBlock}
          onValueChange={setAlsoBlock}
          trackColor={{ true: theme.primary, false: theme.border }}
          accessibilityLabel={`Bloquer aussi ${name}`}
        />
      </View>
      <Button testID="report-submit" label="Envoyer le signalement" loading={busy} onPress={() => void submit()} />
    </ScreenContainer>
  );
}

const styles = StyleSheet.create({
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.md },
  flex: { flex: 1 }
});
