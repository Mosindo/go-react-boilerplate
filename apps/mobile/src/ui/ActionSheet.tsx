import React from "react";
import { Modal, Pressable } from "react-native";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { makeStyles } from "../theme";
import { Icon, type IconName } from "./Icon";
import { modalAnimation } from "./modal";
import { Text } from "./Text";

export type SheetAction = { label: string; icon?: IconName; destructive?: boolean; onPress: () => void; testID?: string };

type ActionSheetProps = {
  visible: boolean;
  title?: string;
  actions: SheetAction[];
  onClose: () => void;
};

export function ActionSheet({ visible, title, actions, onClose }: ActionSheetProps) {
  const styles = useStyles();
  const insets = useSafeAreaInsets();
  return (
    <Modal visible={visible} transparent animationType={modalAnimation("fade")} onRequestClose={onClose}>
      <Pressable style={styles.backdrop} onPress={onClose} accessibilityLabel="Fermer" accessibilityRole="button">
        <Pressable style={[styles.sheet, { paddingBottom: insets.bottom + 12 }]} onPress={() => undefined}>
          {title ? (
            <Text variant="label" tone="muted" align="center" style={styles.title}>
              {title}
            </Text>
          ) : null}
          {actions.map((action) => (
            <Pressable
              key={action.label}
              testID={action.testID}
              accessibilityRole="button"
              accessibilityLabel={action.label}
              style={({ pressed }) => [styles.action, pressed ? styles.pressed : null]}
              onPress={() => {
                onClose();
                action.onPress();
              }}
            >
              {action.icon ? <Icon name={action.icon} size={20} color={action.destructive ? "danger" : "text"} /> : null}
              <Text variant="bodyStrong" tone={action.destructive ? "danger" : "default"}>
                {action.label}
              </Text>
            </Pressable>
          ))}
          <Pressable accessibilityRole="button" style={[styles.action, styles.cancel]} onPress={onClose}>
            <Text variant="bodyStrong" tone="muted">
              Annuler
            </Text>
          </Pressable>
        </Pressable>
      </Pressable>
    </Modal>
  );
}

const useStyles = makeStyles((t) => ({
  backdrop: { flex: 1, backgroundColor: t.colors.overlay, justifyContent: "flex-end", alignItems: "center" },
  sheet: {
    width: "100%",
    maxWidth: 560,
    backgroundColor: t.colors.surfaceRaised,
    borderTopLeftRadius: t.radii.xl,
    borderTopRightRadius: t.radii.xl,
    paddingTop: t.spacing.lg,
    paddingHorizontal: t.spacing.lg
  },
  title: { marginBottom: t.spacing.sm },
  action: {
    flexDirection: "row",
    alignItems: "center",
    gap: t.spacing.md,
    paddingVertical: t.spacing.lg,
    paddingHorizontal: t.spacing.md,
    borderRadius: t.radii.md
  },
  pressed: { backgroundColor: t.colors.surfaceAlt },
  cancel: { justifyContent: "center", marginTop: t.spacing.xs }
}));
