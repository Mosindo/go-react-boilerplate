import { useQueryClient } from "@tanstack/react-query";
import React, { useState } from "react";
import { Alert, StyleSheet, Switch, Text, View } from "react-native";
import { blockUser, reportUser } from "../api/safety";
import type { ReportReason } from "../api/types";
import { queryKeys } from "../hooks/queries";
import { useAsyncAction } from "../hooks/useAsyncAction";
import { markProfileResolved } from "../state/discoverStore";
import { useToast } from "../state/ToastProvider";
import { fontSizes, spacing } from "../theme/palette";
import { useTheme } from "../theme/ThemeProvider";
import { Button } from "./Button";
import { Chip } from "./Chip";
import { Sheet } from "./Sheet";
import { TextField } from "./TextField";

const REASONS: { value: ReportReason; label: string }[] = [
  { value: "fake_profile", label: "Faux profil" },
  { value: "harassment", label: "Harcèlement" },
  { value: "inappropriate_content", label: "Contenu inapproprié" },
  { value: "scam", label: "Arnaque" },
  { value: "spam", label: "Spam" },
  { value: "underage", label: "Personne mineure" },
  { value: "other", label: "Autre" }
];

type Props = {
  visible: boolean;
  userId: string;
  firstName: string;
  onClose: () => void;
  /** Called once the user was blocked (and possibly reported). */
  onBlocked: () => void;
  initialMode?: "menu" | "report";
};

export function SafetySheet({
  visible,
  userId,
  firstName,
  onClose,
  onBlocked,
  initialMode = "menu"
}: Props) {
  const theme = useTheme();
  const toast = useToast();
  const queryClient = useQueryClient();
  const [mode, setMode] = useState<"menu" | "report">(initialMode);
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");
  const [alsoBlock, setAlsoBlock] = useState(true);

  const afterBlock = () => {
    markProfileResolved(userId);
    void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
    void queryClient.invalidateQueries({ queryKey: queryKeys.blocked });
    onBlocked();
  };

  const block = useAsyncAction(async () => {
    await blockUser(userId);
    toast.show(`${firstName} est bloqué·e.`, "success");
    afterBlock();
  });

  const report = useAsyncAction(async () => {
    if (!reason) {
      throw new Error("Choisissez un motif.");
    }
    await reportUser(userId, reason, details.trim(), alsoBlock);
    toast.show("Merci, votre signalement a bien été envoyé.", "success");
    if (alsoBlock) {
      afterBlock();
    } else {
      onClose();
    }
  });

  const confirmBlock = () =>
    Alert.alert(
      `Bloquer ${firstName} ?`,
      "Cette personne ne pourra plus vous voir ni vous écrire, et votre conversation sera supprimée.",
      [
        { text: "Annuler", style: "cancel" },
        { text: "Bloquer", style: "destructive", onPress: () => void block.run() }
      ]
    );

  return (
    <Sheet
      onClose={() => {
        setMode(initialMode);
        onClose();
      }}
      title={mode === "menu" ? firstName : `Signaler ${firstName}`}
      visible={visible}
    >
      {mode === "menu" ? (
        <View style={styles.stack}>
          <Button onPress={() => setMode("report")} title="Signaler" variant="secondary" />
          <Button loading={block.loading} onPress={confirmBlock} title="Bloquer" variant="danger" />
          {block.error ? (
            <Text style={[styles.error, { color: theme.danger }]}>{block.error}</Text>
          ) : null}
        </View>
      ) : (
        <View>
          <Text style={[styles.hint, { color: theme.textMuted }]}>
            Votre signalement est confidentiel. Notre équipe l&apos;examinera.
          </Text>
          <View style={styles.chips}>
            {REASONS.map((r) => (
              <Chip
                key={r.value}
                label={r.label}
                onPress={() => setReason(r.value)}
                selected={reason === r.value}
              />
            ))}
          </View>
          <TextField
            label="Détails (facultatif)"
            maxLength={1000}
            multiline
            onChangeText={setDetails}
            value={details}
          />
          <View style={styles.switchRow}>
            <Text style={[styles.switchLabel, { color: theme.text }]}>
              Bloquer aussi cette personne
            </Text>
            <Switch
              onValueChange={setAlsoBlock}
              trackColor={{ true: theme.primary, false: theme.border }}
              value={alsoBlock}
            />
          </View>
          {report.error ? (
            <Text style={[styles.error, { color: theme.danger }]}>{report.error}</Text>
          ) : null}
          <Button
            loading={report.loading}
            onPress={() => void report.run()}
            title="Envoyer le signalement"
          />
        </View>
      )}
    </Sheet>
  );
}

const styles = StyleSheet.create({
  stack: { gap: spacing.sm },
  chips: { flexDirection: "row", flexWrap: "wrap", marginBottom: spacing.md },
  hint: { fontSize: fontSizes.sm, lineHeight: 20, marginBottom: spacing.md },
  switchRow: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    marginBottom: spacing.lg
  },
  switchLabel: { fontSize: fontSizes.md, fontWeight: "600", flex: 1, marginRight: spacing.md },
  error: { fontSize: fontSizes.sm, marginBottom: spacing.sm }
});
