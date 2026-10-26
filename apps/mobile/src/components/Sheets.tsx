import React, { useState } from "react";
import { Modal, Pressable, StyleSheet, View } from "react-native";
import type { ReportReason } from "../api";
import { radius, spacing, useTheme } from "../theme";
import { Button } from "./Button";
import { Chip } from "./Chip";
import { Text } from "./Text";
import { TextField } from "./TextField";

export type SheetAction = { label: string; onPress: () => void; destructive?: boolean; testID?: string };

/** Bottom sheet listing a few actions (profile/chat menus). */
export function ActionSheet({
  visible,
  title,
  actions,
  onClose
}: {
  visible: boolean;
  title?: string;
  actions: SheetAction[];
  onClose: () => void;
}) {
  const theme = useTheme();
  return (
    <Modal visible={visible} transparent animationType="slide" onRequestClose={onClose}>
      <Pressable
        style={[styles.backdrop, { backgroundColor: theme.overlay }]}
        onPress={onClose}
        accessibilityLabel="Fermer le menu"
      >
        <Pressable
          style={[styles.sheet, { backgroundColor: theme.surface }]}
          onPress={() => undefined}
          accessibilityViewIsModal
        >
          {title ? <Text variant="heading">{title}</Text> : null}
          {actions.map((a) => (
            <Pressable
              key={a.label}
              testID={a.testID}
              accessibilityRole="button"
              onPress={() => {
                onClose();
                a.onPress();
              }}
              style={({ pressed }) => [
                styles.action,
                { backgroundColor: pressed ? theme.surfaceAlt : "transparent" }
              ]}
            >
              <Text variant="bodyStrong" color={a.destructive ? theme.danger : theme.text}>
                {a.label}
              </Text>
            </Pressable>
          ))}
          <Button label="Annuler" variant="secondary" small onPress={onClose} />
        </Pressable>
      </Pressable>
    </Modal>
  );
}

const REASONS: { value: ReportReason; label: string }[] = [
  { value: "fake", label: "Faux profil" },
  { value: "harassment", label: "Harcèlement" },
  { value: "inappropriate", label: "Contenu inapproprié" },
  { value: "spam", label: "Spam ou arnaque" },
  { value: "underage", label: "Personne mineure" },
  { value: "other", label: "Autre" }
];

export function ReportSheet({
  visible,
  name,
  onClose,
  onSubmit
}: {
  visible: boolean;
  name: string;
  onClose: () => void;
  onSubmit: (reason: ReportReason, details: string) => Promise<void>;
}) {
  const theme = useTheme();
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const submit = async () => {
    if (!reason) return setError("Choisissez un motif.");
    setLoading(true);
    setError(null);
    try {
      await onSubmit(reason, details.trim());
      setReason(null);
      setDetails("");
      onClose();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Signalement impossible.");
    } finally {
      setLoading(false);
    }
  };

  return (
    <Modal visible={visible} transparent animationType="slide" onRequestClose={onClose}>
      <Pressable style={[styles.backdrop, { backgroundColor: theme.overlay }]} onPress={onClose}>
        <Pressable
          style={[styles.sheet, { backgroundColor: theme.surface }]}
          onPress={() => undefined}
          accessibilityViewIsModal
        >
          <Text variant="heading">Signaler {name}</Text>
          <Text muted>
            Votre signalement est confidentiel. {name} ne saura pas que vous l&apos;avez signalé(e).
          </Text>
          <View style={styles.reasons}>
            {REASONS.map((r) => (
              <Chip
                key={r.value}
                label={r.label}
                selected={reason === r.value}
                onPress={() => setReason(r.value)}
                testID={`reason-${r.value}`}
              />
            ))}
          </View>
          <TextField
            label="Détails (facultatif)"
            value={details}
            onChangeText={setDetails}
            multiline
            maxLength={1000}
          />
          {error ? (
            <Text color={theme.danger} accessibilityRole="alert">
              {error}
            </Text>
          ) : null}
          <View style={styles.footer}>
            <Button label="Annuler" variant="ghost" small onPress={onClose} />
            <Button
              label="Envoyer le signalement"
              small
              loading={loading}
              onPress={() => void submit()}
              testID="send-report"
            />
          </View>
        </Pressable>
      </Pressable>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, justifyContent: "flex-end" },
  sheet: {
    borderTopLeftRadius: radius.lg,
    borderTopRightRadius: radius.lg,
    padding: spacing.xl,
    gap: spacing.md,
    width: "100%",
    maxWidth: 560,
    alignSelf: "center"
  },
  action: { paddingVertical: spacing.md, paddingHorizontal: spacing.sm, borderRadius: radius.md },
  reasons: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  footer: { flexDirection: "row", justifyContent: "flex-end", gap: spacing.sm }
});
