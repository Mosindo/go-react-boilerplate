import React, { useState, type ReactNode } from "react";
import { Modal, Pressable, StyleSheet, View } from "react-native";
import type { ReportReason } from "../api/types";
import { Button, Chip, Input, Text, colors, radii, spacing } from "../shared/ui";

// Alert.alert is a no-op on the web, so confirmations are in-app modals that behave the same everywhere.

type SheetOption = { label: string; onPress: () => void; destructive?: boolean; testID?: string };

type ActionSheetProps = { visible: boolean; title?: string; options: SheetOption[]; onClose: () => void };

export function ActionSheet({ visible, title, options, onClose }: ActionSheetProps) {
  return (
    <Modal animationType="fade" onRequestClose={onClose} transparent visible={visible}>
      <Pressable accessibilityLabel="Fermer le menu" onPress={onClose} style={styles.backdrop}>
        <Pressable style={styles.sheet}>
          {title ? (
            <Text tone="muted" variant="label" weight="semibold">
              {title}
            </Text>
          ) : null}
          {options.map((option) => (
            <Button
              fullWidth
              key={option.label}
              label={option.label}
              onPress={() => {
                onClose();
                option.onPress();
              }}
              testID={option.testID}
              variant={option.destructive ? "outline" : "secondary"}
              textStyle={option.destructive ? styles.destructiveText : undefined}
            />
          ))}
          <Button fullWidth label="Annuler" onPress={onClose} variant="ghost" />
        </Pressable>
      </Pressable>
    </Modal>
  );
}

type ConfirmDialogProps = {
  visible: boolean;
  title: string;
  message: string;
  confirmLabel: string;
  destructive?: boolean;
  loading?: boolean;
  error?: string | null;
  children?: ReactNode;
  onConfirm: () => void;
  onCancel: () => void;
};

export function ConfirmDialog({
  visible,
  title,
  message,
  confirmLabel,
  destructive,
  loading,
  error,
  children,
  onConfirm,
  onCancel
}: ConfirmDialogProps) {
  return (
    <Modal animationType="fade" onRequestClose={onCancel} transparent visible={visible}>
      <View style={styles.centered}>
        <View accessibilityViewIsModal style={styles.dialog}>
          <Text variant="heading" weight="bold">
            {title}
          </Text>
          <Text tone="muted">{message}</Text>
          {children}
          {error ? (
            <Text tone="danger" variant="label" weight="medium">
              {error}
            </Text>
          ) : null}
          <View style={styles.row}>
            <Button label="Annuler" onPress={onCancel} variant="ghost" />
            <Button
              label={confirmLabel}
              loading={loading}
              onPress={onConfirm}
              testID="confirm-dialog-confirm"
              variant={destructive ? "destructive" : "primary"}
            />
          </View>
        </View>
      </View>
    </Modal>
  );
}

const REASONS: { value: ReportReason; label: string }[] = [
  { value: "fake_profile", label: "Faux profil" },
  { value: "harassment", label: "Harcèlement" },
  { value: "inappropriate_content", label: "Contenu inapproprié" },
  { value: "spam", label: "Spam" },
  { value: "underage", label: "Personne mineure" },
  { value: "other", label: "Autre" }
];

type ReportDialogProps = {
  visible: boolean;
  name: string;
  loading?: boolean;
  error?: string | null;
  onSubmit: (reason: ReportReason, details: string, alsoBlock: boolean) => void;
  onCancel: () => void;
};

export function ReportDialog({ visible, name, loading, error, onSubmit, onCancel }: ReportDialogProps) {
  const [reason, setReason] = useState<ReportReason | null>(null);
  const [details, setDetails] = useState("");
  const [alsoBlock, setAlsoBlock] = useState(true);

  return (
    <Modal animationType="fade" onRequestClose={onCancel} transparent visible={visible}>
      <View style={styles.centered}>
        <View accessibilityViewIsModal style={styles.dialog}>
          <Text variant="heading" weight="bold">
            Signaler {name}
          </Text>
          <Text tone="muted">Votre signalement reste confidentiel. {name} ne sera pas prévenu(e).</Text>
          <View style={styles.chips}>
            {REASONS.map((item) => (
              <Chip
                key={item.value}
                label={item.label}
                onPress={() => setReason(item.value)}
                selected={reason === item.value}
                testID={`report-${item.value}`}
              />
            ))}
          </View>
          <Input maxLength={1000} multiline onChangeText={setDetails} placeholder="Précisions (facultatif)" value={details} />
          <Chip
            label={alsoBlock ? "✓ Bloquer aussi cette personne" : "Bloquer aussi cette personne"}
            onPress={() => setAlsoBlock((value) => !value)}
            selected={alsoBlock}
          />
          {error ? (
            <Text tone="danger" variant="label" weight="medium">
              {error}
            </Text>
          ) : null}
          <View style={styles.row}>
            <Button label="Annuler" onPress={onCancel} variant="ghost" />
            <Button
              disabled={!reason}
              label="Envoyer"
              loading={loading}
              onPress={() => reason && onSubmit(reason, details.trim(), alsoBlock)}
              testID="report-submit"
              variant="destructive"
            />
          </View>
        </View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, backgroundColor: colors.overlay, justifyContent: "flex-end" },
  sheet: {
    backgroundColor: colors.backgroundElevated,
    padding: spacing.lg,
    gap: spacing.sm,
    borderTopLeftRadius: radii.xl,
    borderTopRightRadius: radii.xl,
    width: "100%",
    maxWidth: 560,
    alignSelf: "center"
  },
  destructiveText: { color: colors.danger },
  centered: { flex: 1, backgroundColor: colors.overlay, alignItems: "center", justifyContent: "center", padding: spacing.lg },
  dialog: {
    width: "100%",
    maxWidth: 440,
    backgroundColor: colors.backgroundElevated,
    borderRadius: radii.xl,
    padding: spacing.xl,
    gap: spacing.md
  },
  row: { flexDirection: "row", justifyContent: "flex-end", gap: spacing.sm, marginTop: spacing.sm },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs }
});
