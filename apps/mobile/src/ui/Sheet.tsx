import React from "react";
import { KeyboardAvoidingView, Modal, Platform, Pressable, StyleSheet, View } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { radii, spacing, useTheme } from "../theme";
import { Button } from "./Button";
import { Text } from "./Text";

type Props = {
  visible: boolean;
  onClose: () => void;
  title?: string;
  children: React.ReactNode;
};

/** Bottom sheet built on the platform Modal (no extra dependency). */
export function Sheet({ visible, onClose, title, children }: Props) {
  const { colors } = useTheme();
  const insets = useSafeAreaInsets();
  return (
    <Modal visible={visible} transparent animationType="slide" onRequestClose={onClose} statusBarTranslucent>
      <KeyboardAvoidingView style={styles.flex} behavior={Platform.OS === "ios" ? "padding" : undefined}>
        <Pressable style={[styles.backdrop, { backgroundColor: colors.scrim }]} onPress={onClose} accessibilityLabel="Fermer" />
        <View style={[styles.sheet, { backgroundColor: colors.surface, paddingBottom: insets.bottom + spacing.md }]}>
          <View style={[styles.grabber, { backgroundColor: colors.border }]} />
          {title ? (
            <Text variant="heading" style={styles.title} accessibilityRole="header">
              {title}
            </Text>
          ) : null}
          {children}
        </View>
      </KeyboardAvoidingView>
    </Modal>
  );
}

type Action = { label: string; onPress: () => void; destructive?: boolean; testID?: string };

export function ActionSheet({
  visible,
  onClose,
  title,
  actions
}: {
  visible: boolean;
  onClose: () => void;
  title?: string;
  actions: Action[];
}) {
  const { colors } = useTheme();
  return (
    <Sheet visible={visible} onClose={onClose} title={title}>
      {actions.map((action) => (
        <Pressable
          key={action.label}
          accessibilityRole="button"
          testID={action.testID}
          onPress={() => {
            onClose();
            action.onPress();
          }}
          style={({ pressed }) => [styles.action, { borderBottomColor: colors.border }, pressed && { opacity: 0.6 }]}
        >
          <Text variant="label" color={action.destructive ? "danger" : "text"} style={styles.actionText}>
            {action.label}
          </Text>
        </Pressable>
      ))}
      <Pressable accessibilityRole="button" onPress={onClose} style={styles.action}>
        <Text variant="label" color="textMuted" style={styles.actionText}>
          Annuler
        </Text>
      </Pressable>
    </Sheet>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1, justifyContent: "flex-end" },
  backdrop: { ...StyleSheet.absoluteFillObject },
  sheet: {
    borderTopLeftRadius: radii.xl,
    borderTopRightRadius: radii.xl,
    paddingHorizontal: spacing.lg,
    paddingTop: spacing.sm,
    maxHeight: "88%",
    width: "100%",
    maxWidth: 640,
    alignSelf: "center"
  },
  grabber: { width: 40, height: 5, borderRadius: 3, alignSelf: "center", marginBottom: spacing.md },
  title: { marginBottom: spacing.md },
  action: { paddingVertical: spacing.md, borderBottomWidth: StyleSheet.hairlineWidth },
  actionText: { fontSize: 16, textAlign: "center" }
});

type ConfirmProps = {
  visible: boolean;
  title: string;
  message: string;
  confirmLabel: string;
  destructive?: boolean;
  loading?: boolean;
  onConfirm: () => void;
  onClose: () => void;
};

/** Cross-platform confirmation dialog (Alert.alert has no buttons on web). */
export function ConfirmSheet({ visible, title, message, confirmLabel, destructive, loading, onConfirm, onClose }: ConfirmProps) {
  return (
    <Sheet visible={visible} onClose={onClose} title={title}>
      <Text color="textMuted" style={{ marginBottom: spacing.lg }}>
        {message}
      </Text>
      <View style={{ gap: spacing.sm }}>
        <Button
          label={confirmLabel}
          variant={destructive ? "danger" : "primary"}
          loading={loading}
          onPress={onConfirm}
          testID="confirm-action"
        />
        <Button label="Annuler" variant="ghost" onPress={onClose} />
      </View>
    </Sheet>
  );
}
